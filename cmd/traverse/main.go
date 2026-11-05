// Command traverse is the Offline Architecture Analyzer CLI: offline, read-only static
// architecture analysis of one or many source repositories.
package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/portfolio"
	"github.com/oskillarrr/traverse/internal/report"
	"github.com/oskillarrr/traverse/internal/rules"
	"github.com/oskillarrr/traverse/internal/scan"
	"github.com/oskillarrr/traverse/internal/secure"
	"github.com/oskillarrr/traverse/internal/survey"
	"github.com/oskillarrr/traverse/internal/vuln"
)

func loadSealer(keyFile string) (*secure.Sealer, error) {
	if keyFile != "" {
		return secure.FromKeyFile(keyFile)
	}
	if p := os.Getenv("TRAVERSE_PASSPHRASE"); p != "" {
		return secure.FromPassphrase(p)
	}
	return nil, nil
}

func cmdKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	out := fs.String("out", "", "file to create (required); keep it out of repositories and reports")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("usage: traverse keygen --out team.key")
	}
	if _, err := os.Stat(*out); err == nil {
		return fmt.Errorf("%s already exists: refusing to overwrite a key", *out)
	}
	k, err := secure.GenerateKey()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o700); err != nil { // e.g. ~/keys on first use
		return err
	}
	if err := os.WriteFile(*out, []byte(k+"\n"), 0o600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Key written to %s. Share it only with people allowed to see host names.\n", *out)
	return nil
}

func cmdReveal(args []string) error {
	fs := flag.NewFlagSet("reveal", flag.ContinueOnError)
	keyFile := fs.String("key-file", os.Getenv("TRAVERSE_KEY_FILE"), "key file used for the scan")
	pos, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("usage: traverse reveal --key-file team.key REPORT_DIR")
	}
	s, err := loadSealer(*keyFile)
	if err != nil {
		return err
	}
	if s == nil {
		return errors.New("a key is required (--key-file, TRAVERSE_KEY_FILE or TRAVERSE_PASSPHRASE)")
	}
	m, err := s.Reveal(filepath.Join(pos[0], "hosts.vault.json"))
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"token", "host"})
	for _, k := range keys {
		w.Write([]string{k, m[k]})
	}
	w.Flush()
	return w.Error()
}

var version = "dev" // set with -ldflags "-X main.version=..."

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

const usage = `Traverse – Offline Architecture Analyzer %s
Offline, read-only static architecture analysis of source repositories.

Usage:
  traverse scan --out DIR [flags] REPO_PATH...      analyze one or more repositories
  traverse scan --out DIR --manifest repos.csv      analyze repositories listed in a CSV (name,path,group)
  traverse survey [--depth N] [--out DIR] PATH...   show where the code is, copies and main line (reads no code)
  traverse rules list [--rules FILE]                list detection rules
  traverse rules test [--rules FILE]                verify rule examples (for rule authors)
  traverse keygen --out team.key                    create a key to encrypt host names
  traverse reveal --key-file team.key DIR           decrypt the host pseudonyms of a report (prints CSV)
  traverse vulndb info PATH...                      summarize an offline OSV database (all.zip files or folders)
  traverse version

Run "traverse scan -h" for all scan options.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, usage, version)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "scan":
		err = cmdScan(os.Args[2:])
	case "survey":
		err = cmdSurvey(os.Args[2:])
	case "keygen":
		err = cmdKeygen(os.Args[2:])
	case "reveal":
		err = cmdReveal(os.Args[2:])
	case "rules":
		err = cmdRules(os.Args[2:])
	case "vulndb":
		err = cmdVulnDB(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println("traverse", version, "schema", model.SchemaVersion)
	case "help", "-h", "--help":
		fmt.Printf(usage, version)
	default:
		fmt.Fprintf(os.Stderr, usage, version)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// parseInterleaved allows flags before and after positional arguments.
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return pos, nil
		}
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}

type repoSpec struct{ name, path, group string }

func cmdScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	out := fs.String("out", "", "output directory (required, must be outside the repositories)")
	manifest := fs.String("manifest", "", "CSV file with columns name,path[,group]; relative paths are resolved from the CSV location")
	name := fs.String("name", "", "system name (single repository only; default: folder name)")
	group := fs.String("group", "", "group / business domain for a single repository")
	lang := fs.String("lang", envOr("TRAVERSE_LANG", "en"), "report language: en | es")
	cfgPath := fs.String("config", "", "optional traverse.json (exclusions, layer overrides, extra rule packs, classification banner)")
	var ruleFiles, excludes, keep multi
	fs.Var(&ruleFiles, "rules", "additional rules JSON file (repeatable; same id overrides)")
	fs.Var(&excludes, "exclude", "path glob to exclude, relative to each repository (repeatable)")
	fs.Var(&keep, "keep-dir", "analyze a directory name excluded by default, e.g. bin or build (repeatable)")
	noDefault := fs.Bool("no-default-rules", false, "do not load the embedded rule catalog")
	maxMB := fs.Int("max-file-size", 5, "maximum file size to analyze, in MB")
	tests := fs.Bool("include-tests", false, "include test sources in components, graphs and findings")
	layout := fs.String("layout", "auto", "auto: analyze only the main line when the repository holds copies of the same project (release copies, branches, tags); all: analyze everything")
	svnAll := fs.Bool("svn-all", false, "same as --layout all")
	hosts := fs.Bool("include-hosts", false, "require host capture: fails if no key is configured (hosts are always encrypted)")
	keyFile := fs.String("key-file", os.Getenv("TRAVERSE_KEY_FILE"), "key file from `traverse keygen` (or TRAVERSE_KEY_FILE); enables encrypted host names")
	minConf := fs.String("min-confidence", "low", "minimum confidence of relationships drawn in diagrams: low | medium | high (tables keep all)")
	maxNodes := fs.Int("max-nodes", 60, "maximum packages drawn in architecture diagrams")
	workers := fs.Int("workers", 0, "parallel file workers (default: number of CPUs)")
	classification := fs.String("classification", "", "banner printed on every report, e.g. \"Internal use only\"")
	var vulnDBs multi
	fs.Var(&vulnDBs, "vulndb", "offline OSV database: an <Ecosystem>/all.zip from osv.dev or a folder of them (repeatable; also TRAVERSE_VULNDB, path-list separated). Never uses the network")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: traverse scan --out DIR [flags] REPO_PATH...\n\nFlags:\n")
		fs.PrintDefaults()
	}
	paths, err := parseInterleaved(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *out == "" {
		return errors.New("--out is required")
	}
	if *lang != "en" && *lang != "es" {
		return errors.New("--lang must be en or es")
	}
	switch *layout {
	case "auto":
	case "all":
		*svnAll = true
	default:
		return errors.New("--layout must be auto or all")
	}
	var specs []repoSpec
	for _, p := range paths {
		n := ""
		if len(paths) == 1 {
			n = *name
		}
		specs = append(specs, repoSpec{n, p, *group})
	}
	if *manifest != "" {
		ms, err := readManifest(*manifest)
		if err != nil {
			return err
		}
		specs = append(specs, ms...)
	}
	if len(specs) == 0 {
		return errors.New("no repositories given (paths or --manifest)")
	}
	outAbs, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	for _, s := range specs {
		ra, err := filepath.Abs(s.path)
		if err != nil {
			return err
		}
		if within(outAbs, ra) || within(ra, outAbs) {
			return fmt.Errorf("output directory %s must be outside repository %s (and must not contain it)", outAbs, ra)
		}
	}
	switch *minConf {
	case "low", "medium", "high":
	default:
		return errors.New("--min-confidence must be low, medium or high")
	}
	sealer, err := loadSealer(*keyFile)
	if err != nil {
		return err
	}
	if *hosts && sealer == nil {
		return errors.New("--include-hosts needs a key: create one with `traverse keygen --out team.key` and pass --key-file (or set TRAVERSE_KEY_FILE / TRAVERSE_PASSPHRASE)")
	}
	cfg, err := scan.LoadConfig(*cfgPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if *classification == "" {
		*classification = cfg.Classification
	}
	cat, err := rules.Load(!*noDefault, append(append([]string{}, cfg.Rules...), ruleFiles...))
	if err != nil {
		return err
	}
	now := timestamp()
	opts := scan.Options{MaxFileSize: int64(*maxMB) << 20, ExtraExclude: excludes, KeepDirs: keep, IncludeTests: *tests,
		SVNAll: *svnAll, Sealer: sealer, Catalog: cat, Config: cfg, Now: now, ToolName: "traverse", ToolVersion: version, Workers: *workers}

	var all []*model.Facts
	failed := 0
	names := map[string]int{}
	for i, s := range specs {
		start := time.Now()
		fmt.Fprintf(os.Stderr, "[%d/%d] %s ... ", i+1, len(specs), s.path)
		f, err := scan.Repo(s.path, s.name, s.group, opts)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAILED: %v\n", err)
			continue
		}
		if n := names[f.Repo.Name]; n > 0 {
			f.Repo.Name = fmt.Sprintf("%s-%d", f.Repo.Name, n+1)
		}
		names[f.Repo.Name]++
		files := 0
		for _, l := range f.Languages {
			files += l.Files
		}
		fmt.Fprintf(os.Stderr, "%d files, %d components, %d contracts, %d findings (%s)\n",
			files, len(f.Components), len(f.Contracts), len(f.Findings), time.Since(start).Round(time.Millisecond))
		if sv := f.Repo.Survey; sv != nil && len(sv.Copies) > 0 {
			fmt.Fprintf(os.Stderr, "      structure: %s\n", f.Repo.Layout)
		}
		all = append(all, f)
	}
	if len(all) == 0 {
		return errors.New("no repository could be analyzed")
	}
	if env := os.Getenv("TRAVERSE_VULNDB"); env != "" && len(vulnDBs) == 0 {
		vulnDBs = filepath.SplitList(env)
	}
	if len(vulnDBs) > 0 {
		start := time.Now()
		db, err := vuln.Load(vulnDBs, vuln.Wanted(all))
		if err != nil {
			return fmt.Errorf("vulnerability database: %w", err)
		}
		for _, d := range db.Diag {
			fmt.Fprintln(os.Stderr, "vulndb:", d)
		}
		total, vulns := 0, 0
		for _, e := range db.Ecosystems() {
			total += db.Advisories[e]
		}
		for _, f := range all {
			db.Match(f, now)
			vulns += len(f.Vulnerabilities)
		}
		fmt.Fprintf(os.Stderr, "Vulnerabilities: %d (OSV ecosystems: %s; %d advisories; data as of %s; %s)\n", vulns, strings.Join(db.Ecosystems(), ", "), total, db.AsOf, time.Since(start).Round(time.Millisecond))
		if days := vuln.AgeDays(db.AsOf, now); days > 30 {
			fmt.Fprintf(os.Stderr, "WARNING: the vulnerability database is %d days old; download a fresh OSV dump\n", days)
		}
		if total == 0 {
			fmt.Fprintln(os.Stderr, "WARNING: no OSV advisories found in --vulndb; check the path")
		}
	}
	p := portfolio.Build(all, model.ToolInfo{Name: "traverse", Version: version, RulesVersion: cat.Version}, now)
	if err := report.Write(outAbs, all, p, report.Options{Lang: *lang, MaxNodes: *maxNodes, Classification: *classification,
		MinConfidence: *minConf, HostsSealed: sealer != nil}); err != nil {
		return err
	}
	if sealer != nil {
		b, _ := json.MarshalIndent(sealer.Vault(), "", " ")
		if err := os.WriteFile(filepath.Join(outAbs, "hosts.vault.json"), b, 0o600); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Host names encrypted: %d (vault: hosts.vault.json; decrypt with `traverse reveal`)\n", len(sealer.Vault().Entries))
	}

	fmt.Fprintf(os.Stderr, "\nReport: %s\n", filepath.Join(outAbs, "index.html"))
	fmt.Fprintf(os.Stderr, "Systems: %d · relationships: %d · shared tables: %d", len(p.Systems), len(p.Links), len(p.SharedTables))
	if failed > 0 {
		fmt.Fprintf(os.Stderr, " · FAILED: %d", failed)
	}
	fmt.Fprintln(os.Stderr)
	return nil
}

func cmdSurvey(args []string) error {
	fs := flag.NewFlagSet("survey", flag.ContinueOnError)
	depth := fs.Int("depth", 4, "directory levels printed")
	out := fs.String("out", "", "optional directory where survey.md and survey.json are written")
	lang := fs.String("lang", envOr("TRAVERSE_LANG", "en"), "language: en | es")
	var excludes multi
	fs.Var(&excludes, "exclude", "path glob to exclude (repeatable)")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: traverse survey [--depth N] [--out DIR] PATH...\n\nReads only directory names, file sizes and build descriptors; never source code.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	paths, err := parseInterleaved(fs, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if len(paths) == 0 {
		return errors.New("usage: traverse survey [--depth N] [--out DIR] PATH...")
	}
	ex := map[string]bool{}
	for _, d := range scan.DefaultExcludedDirs {
		ex[d] = true
	}
	var md strings.Builder
	var js []json.RawMessage
	for _, p := range paths {
		r, err := survey.Run(p, survey.Options{ExcludeDirs: ex, ExcludeGlobs: excludes, MaxDepth: *depth})
		if err != nil {
			return err
		}
		text, plan := r.Text(*lang), r.Plan(*lang)
		fmt.Println(text)
		fmt.Println(plan)
		fmt.Fprintf(os.Stderr, "(%s)\n\n", r.Took().Round(time.Millisecond))
		md.WriteString("## " + scan.SystemName(r.Root) + "\n\n```\n" + text + "```\n\n" + plan + "\n")
		b, _ := r.JSON()
		js = append(js, b)
	}
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(*out, "survey.md"), []byte("# Survey\n\n"+md.String()), 0o644); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(js, "", "  ")
		if err := os.WriteFile(filepath.Join(*out, "survey.json"), b, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Written: %s, %s\n", filepath.Join(*out, "survey.md"), filepath.Join(*out, "survey.json"))
	}
	return nil
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// timestamp honours SOURCE_DATE_EPOCH for reproducible outputs.
func timestamp() string {
	if v := os.Getenv("SOURCE_DATE_EPOCH"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Unix(n, 0).UTC().Format(time.RFC3339)
		}
	}
	return time.Now().Format(time.RFC3339)
}

func readManifest(path string) ([]repoSpec, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	r := csv.NewReader(fh)
	r.FieldsPerRecord = -1
	r.Comment = '#'
	r.TrimLeadingSpace = true
	base := filepath.Dir(path)
	var out []repoSpec
	for line := 1; ; line++ {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if len(rec) == 0 || (line == 1 && strings.EqualFold(strings.TrimPrefix(rec[0], "\ufeff"), "name")) {
			continue
		}
		if len(rec) < 2 {
			return nil, fmt.Errorf("%s:%d: expected name,path[,group]", path, line)
		}
		p := strings.TrimSpace(rec[1])
		if !filepath.IsAbs(p) {
			p = filepath.Join(base, p)
		}
		s := repoSpec{name: strings.TrimSpace(rec[0]), path: p}
		if len(rec) > 2 {
			s.group = strings.TrimSpace(rec[2])
		}
		out = append(out, s)
	}
	return out, nil
}

func cmdRules(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: traverse rules list|test [--rules FILE] [--no-default-rules]")
	}
	fs := flag.NewFlagSet("rules", flag.ContinueOnError)
	var files multi
	fs.Var(&files, "rules", "additional rules JSON file (repeatable)")
	noDefault := fs.Bool("no-default-rules", false, "only the given files")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cat, err := rules.Load(!*noDefault, files)
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		w := csv.NewWriter(os.Stdout)
		w.Write([]string{"id", "severity", "category", "technology", "title", "languages"})
		for _, r := range cat.Rules {
			w.Write([]string{r.ID, r.Severity, r.Category, r.Technology, r.Title, strings.Join(r.Languages, " ")})
		}
		w.Flush()
		fmt.Fprintf(os.Stderr, "%d rules, catalog version %s\n", len(cat.Rules), cat.Version)
	case "test":
		errs := cat.SelfTest()
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "FAIL", e)
		}
		if len(errs) > 0 {
			return fmt.Errorf("%d rule examples failed", len(errs))
		}
		fmt.Fprintf(os.Stderr, "OK: %d rules, all examples pass\n", len(cat.Rules))
	default:
		return errors.New("usage: traverse rules list|test")
	}
	return nil
}

func cmdVulnDB(args []string) error {
	if len(args) < 2 || args[0] != "info" {
		return errors.New("usage: traverse vulndb info PATH... (OSV all.zip files or folders)")
	}
	db, err := vuln.Info(args[1:])
	if err != nil {
		return err
	}
	fmt.Printf("Sources: %s\nData as of: %s\n", strings.Join(db.Sources, ", "), db.AsOf)
	if days := vuln.AgeDays(db.AsOf, time.Now().UTC().Format(time.RFC3339)); days > 0 {
		fmt.Printf("Age: %d days\n", days)
	}
	ecos := make([]string, 0, len(db.Advisories))
	for e := range db.Advisories {
		ecos = append(ecos, e)
	}
	sort.Strings(ecos)
	partial := false
	for _, e := range ecos {
		if db.Complete[e] {
			fmt.Printf("  %-12s %6d advisories  complete\n", e, db.Advisories[e])
		} else {
			partial = true
			fmt.Printf("  %-12s %6d advisories  partial (not evaluated)\n", e, db.Advisories[e])
		}
	}
	if partial {
		fmt.Println("Partial ecosystems only appear in advisories shared with another ecosystem; download <Ecosystem>/all.zip to evaluate them.")
	}
	for _, d := range db.Diag {
		fmt.Println("  warning:", d)
	}
	if len(ecos) == 0 {
		return errors.New("no OSV records found")
	}
	return nil
}
