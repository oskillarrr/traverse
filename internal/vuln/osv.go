package vuln

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/oskillarrr/traverse/internal/model"
)

// Limits protecting against hostile or corrupt archives.
var (
	MaxEntryBytes int64 = 16 << 20
	MaxTotalBytes int64 = 8 << 30
	MaxEntries          = 2_000_000
)

// Advisory is the subset of an OSV record Traverse needs.
type Advisory struct {
	ID        string
	Aliases   []string
	Summary   string
	Published string
	Modified  string
	Score     float64
	Vector    string
	Severity  string
	Malicious bool
	Affected  []Affected
}

type Affected struct {
	Ecosystem string
	Name      string // normalized
	Ranges    []Range
	Versions  map[string]bool
}

type Range struct {
	Type   string
	Events []Event
}

type Event struct {
	Kind    string // introduced | fixed | last_affected
	Version string
}

// DB is an in-memory, filtered view of one or more OSV dumps.
type DB struct {
	Sources    []string
	AsOf       string
	Advisories map[string]int  // per ecosystem, whole database
	Complete   map[string]bool // ecosystems whose own dump was loaded (not just cross-ecosystem mentions)
	Loaded     int
	Diag       []string
	byKey      map[string][]*Advisory
	byArtifact map[string][]*Advisory // Maven artifactId -> advisories
}

// Want selects which advisories are kept in memory. A nil Want keeps none
// (used by `traverse vulndb info`).
type Want struct {
	Keys      map[string]bool // ecosystem|normalized name
	Artifacts map[string]bool // Maven artifactId (vendored jars without metadata)
}

type osvRecord struct {
	ID        string   `json:"id"`
	Modified  string   `json:"modified"`
	Published string   `json:"published"`
	Withdrawn string   `json:"withdrawn"`
	Aliases   []string `json:"aliases"`
	Summary   string   `json:"summary"`
	Details   string   `json:"details"`
	Severity  []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
		Ranges []struct {
			Type   string              `json:"type"`
			Events []map[string]string `json:"events"`
		} `json:"ranges"`
		Versions []string `json:"versions"`
		Severity []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		} `json:"severity"`
	} `json:"affected"`
	DatabaseSpecific map[string]any `json:"database_specific"`
}

type osvHead struct {
	ID        string `json:"id"`
	Modified  string `json:"modified"`
	Withdrawn string `json:"withdrawn"`
	Affected  []struct {
		Package struct {
			Ecosystem string `json:"ecosystem"`
			Name      string `json:"name"`
		} `json:"package"`
	} `json:"affected"`
}

// baseEco drops the release suffix of an OSV ecosystem ("Debian:11" -> "Debian").
func baseEco(e string) string {
	b, _, _ := strings.Cut(e, ":")
	return b
}

var rePyNorm = regexp.MustCompile(`[-_.]+`)

// NormName normalizes a package name the way each registry compares names.
func NormName(eco, name string) string {
	switch eco {
	case "PyPI":
		return rePyNorm.ReplaceAllString(strings.ToLower(name), "-")
	case "Go":
		return name
	}
	return strings.ToLower(name)
}

// Key builds the lookup key of a package.
func Key(eco, name string) string { return eco + "|" + NormName(eco, name) }

// Load reads OSV dumps: all.zip files, directories of zips or JSON files, or
// single JSON records. It never opens a network connection.
func Load(paths []string, want *Want) (*DB, error) {
	db := &DB{Advisories: map[string]int{}, Complete: map[string]bool{}, byKey: map[string][]*Advisory{}, byArtifact: map[string][]*Advisory{}}
	var total int64
	entries := 0
	// A dump unit is one zip, or the loose JSON records of one folder. An
	// ecosystem is complete when it appears in at least half of the records
	// of some unit: an OSV <Ecosystem>/all.zip lists every advisory of that
	// ecosystem, while other ecosystems only appear in a few shared advisories.
	unitTotal := map[string]int{}
	unitEco := map[string]map[string]int{}
	unit := ""
	handle := func(name string, b []byte) {
		entries++
		var h osvHead
		if err := json.Unmarshal(b, &h); err != nil || h.ID == "" {
			return // not an OSV record (e.g. a README in the folder)
		}
		unitTotal[unit]++
		if unitEco[unit] == nil {
			unitEco[unit] = map[string]int{}
		}
		if h.Modified > db.AsOf {
			db.AsOf = h.Modified
		}
		seen := map[string]bool{}
		keep := false
		for _, a := range h.Affected {
			e := baseEco(a.Package.Ecosystem)
			if !seen[e] {
				seen[e] = true
				db.Advisories[e]++
				unitEco[unit][e]++
			}
			if want != nil {
				if want.Keys[Key(e, a.Package.Name)] {
					keep = true
				}
				if e == "Maven" && len(want.Artifacts) > 0 {
					_, art, _ := strings.Cut(strings.ToLower(a.Package.Name), ":")
					if want.Artifacts[art] {
						keep = true
					}
				}
			}
		}
		if !keep || h.Withdrawn != "" {
			return
		}
		var r osvRecord
		if err := json.Unmarshal(b, &r); err != nil {
			db.Diag = append(db.Diag, fmt.Sprintf("%s: %v", name, err))
			return
		}
		db.add(toAdvisory(&r))
	}
	read := func(name string, size int64, open func() (io.ReadCloser, error)) error {
		if entries >= MaxEntries {
			return errors.New("too many entries (limit reached)")
		}
		if size > MaxEntryBytes {
			db.Diag = append(db.Diag, fmt.Sprintf("%s: skipped, %d bytes exceeds the per-record limit", name, size))
			return nil
		}
		total += size
		if total > MaxTotalBytes {
			return errors.New("database exceeds the total size limit")
		}
		rc, err := open()
		if err != nil {
			db.Diag = append(db.Diag, fmt.Sprintf("%s: %v", name, err))
			return nil
		}
		defer rc.Close()
		b, err := io.ReadAll(io.LimitReader(rc, MaxEntryBytes+1))
		if err != nil {
			db.Diag = append(db.Diag, fmt.Sprintf("%s: %v", name, err))
			return nil
		}
		if int64(len(b)) > MaxEntryBytes {
			db.Diag = append(db.Diag, fmt.Sprintf("%s: skipped, exceeds the per-record limit", name))
			return nil
		}
		handle(name, b)
		return nil
	}
	readZip := func(p string) error {
		unit = p
		zr, err := zip.OpenReader(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".json") {
				continue
			}
			f := f
			if err := read(filepath.Base(p)+"!"+f.Name, int64(f.UncompressedSize64), f.Open); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
		}
		return nil
	}
	readJSON := func(p string, size int64) error {
		unit = filepath.Dir(p)
		return read(p, size, func() (io.ReadCloser, error) { return os.Open(p) })
	}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		db.Sources = append(db.Sources, filepath.Base(filepath.Clean(p)))
		if !st.IsDir() {
			if strings.HasSuffix(strings.ToLower(p), ".zip") {
				err = readZip(p)
			} else {
				err = readJSON(p, st.Size())
			}
			if err != nil {
				return nil, err
			}
			continue
		}
		err = filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			low := strings.ToLower(q)
			switch {
			case strings.HasSuffix(low, ".zip"):
				return readZip(q)
			case strings.HasSuffix(low, ".json"):
				info, err := d.Info()
				if err != nil {
					return nil
				}
				return readJSON(q, info.Size())
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for u, n := range unitTotal {
		for e, c := range unitEco[u] {
			if 2*c >= n {
				db.Complete[e] = true
			}
		}
	}
	sort.Strings(db.Sources)
	return db, nil
}

// Ecosystems lists the complete ecosystems, sorted.
func (db *DB) Ecosystems() []string {
	var out []string
	for e := range db.Complete {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

func (db *DB) add(a *Advisory) {
	db.Loaded++
	for _, af := range a.Affected {
		k := af.Ecosystem + "|" + af.Name
		db.byKey[k] = append(db.byKey[k], a)
		if af.Ecosystem == "Maven" {
			_, art, _ := strings.Cut(af.Name, ":")
			db.byArtifact[art] = append(db.byArtifact[art], a)
		}
	}
}

func severityFrom(list []struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}) (float64, string) {
	for _, s := range list {
		if s.Type == "CVSS_V3" {
			if sc, ok := CVSS3(s.Score); ok {
				return sc, s.Score
			}
		}
	}
	return 0, ""
}

func toAdvisory(r *osvRecord) *Advisory {
	a := &Advisory{ID: r.ID, Aliases: r.Aliases, Summary: strings.TrimSpace(r.Summary), Published: r.Published, Modified: r.Modified,
		Malicious: strings.HasPrefix(r.ID, "MAL-")}
	if a.Summary == "" {
		d := strings.TrimSpace(r.Details)
		if i := strings.IndexByte(d, '\n'); i > 0 {
			d = d[:i]
		}
		if len(d) > 200 {
			d = d[:200] + "…"
		}
		a.Summary = d
	}
	a.Score, a.Vector = severityFrom(r.Severity)
	for _, af := range r.Affected {
		if a.Vector == "" {
			a.Score, a.Vector = severityFrom(af.Severity)
		}
		eco := baseEco(af.Package.Ecosystem)
		x := Affected{Ecosystem: eco, Name: NormName(eco, af.Package.Name), Versions: map[string]bool{}}
		for _, v := range af.Versions {
			x.Versions[v] = true
		}
		for _, rg := range af.Ranges {
			if rg.Type != "ECOSYSTEM" && rg.Type != "SEMVER" {
				continue
			}
			y := Range{Type: rg.Type}
			for _, ev := range rg.Events {
				for _, k := range []string{"introduced", "fixed", "last_affected"} {
					if v, ok := ev[k]; ok {
						y.Events = append(y.Events, Event{k, v})
					}
				}
			}
			x.Ranges = append(x.Ranges, y)
		}
		a.Affected = append(a.Affected, x)
	}
	switch {
	case a.Malicious:
		a.Severity = "critical"
	case a.Vector != "":
		a.Severity = SeverityOf(a.Score)
	default:
		a.Severity = "unknown"
		if s, ok := r.DatabaseSpecific["severity"].(string); ok {
			switch strings.ToUpper(s) {
			case "CRITICAL":
				a.Severity = "critical"
			case "HIGH":
				a.Severity = "high"
			case "MODERATE", "MEDIUM":
				a.Severity = "medium"
			case "LOW":
				a.Severity = "low"
			}
		}
	}
	return a
}

// affects reports whether version v is affected and, if so, the closest fixed
// version above it (empty when none is known).
func (af *Affected) affects(v string) (bool, string) {
	hit := af.Versions[v]
	fixed := ""
	for _, r := range af.Ranges {
		eco := af.Ecosystem
		if r.Type == "SEMVER" {
			eco = "npm"
		}
		evs := append([]Event(nil), r.Events...)
		sort.SliceStable(evs, func(i, j int) bool {
			a, b := evs[i].Version, evs[j].Version
			if a == "0" || b == "0" {
				return a == "0" && b != "0"
			}
			return Compare(eco, a, b) < 0
		})
		in := false
		for _, e := range evs {
			switch e.Kind {
			case "introduced":
				if e.Version == "0" || Compare(eco, v, e.Version) >= 0 {
					in = true
				}
			case "fixed":
				if Compare(eco, v, e.Version) >= 0 {
					in = false
				}
			case "last_affected":
				if Compare(eco, v, e.Version) > 0 {
					in = false
				}
			}
		}
		if in {
			hit = true
			for _, e := range evs {
				if e.Kind == "fixed" && Compare(eco, e.Version, v) > 0 {
					if fixed == "" || Compare(eco, e.Version, fixed) < 0 {
						fixed = e.Version
					}
				}
			}
		}
	}
	return hit, fixed
}

// coordinate maps an Traverse dependency id to an OSV ecosystem and name.
func coordinate(id string) (eco, name string, artifactOnly bool) {
	p, rest, _ := strings.Cut(id, ":")
	switch p {
	case "maven":
		return "Maven", rest, false
	case "jar":
		return "Maven", rest, true
	case "npm":
		return "npm", rest, false
	case "pypi":
		return "PyPI", rest, false
	case "nuget":
		return "NuGet", rest, false
	case "composer":
		return "Packagist", rest, false
	case "go":
		return "Go", rest, false
	}
	return "", "", false
}

// Wanted collects the packages declared in a set of repositories, so that
// Load keeps only relevant advisories in memory.
func Wanted(all []*model.Facts) *Want {
	w := &Want{Keys: map[string]bool{}, Artifacts: map[string]bool{}}
	for _, f := range all {
		for _, m := range f.Modules {
			for _, d := range append(append([]model.Dependency(nil), m.Deps...), m.Transitive...) {
				eco, name, art := coordinate(d.ID)
				switch {
				case eco == "":
				case art:
					w.Artifacts[strings.ToLower(name)] = true
				default:
					w.Keys[Key(eco, name)] = true
				}
			}
		}
	}
	return w
}

func preferRank(id string) int {
	switch {
	case strings.HasPrefix(id, "MAL-"):
		return 0
	case strings.HasPrefix(id, "GHSA-"):
		return 1
	case strings.HasPrefix(id, "PYSEC-"), strings.HasPrefix(id, "GO-"), strings.HasPrefix(id, "RUSTSEC-"):
		return 2
	case strings.HasPrefix(id, "CVE-"):
		return 4
	}
	return 3
}

type hit struct {
	adv   *Advisory
	fixed string
}

// group unifies advisories that are aliases of each other.
func group(hits []hit) [][]hit {
	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == "" || parent[x] == x {
			parent[x] = x
			return x
		}
		r := find(parent[x])
		parent[x] = r
		return r
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	for _, h := range hits {
		find(h.adv.ID)
		for _, al := range h.adv.Aliases {
			union(h.adv.ID, al)
		}
	}
	groups := map[string][]hit{}
	var roots []string
	for _, h := range hits {
		r := find(h.adv.ID)
		if groups[r] == nil {
			roots = append(roots, r)
		}
		groups[r] = append(groups[r], h)
	}
	sort.Strings(roots)
	out := make([][]hit, 0, len(roots))
	for _, r := range roots {
		out = append(out, groups[r])
	}
	return out
}

// Match evaluates every library of a repository against the database and
// fills f.Vulnerabilities and f.VulnScan.
func (db *DB) Match(f *model.Facts, now string) {
	adv := map[string]int{}
	for e := range db.Complete {
		adv[e] = db.Advisories[e]
	}
	vs := &model.VulnScan{Database: strings.Join(db.Sources, ", "), AsOf: db.AsOf, Advisories: adv, Unevaluated: []model.Unevaluated{}}
	f.VulnScan = vs
	f.Vulnerabilities = nil
	own := map[string]bool{}
	for _, m := range f.Modules {
		own[m.ID] = true
	}
	checked := map[string]bool{}
	unev := map[string]bool{}
	byKey := map[string]*model.Vulnerability{}
	var order []string
	byID := map[string]*model.Module{}
	for i := range f.Modules {
		byID[f.Modules[i].ID] = &f.Modules[i]
	}
	// externalBOM: versions may come from a parent POM or an imported BOM that
	// is not in the repository (directly or through in-repo parents), or from
	// a Gradle platform / dependency-management plugin.
	var externalBOM func(m *model.Module, depth int) bool
	externalBOM = func(m *model.Module, depth int) bool {
		if m.Build == "gradle" {
			return true
		}
		for _, d := range m.Deps {
			if d.Managed && strings.EqualFold(d.Scope, "import") && !own[d.ID] {
				return true
			}
		}
		if m.Parent == "" || depth > 20 {
			return false
		}
		p, ok := byID[m.Parent]
		if !ok {
			return true
		}
		return externalBOM(p, depth+1)
	}
	for _, m := range f.Modules {
		externalBOM := externalBOM(&m, 0)
		for _, d := range append(append([]model.Dependency(nil), m.Deps...), m.Transitive...) {
			if d.Managed || own[d.ID] {
				continue
			}
			eco, name, artOnly := coordinate(d.ID)
			if eco == "" {
				continue
			}
			declared := d.Version
			if d.Locked != "" {
				declared = d.Locked
			}
			v, reason := Exact(declared)
			if reason != "" && d.Kind == "transitive" {
				continue // git/file/workspace entries of a lockfile
			}
			if reason == "missing" && eco == "Maven" && externalBOM {
				reason = "external-bom"
			}
			if reason == "" && !db.Complete[eco] {
				reason = "ecosystem-not-loaded"
			}
			if reason != "" {
				k := d.ID + "|" + d.Version + "|" + reason
				if !unev[k] {
					unev[k] = true
					vs.Unevaluated = append(vs.Unevaluated, model.Unevaluated{Package: d.ID, Version: d.Version, Reason: reason, File: d.File, Line: d.Line})
				}
				continue
			}
			checked[d.ID+"@"+v] = true
			var cands []*Advisory
			conf := "high"
			if artOnly {
				art := strings.ToLower(name)
				cands = db.byArtifact[art]
				groups := map[string]bool{}
				for _, a := range cands {
					for _, af := range a.Affected {
						if af.Ecosystem == "Maven" && strings.HasSuffix(af.Name, ":"+art) {
							groups[af.Name] = true
						}
					}
				}
				conf = "medium"
				if len(groups) > 1 {
					conf = "low"
				}
			} else {
				cands = db.byKey[Key(eco, name)]
			}
			if d.Kind == "assembly-ref" {
				conf = "low"
			}
			var hits []hit
			for _, a := range cands {
				for i := range a.Affected {
					af := &a.Affected[i]
					if af.Ecosystem != eco {
						continue
					}
					if artOnly {
						_, art, _ := strings.Cut(af.Name, ":")
						if art != strings.ToLower(name) {
							continue
						}
					} else if af.Name != NormName(eco, name) {
						continue
					}
					if ok, fx := af.affects(v); ok {
						hits = append(hits, hit{a, fx})
						break
					}
				}
			}
			for _, g := range group(hits) {
				sort.Slice(g, func(i, j int) bool {
					ri, rj := preferRank(g[i].adv.ID), preferRank(g[j].adv.ID)
					if ri != rj {
						return ri < rj
					}
					return g[i].adv.ID < g[j].adv.ID
				})
				best := g[0].adv
				key := best.ID + "|" + d.ID + "|" + v
				if x, ok := byKey[key]; ok {
					if !contains(x.Modules, m.ID) {
						x.Modules = append(x.Modules, m.ID)
					}
					continue
				}
				x := &model.Vulnerability{ID: best.ID, Package: d.ID, Ecosystem: eco, Version: v, Severity: best.Severity,
					Score: best.Score, Vector: best.Vector, Summary: best.Summary, Malicious: best.Malicious, Confidence: conf,
					Kind: d.Kind, Scope: d.Scope, Modules: []string{m.ID}, Published: best.Published, File: d.File, Line: d.Line}
				if d.LockFile != "" {
					x.LockFile = d.LockFile
				}
				ids := map[string]bool{}
				fixed := map[string]bool{}
				for _, h := range g {
					ids[h.adv.ID] = true
					for _, al := range h.adv.Aliases {
						ids[al] = true
					}
					if h.fixed != "" {
						fixed[h.fixed] = true
					}
					if SeverityRank(h.adv.Severity) > SeverityRank(x.Severity) || (h.adv.Score > x.Score && h.adv.Vector != "") {
						x.Severity, x.Score, x.Vector = h.adv.Severity, h.adv.Score, h.adv.Vector
					}
					if x.Summary == "" {
						x.Summary = h.adv.Summary
					}
					x.Malicious = x.Malicious || h.adv.Malicious
				}
				delete(ids, best.ID)
				for id := range ids {
					x.Aliases = append(x.Aliases, id)
				}
				sort.Slice(x.Aliases, func(i, j int) bool {
					a, b := x.Aliases[i], x.Aliases[j]
					ca, cb := strings.HasPrefix(a, "CVE-"), strings.HasPrefix(b, "CVE-")
					if ca != cb {
						return ca
					}
					return a < b
				})
				for fx := range fixed {
					x.Fixed = append(x.Fixed, fx)
				}
				sort.Slice(x.Fixed, func(i, j int) bool { return Compare(eco, x.Fixed[i], x.Fixed[j]) < 0 })
				byKey[key] = x
				order = append(order, key)
			}
		}
	}
	vs.Checked = len(checked)
	for _, k := range order {
		x := byKey[k]
		sort.Strings(x.Modules)
		f.Vulnerabilities = append(f.Vulnerabilities, *x)
	}
	sort.SliceStable(f.Vulnerabilities, func(i, j int) bool {
		a, b := f.Vulnerabilities[i], f.Vulnerabilities[j]
		if ra, rb := SeverityRank(a.Severity), SeverityRank(b.Severity); ra != rb {
			return ra > rb
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.ID < b.ID
	})
	sort.SliceStable(vs.Unevaluated, func(i, j int) bool {
		a, b := vs.Unevaluated[i], vs.Unevaluated[j]
		if a.Reason != b.Reason {
			return a.Reason < b.Reason
		}
		return a.Package < b.Package
	})
	if days := AgeDays(db.AsOf, now); days > 30 {
		f.Diagnostics = append(f.Diagnostics, model.Diagnostic{Kind: "vulndb-stale",
			Message: fmt.Sprintf("vulnerability database is %d days old (data as of %s); download a fresh OSV dump", days, db.AsOf)})
	}
}

// AgeDays returns the days elapsed between the data date and the scan date
// (both RFC 3339), or 0 when either is unknown.
func AgeDays(asOf, now string) int {
	a, err1 := time.Parse(time.RFC3339, asOf)
	n, err2 := time.Parse(time.RFC3339, now)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int(n.Sub(a).Hours() / 24)
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// Info summarizes a database without keeping it in memory.
func Info(paths []string) (*DB, error) { return Load(paths, nil) }
