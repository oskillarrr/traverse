// Package survey reads the directory structure of a repository before it is
// analyzed: where the code is, which directories are copies of the same
// project (release copies, SVN branches, "entregas/<version>"...), which one is
// the main line, and what is not code at all (documentation, models, built
// binaries). It only looks at names, sizes and build descriptors; it never
// parses source code.
package survey

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/oskillarrr/traverse/internal/build"
	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/model"
)

// Options tune the reconnaissance.
type Options struct {
	ExcludeDirs  map[string]bool // directory names skipped at any depth
	ExcludeGlobs []string        // relative path globs skipped
	MaxDepth     int             // depth of the printed tree (default 4)
	MinFiles     int             // minimum source files for a directory to be compared (default 5)
	Threshold    float64         // Jaccard similarity to call two directories copies (default 0.8)
}

// Node is one directory of the printed tree.
type Node struct {
	Path        string         `json:"path"`
	Depth       int            `json:"depth"`
	Files       int            `json:"files"`
	Source      int            `json:"source"` // code, markup, build, config and SQL files
	ByCat       map[string]int `json:"byCategory"`
	Descriptors []string       `json:"descriptors,omitempty"`
	Coord       string         `json:"coord,omitempty"`
	Version     string         `json:"version,omitempty"`
	Role        string         `json:"role"` // main | copy | code | data | docs | models | binaries | other | container
	Note        string         `json:"note,omitempty"`
	Hidden      int            `json:"hidden,omitempty"` // children not printed
}

// Result is the full reconnaissance of one repository.
type Result struct {
	Root    string        `json:"root"`
	Summary model.Survey  `json:"summary"`
	Tree    []Node        `json:"tree"`
	Skip    []string      `json:"-"` // relative directories to leave out of the analysis
	took    time.Duration // for the CLI
}

// Took is how long the reconnaissance took.
func (r *Result) Took() time.Duration { return r.took }

// File categories used by the survey.
const (
	catCode       = "code"
	catBuild      = "build"
	catConfig     = "config"
	catSQL        = "sql"
	catDocs       = "docs"
	catModels     = "models"
	catDeployable = "deployable"
	catLibrary    = "library"
	catOther      = "other"
)

var officeExt = map[string]bool{".doc": true, ".docx": true, ".pdf": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true, ".vsd": true, ".vsdx": true, ".rtf": true}
var modelExt = map[string]bool{".eap": true, ".eapx": true, ".qea": true, ".qeax": true, ".feap": true, ".vpp": true,
	".archimate": true, ".mdj": true, ".uml": true, ".mdzip": true, ".emx": true, ".bpm": true}

// Category classifies a file by name only.
func Category(rel string) string {
	ext := strings.ToLower(path.Ext(rel))
	switch {
	case officeExt[ext]:
		return catDocs
	case modelExt[ext]:
		return catModels
	case ext == ".war" || ext == ".ear":
		return catDeployable
	case lang.IsArchive(rel) || ext == ".exe" || ext == ".msi" || ext == ".so" || ext == ".dylib":
		return catLibrary
	}
	l, ok := lang.Detect(rel)
	if !ok {
		return catOther
	}
	switch {
	case l.Name == "SQL" || l.Name == "PL/SQL" || l.Name == "Transact-SQL":
		return catSQL
	case l.Category == lang.CatCode || l.Category == lang.CatMarkup:
		return catCode
	case l.Category == lang.CatBuild:
		return catBuild
	case l.Category == lang.CatConfig || l.Category == lang.CatData:
		return catConfig
	case l.Category == lang.CatDocs:
		return catDocs
	}
	return catOther
}

func isSource(cat string) bool {
	return cat == catCode || cat == catBuild || cat == catConfig || cat == catSQL
}

// descriptor names that make a directory the root of a project.
func isDescriptor(name string) bool {
	n := strings.ToLower(name)
	switch n {
	case "pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts",
		"package.json", "go.mod", "pyproject.toml", "setup.py", "composer.json", "build.xml", "cargo.toml":
		return true
	}
	ext := path.Ext(n)
	return ext == ".sln" || ext == ".csproj" || ext == ".vbproj" || ext == ".vbp"
}

type dirStat struct {
	rel     string
	depth   int
	files   int
	source  int
	byCat   map[string]int
	descs   []string
	coord   string
	version string
	mtime   time.Time
	fp      map[string]int64 // source path relative to this dir -> size
	deploy  []string         // .war/.ear directly inside
	codeN   int              // code (not build/config) files below
}

// environment folders (common abbreviations in several languages) hold per-environment variants of the same files: they
// are not copies of the project and must all be analyzed.
var envNames = map[string]bool{"int": true, "pre": true, "pro": true, "cer": true, "des": true, "dev": true, "test": true,
	"tst": true, "local": true, "prod": true, "production": true, "produccion": true, "producción": true, "preproduccion": true,
	"preproducción": true, "desarrollo": true, "integracion": true, "integración": true, "staging": true, "stage": true,
	"qa": true, "uat": true, "sit": true, "pru": true, "pruebas": true, "certificacion": true, "certificación": true,
	"preprod": true, "development": true, "release": true, "prd": true, "stg": true, "perf": true, "sandbox": true,
	"acc": true, "acceptance": true, "hml": true, "hom": true, "homologacao": true, "homologação": true, "producao": true,
	"produção": true, "recette": true, "rec": true, "abnahme": true, "collaudo": true}

// EnvOf returns the environment a path belongs to (int, pre, pro…), if any.
func EnvOf(rel string) string {
	low := strings.ToLower(rel)
	// only configuration says which environment it belongs to; a source
	// package named "integracion" is not the integration environment
	if strings.Contains(low, "/java/") || strings.Contains(low, "/kotlin/") || strings.Contains(low, "/scala/") {
		return ""
	}
	if i := strings.LastIndexByte(low, ':'); i > 0 {
		low = low[:i] // "file:line" evidence
	}
	switch path.Ext(low) {
	case ".java", ".kt", ".scala", ".cs", ".vb", ".js", ".ts", ".jsx", ".tsx", ".jsp", ".py", ".go", ".rb", ".php", ".cbl", ".cob", ".pli":
		return ""
	}
	for _, seg := range strings.Split(low, "/") {
		if envNames[seg] && seg != "test" && seg != "release" {
			return seg
		}
	}
	return ""
}

// vendored runtimes (JDK, application servers) committed inside a repository
var reRuntimeDir = regexp.MustCompile(`(?i)^(jdk|jre|openjdk|java|wlp|wildfly|jboss|jboss-eap|apache-tomcat|tomcat|glassfish|payara|liberty)([-_.\s]?\d.*)?$`)

var reJSONName = regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`)
var reJSONVersion = regexp.MustCompile(`"version"\s*:\s*"([^"]+)"`)
var reAntName = regexp.MustCompile(`<project[^>]*\sname\s*=\s*"([^"]+)"`)

// Run surveys the repository at root.
func Run(root string, o Options) (*Result, error) {
	start := time.Now()
	if o.MaxDepth <= 0 {
		o.MaxDepth = 4
	}
	if o.MinFiles <= 0 {
		o.MinFiles = 5
	}
	if o.Threshold <= 0 {
		o.Threshold = 0.8
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	const cmpDepth = 6 // directories deeper than this are not compared as copy candidates
	dirs := map[string]*dirStat{"": {rel: "", byCat: map[string]int{}}}
	get := func(rel string) *dirStat {
		d := dirs[rel]
		if d == nil {
			depth := 0
			if rel != "" {
				depth = strings.Count(rel, "/") + 1
			}
			d = &dirStat{rel: rel, depth: depth, byCat: map[string]int{}}
			dirs[rel] = d
		}
		return d
	}
	err = filepath.WalkDir(abs, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(abs, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		if de.IsDir() {
			if rel == "" {
				return nil
			}
			if o.ExcludeDirs[de.Name()] || matchGlob(rel+"/", o.ExcludeGlobs) {
				return filepath.SkipDir
			}
			get(rel)
			return nil
		}
		if !de.Type().IsRegular() || matchGlob(rel, o.ExcludeGlobs) {
			return nil
		}
		info, err := de.Info()
		if err != nil {
			return nil
		}
		cat := Category(rel)
		dir := path.Dir(rel)
		if dir == "." {
			dir = ""
		}
		if cat == catDeployable {
			d := get(dir)
			d.deploy = append(d.deploy, path.Base(rel))
		}
		if isDescriptor(path.Base(rel)) {
			d := get(dir)
			d.descs = append(d.descs, path.Base(rel))
		}
		// aggregate into every ancestor
		for a := dir; ; a = parent(a) {
			d := get(a)
			d.files++
			d.byCat[cat]++
			if isSource(cat) {
				d.source++
				if d.depth <= cmpDepth {
					if d.fp == nil {
						d.fp = map[string]int64{}
					}
					d.fp[strings.ToLower(strings.TrimPrefix(rel, prefix(a)))] = info.Size()
				}
			}
			if cat == catCode {
				d.codeN++
			}
			if mt := info.ModTime(); mt.After(d.mtime) {
				d.mtime = mt
			}
			if a == "" {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// coordinates of project roots
	for _, d := range dirs {
		if len(d.descs) > 0 {
			d.coord, d.version = coordinates(abs, d)
		}
	}
	res := &Result{Root: abs}
	// Subversion standard layout: branches/ and tags/ next to trunk/ are copies
	var svnSkip []string
	for rel := range dirs {
		if b := strings.ToLower(path.Base(rel)); rel != "" && (b == "branches" || b == "tags") {
			if _, ok := dirs[prefix(parent(rel))+"trunk"]; ok {
				svnSkip = append(svnSkip, rel)
			}
		}
	}
	sort.Strings(svnSkip)
	// vendored runtimes: a JDK or an application server committed in the repo
	var runtimes []string
	for _, rel := range sortedKeys(dirs) {
		if rel == "" || !reRuntimeDir.MatchString(path.Base(rel)) || insideList(rel, runtimes) {
			continue
		}
		if hasDescendant(dirs, rel, 3, func(n string) bool {
			b := strings.ToLower(path.Base(n))
			return b == "lib" || b == "jmods" || b == "modules"
		}) {
			runtimes = append(runtimes, rel)
		}
	}
	excluded := append(append([]string{}, svnSkip...), runtimes...)
	pairs := copyPairs(dirs, o, cmpDepth, excluded)
	groups := groupCopies(pairs)
	mains := map[string]bool{}
	copyOf := map[string]string{}
	for _, g := range groups {
		m, reason := chooseMain(g, dirs)
		mains[m] = true
		md := dirs[m]
		res.Summary.Roots = append(res.Summary.Roots, model.SurveyRoot{Path: m, Coord: md.coord, Version: md.version,
			Files: md.files, Main: true, Reason: reason})
		for _, c := range g {
			if c == m {
				continue
			}
			cd := dirs[c]
			copyOf[c] = m
			res.Skip = append(res.Skip, c)
			res.Summary.Copies = append(res.Summary.Copies, model.SurveyCopy{Path: c, Of: m, Version: cd.version,
				Files: cd.files, Identical: identical(md.fp, cd.fp), Overlap: round2(jaccard(md.fp, cd.fp))})
		}
	}
	// branches/tags children not already grouped: copies of the sibling trunk
	grouped := map[string]bool{}
	for _, c := range res.Summary.Copies {
		grouped[c.Path] = true
	}
	for _, sk := range svnSkip {
		trunk := prefix(parent(sk)) + "trunk"
		td := dirs[trunk]
		for _, rel := range sortedKeys(dirs) {
			if parent(rel) != sk || grouped[rel] {
				continue
			}
			cd := dirs[rel]
			res.Summary.Copies = append(res.Summary.Copies, model.SurveyCopy{Path: rel, Of: trunk, Version: cd.version, Files: cd.files,
				Identical: identical(td.fp, cd.fp), Overlap: round2(jaccard(td.fp, cd.fp))})
			copyOf[rel] = trunk
		}
		res.Skip = append(res.Skip, sk)
		if !mains[trunk] && td != nil && !insideAny(trunk, mains) {
			mains[trunk] = true
			res.Summary.Roots = append(res.Summary.Roots, model.SurveyRoot{Path: trunk, Coord: td.coord, Version: td.version,
				Files: td.files, Main: true, Reason: "trunk"})
		}
	}
	for _, rt := range runtimes {
		res.Skip = append(res.Skip, rt)
	}
	skipped := func(rel string) bool {
		for _, s := range res.Skip {
			if rel == s || strings.HasPrefix(rel, s+"/") {
				return true
			}
		}
		return false
	}
	// other code roots: top-most project roots outside copies, then loose code
	for _, rel := range sortedKeys(dirs) {
		d := dirs[rel]
		if len(d.descs) == 0 || skipped(rel) || mains[rel] || insideAny(rel, mains) || hasProjectAncestor(rel, dirs, skipped) {
			continue
		}
		res.Summary.Roots = append(res.Summary.Roots, model.SurveyRoot{Path: rel, Coord: d.coord, Version: d.version, Files: d.files})
	}
	// loose source outside any project (configuration folders, scripts...)
	projectBelow := func(rel string) bool {
		for k, d := range dirs {
			if len(d.descs) > 0 && k != rel && isAncestor(rel, k) && !skipped(k) {
				return true
			}
		}
		return false
	}
	liveSource := func(rel string) int { // source files under rel outside skipped copies
		n := dirs[rel].source
		for _, sk := range res.Skip {
			if isAncestor(rel, sk) && sk != rel {
				n -= dirs[sk].source
			}
		}
		return n
	}
	for _, rel := range sortedKeys(dirs) {
		d := dirs[rel]
		if rel == "" || liveSource(rel) <= 0 || skipped(rel) || len(d.descs) > 0 || hasProjectAncestor(rel, dirs, skipped) ||
			projectBelow(rel) || d.byCat[catSQL] == d.source {
			continue
		}
		if p := parent(rel); p != "" && !projectBelow(p) {
			continue // reported at a higher level
		}
		res.Summary.Roots = append(res.Summary.Roots, model.SurveyRoot{Path: rel, Files: d.files})
	}
	// non-code areas: top-most directories without any source file
	for _, rel := range sortedKeys(dirs) {
		d := dirs[rel]
		if rel == "" || d.files == 0 || d.source > 0 || skipped(rel) {
			continue
		}
		if p := parent(rel); p != "" && dirs[p].source == 0 {
			continue // reported at the parent
		}
		res.Summary.NonCode = append(res.Summary.NonCode, model.SurveyArea{Path: rel, Kind: areaKind(d), Files: d.files})
	}
	for _, rt := range runtimes {
		res.Summary.NonCode = append(res.Summary.NonCode, model.SurveyArea{Path: rt, Kind: "runtime", Files: dirs[rt].files})
	}
	// SQL-only areas are analyzed (data model) but worth naming
	for _, rel := range sortedKeys(dirs) {
		d := dirs[rel]
		if rel == "" || skipped(rel) || d.byCat[catSQL] == 0 || d.byCat[catSQL] != d.source {
			continue
		}
		if p := parent(rel); p != "" && dirs[p].byCat[catSQL] == dirs[p].source && dirs[p].source > 0 {
			continue
		}
		res.Summary.NonCode = append(res.Summary.NonCode, model.SurveyArea{Path: rel, Kind: "data", Files: d.files})
	}
	// modules that ship built artifacts but no code: the project that owns the
	// artifact (nearest directory with a build descriptor) has no code at all
	for _, rel := range sortedKeys(dirs) {
		d := dirs[rel]
		if len(d.deploy) == 0 || skipped(rel) {
			continue
		}
		owner := rel
		for owner != "" && len(dirs[owner].descs) == 0 {
			owner = parent(owner)
		}
		if dirs[owner].codeN > 0 {
			continue
		}
		arts := append([]string{}, d.deploy...)
		sort.Strings(arts)
		res.Summary.BinaryOnly = append(res.Summary.BinaryOnly, model.SurveyBinary{Path: rel, Artifacts: arts})
	}
	res.Summary.Decision = decision(res)
	res.Tree = tree(dirs, o.MaxDepth, mains, copyOf, res)
	res.took = time.Since(start)
	return res, nil
}

func prefix(dir string) string {
	if dir == "" {
		return ""
	}
	return dir + "/"
}

func parent(rel string) string {
	p := path.Dir(rel)
	if p == "." {
		return ""
	}
	return p
}

func isAncestor(a, b string) bool { // a is an ancestor of b (or equal)
	return a == "" || a == b || strings.HasPrefix(b, a+"/")
}

var sourceTreeSegs = map[string]bool{"src": true, "source": true, "main": true, "test": true, "tests": true,
	"resources": true, "webapp": true, "java": true, "web-inf": true, "meta-inf": true}

func inSourceTree(rel string) bool {
	for _, s := range strings.Split(strings.ToLower(rel), "/") {
		if sourceTreeSegs[s] {
			return true
		}
	}
	return false
}

func insideList(rel string, list []string) bool {
	for _, s := range list {
		if isAncestor(s, rel) {
			return true
		}
	}
	return false
}

func hasDescendant(dirs map[string]*dirStat, rel string, maxDepth int, pred func(string) bool) bool {
	base := strings.Count(rel, "/")
	for k := range dirs {
		if strings.HasPrefix(k, rel+"/") && strings.Count(k, "/")-base <= maxDepth && pred(k) {
			return true
		}
	}
	return false
}

func insideAny(rel string, set map[string]bool) bool {
	for s := range set {
		if s != rel && isAncestor(s, rel) {
			return true
		}
	}
	return false
}

// hasProjectAncestor reports whether a project root sits above rel (rel is a
// module of a bigger project) — only the top-most root is listed.
func hasProjectAncestor(rel string, dirs map[string]*dirStat, skipped func(string) bool) bool {
	if rel == "" {
		return false
	}
	for a := parent(rel); ; a = parent(a) {
		if d := dirs[a]; d != nil && len(d.descs) > 0 && !skipped(a) {
			return true
		}
		if a == "" {
			return false
		}
	}
}

func sortedKeys(m map[string]*dirStat) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func coordinates(root string, d *dirStat) (coord, version string) {
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(prefix(d.rel)+name)))
		if err != nil || len(b) > 2<<20 {
			return ""
		}
		return string(b)
	}
	for _, n := range d.descs {
		switch ln := strings.ToLower(n); {
		case ln == "pom.xml":
			if p := build.ParsePom(prefix(d.rel)+n, read(n)); p != nil {
				g, v := p.G, p.V
				if g == "" {
					g = p.PG
				}
				if v == "" {
					v = p.PV
				}
				if p.A != "" {
					return "maven:" + g + ":" + p.A, v
				}
			}
		case ln == "package.json":
			t := read(n)
			if m := reJSONName.FindStringSubmatch(t); m != nil {
				v := ""
				if mv := reJSONVersion.FindStringSubmatch(t); mv != nil {
					v = mv[1]
				}
				return "npm:" + m[1], v
			}
		case ln == "build.xml":
			if m := reAntName.FindStringSubmatch(read(n)); m != nil {
				return "ant:" + m[1], ""
			}
		case strings.HasSuffix(ln, ".sln") || strings.HasSuffix(ln, ".csproj") || strings.HasSuffix(ln, ".vbproj") || strings.HasSuffix(ln, ".vbp"):
			return "msbuild:" + strings.TrimSuffix(n, path.Ext(n)), ""
		}
	}
	return "", ""
}

type pair struct {
	a, b string
	j    float64
}

// copyPairs finds pairs of directories that hold the same project. Same build
// coordinates make two project roots copies whatever their content; distinct
// coordinates never do; without coordinates the source trees must overlap.
func copyPairs(dirs map[string]*dirStat, o Options, cmpDepth int, excluded []string) []pair {
	var cand []*dirStat
	for _, d := range dirs {
		if d.rel != "" && d.depth <= cmpDepth && (len(d.fp) >= o.MinFiles || d.coord != "") && !insideList(d.rel, excluded) {
			cand = append(cand, d)
		}
	}
	sort.Slice(cand, func(i, j int) bool { return cand[i].rel < cand[j].rel })
	// effective coordinates: a directory belongs to the nearest project above it
	eff := func(d *dirStat) string {
		for a := d.rel; ; a = parent(a) {
			if x := dirs[a]; x != nil && x.coord != "" {
				return x.coord
			}
			if a == "" {
				return ""
			}
		}
	}
	set := map[[2]string]bool{}
	var out []pair
	for i := 0; i < len(cand); i++ {
		for j := i + 1; j < len(cand); j++ {
			a, b := cand[i], cand[j]
			if isAncestor(a.rel, b.rel) || isAncestor(b.rel, a.rel) {
				continue
			}
			if envNames[strings.ToLower(path.Base(a.rel))] && envNames[strings.ToLower(path.Base(b.rel))] {
				continue // per-environment variants (int/pre/pro...), not copies
			}
			if ea, eb := eff(a), eff(b); ea != "" && eb != "" && ea != eb {
				continue // parts of different projects
			}
			var jv float64
			switch {
			case a.coord != "" && b.coord != "":
				if a.coord != b.coord {
					continue
				}
				jv = jaccard(a.fp, b.fp)
			default:
				// without build coordinates, parts of a source tree (src/main/…,
				// src/test/…) are never project copies: only whole trees are
				if inSourceTree(a.rel) || inSourceTree(b.rel) {
					continue
				}
				na, nb := len(a.fp), len(b.fp)
				if na < o.MinFiles || nb < o.MinFiles {
					continue
				}
				lo, hi := na, nb
				if lo > hi {
					lo, hi = hi, lo
				}
				if float64(lo) < o.Threshold*float64(hi) {
					continue
				}
				jv = jaccardAtLeast(a.fp, b.fp, o.Threshold)
				if jv < o.Threshold {
					continue
				}
			}
			out = append(out, pair{a.rel, b.rel, jv})
			set[[2]string{a.rel, b.rel}] = true
		}
	}
	// keep maximal pairs only: shallow pairs first; a deeper pair whose two
	// sides already sit inside copies of the same group says nothing new
	sort.SliceStable(out, func(i, j int) bool {
		di := strings.Count(out[i].a, "/") + strings.Count(out[i].b, "/")
		dj := strings.Count(out[j].a, "/") + strings.Count(out[j].b, "/")
		return di < dj
	})
	up := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if up[x] == "" || up[x] == x {
			return x
		}
		r := find(up[x])
		up[x] = r
		return r
	}
	member := map[string]bool{}
	nearest := func(x string) string {
		for a := parent(x); ; a = parent(a) {
			if member[a] {
				return a
			}
			if a == "" {
				return ""
			}
		}
	}
	var maximal []pair
	for _, p := range out {
		na, nb := nearest(p.a), nearest(p.b)
		if (na != "" || member[""]) && (nb != "" || member[""]) && find(na) == find(nb) {
			continue
		}
		maximal = append(maximal, p)
		member[p.a], member[p.b] = true, true
		if ra, rb := find(p.a), find(p.b); ra != rb {
			up[ra] = rb
		}
	}
	_ = set
	return maximal
}

func groupCopies(pairs []pair) [][]string {
	up := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if up[x] == "" || up[x] == x {
			up[x] = x
			return x
		}
		r := find(up[x])
		up[x] = r
		return r
	}
	for _, p := range pairs {
		ra, rb := find(p.a), find(p.b)
		if ra != rb {
			up[ra] = rb
		}
	}
	byRoot := map[string][]string{}
	for x := range up {
		r := find(x)
		byRoot[r] = append(byRoot[r], x)
	}
	var groups [][]string
	for _, g := range byRoot {
		if len(g) < 2 {
			continue
		}
		sort.Strings(g)
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}

func hasSegment(rel, seg string) bool {
	for _, s := range strings.Split(strings.ToLower(rel), "/") {
		if s == seg {
			return true
		}
	}
	return false
}

// chooseMain picks the main line of a group of copies and says why.
func chooseMain(g []string, dirs map[string]*dirStat) (string, string) {
	var trunk []string
	for _, c := range g {
		if hasSegment(c, "trunk") {
			trunk = append(trunk, c)
		}
	}
	if len(trunk) > 0 {
		sort.Slice(trunk, func(i, j int) bool {
			return len(trunk[i]) < len(trunk[j]) || len(trunk[i]) == len(trunk[j]) && trunk[i] < trunk[j]
		})
		return trunk[0], "trunk"
	}
	best := ""
	for _, c := range g {
		if dirs[c].version != "" && (best == "" || compareVersions(dirs[c].version, dirs[best].version) > 0) {
			best = c
		}
	}
	if best != "" {
		// ties on version: most recently modified, then shortest path
		var tied []string
		for _, c := range g {
			if dirs[c].version != "" && compareVersions(dirs[c].version, dirs[best].version) == 0 {
				tied = append(tied, c)
			}
		}
		if len(tied) == 1 {
			return best, "highest-version"
		}
		return newest(tied, dirs), "highest-version+newest"
	}
	return newest(g, dirs), "newest"
}

func newest(g []string, dirs map[string]*dirStat) string {
	s := append([]string{}, g...)
	sort.Slice(s, func(i, j int) bool {
		a, b := dirs[s[i]], dirs[s[j]]
		if !a.mtime.Equal(b.mtime) {
			return a.mtime.After(b.mtime)
		}
		if len(a.rel) != len(b.rel) {
			return len(a.rel) < len(b.rel)
		}
		return a.rel < b.rel
	})
	return s[0]
}

var reVerTok = regexp.MustCompile(`\d+|[^\d.\-_]+`)

// compareVersions compares versions naturally (2.10.2 > 2.9.0; 1.0 < 1.0.1).
func compareVersions(a, b string) int {
	ta, tb := reVerTok.FindAllString(strings.ToLower(a), -1), reVerTok.FindAllString(strings.ToLower(b), -1)
	for i := 0; i < len(ta) || i < len(tb); i++ {
		if i >= len(ta) {
			return -1
		}
		if i >= len(tb) {
			return 1
		}
		na, ea := strconv.Atoi(ta[i])
		nb, eb := strconv.Atoi(tb[i])
		switch {
		case ea == nil && eb == nil:
			if na != nb {
				if na < nb {
					return -1
				}
				return 1
			}
		case ea == nil: // number beats qualifier (1.0 > 1.0-SNAPSHOT handled loosely)
			return 1
		case eb == nil:
			return -1
		default:
			if c := strings.Compare(ta[i], tb[i]); c != 0 {
				return c
			}
		}
	}
	return 0
}

func jaccard(a, b map[string]int64) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	in := 0
	small, big := a, b
	if len(small) > len(big) {
		small, big = big, small
	}
	for k := range small {
		if _, ok := big[k]; ok {
			in++
		}
	}
	return float64(in) / float64(len(a)+len(b)-in)
}

// jaccardAtLeast computes the similarity, giving up early once it cannot reach t.
func jaccardAtLeast(a, b map[string]int64, t float64) float64 {
	small, big := a, b
	if len(small) > len(big) {
		small, big = big, small
	}
	need := t * float64(len(a)+len(b)) / (1 + t)
	allowedMiss := float64(len(small)) - need
	miss, in := 0, 0
	for k := range small {
		if _, ok := big[k]; ok {
			in++
		} else {
			miss++
			if float64(miss) > allowedMiss {
				return 0
			}
		}
	}
	return float64(in) / float64(len(a)+len(b)-in)
}

func identical(a, b map[string]int64) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func areaKind(d *dirStat) string {
	best, n := "other", 0
	for _, c := range []string{catDocs, catModels, catDeployable, catLibrary, catOther} {
		if d.byCat[c] > n {
			best, n = c, d.byCat[c]
		}
	}
	switch best {
	case catDeployable, catLibrary:
		return "binaries"
	}
	return best
}

func decision(r *Result) string {
	var parts []string
	for _, root := range r.Summary.Roots {
		if !root.Main {
			continue
		}
		n := 0
		for _, c := range r.Summary.Copies {
			if c.Of == root.Path {
				n++
			}
		}
		s := "main line " + display(root.Path)
		if root.Version != "" {
			s += " (" + root.Version + ")"
		}
		parts = append(parts, s+", "+strconv.Itoa(n)+" copies left out ["+root.Reason+"]")
	}
	if len(parts) == 0 {
		return "no copies detected: everything is analyzed"
	}
	return strings.Join(parts, "; ")
}

func display(rel string) string {
	if rel == "" {
		return "."
	}
	return rel
}

func tree(dirs map[string]*dirStat, maxDepth int, mains map[string]bool, copyOf map[string]string, r *Result) []Node {
	nonCode := map[string]string{}
	for _, a := range r.Summary.NonCode {
		nonCode[a.Path] = a.Kind
	}
	binOnly := map[string]bool{}
	for _, b := range r.Summary.BinaryOnly {
		binOnly[b.Path] = true
	}
	roots := map[string]bool{}
	for _, x := range r.Summary.Roots {
		roots[x.Path] = true
	}
	children := map[string][]string{}
	for rel := range dirs {
		if rel != "" {
			p := parent(rel)
			children[p] = append(children[p], rel)
		}
	}
	const maxChildren = 30
	var out []Node
	var visit func(rel string)
	visit = func(rel string) {
		d := dirs[rel]
		n := Node{Path: rel, Depth: d.depth, Files: d.files, Source: d.source, ByCat: d.byCat, Descriptors: d.descs,
			Coord: d.coord, Version: d.version, Role: "container"}
		switch {
		case mains[rel]:
			n.Role = "main"
		case copyOf[rel] != "":
			n.Role, n.Note = "copy", "copy of "+display(copyOf[rel])
			for _, c := range r.Summary.Copies {
				if c.Path == rel && c.Identical {
					n.Note = "identical to " + display(copyOf[rel])
				}
			}
		case binOnly[rel]:
			n.Role = "binaries"
		case nonCode[rel] != "":
			n.Role = nonCode[rel]
		case roots[rel]:
			n.Role = "code"
		case d.files == 0:
			n.Role = "empty"
		case d.source == 0:
			n.Role = "other"
		}
		kids := children[rel]
		sort.Strings(kids)
		stop := d.depth >= maxDepth || n.Role == "copy" || n.Role == "docs" || n.Role == "models" || n.Role == "data"
		if stop {
			n.Hidden = len(kids)
		} else if len(kids) > maxChildren {
			n.Hidden = len(kids) - maxChildren
			kids = kids[:maxChildren]
		}
		out = append(out, n)
		if stop {
			return
		}
		for _, k := range kids {
			visit(k)
		}
	}
	visit("")
	return out
}

func matchGlob(rel string, globs []string) bool {
	for _, g := range globs {
		if ok, _ := path.Match(g, rel); ok {
			return true
		}
		if ok, _ := path.Match(strings.TrimPrefix(g, "**/"), path.Base(strings.TrimSuffix(rel, "/"))); ok {
			return true
		}
	}
	return false
}

// JSON renders the result as indented JSON.
func (r *Result) JSON() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
