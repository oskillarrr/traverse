// Package scan extracts the facts of one repository.
package scan

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oskillarrr/traverse/internal/build"
	"github.com/oskillarrr/traverse/internal/deploy"
	"github.com/oskillarrr/traverse/internal/descriptors"
	"github.com/oskillarrr/traverse/internal/java"
	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/legacy"
	"github.com/oskillarrr/traverse/internal/lite"
	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/rules"
	"github.com/oskillarrr/traverse/internal/secure"
	"github.com/oskillarrr/traverse/internal/survey"
)

type Options struct {
	root         string // absolute repository root, set by Repo
	MaxFileSize  int64
	ExtraExclude []string
	KeepDirs     []string
	IncludeTests bool
	SVNAll       bool            // --layout all: analyze every copy (release copies, branches, tags)
	introspect   map[string]bool // built artifacts read from the inside (modules shipped without code)
	Sealer       *secure.Sealer  // when set, host names are kept as encrypted pseudonyms
	Catalog      *rules.Catalog
	Config       *Config
	Now          string
	ToolName     string
	ToolVersion  string
	Workers      int
}

type fileResult struct {
	file      model.File
	jf        *java.File
	hits      []rules.Hit
	desc      *descriptors.Result
	manifest  *model.Module
	pom       *build.Pom
	jar       *model.Dependency
	contracts []model.Contract // from non-Java sources (SQL, properties...)
	ctxPath   string           // Spring Boot server.servlet.context-path
	minimal   []route          // ASP.NET Core minimal API routes
	lite      *lite.File       // JS/TS, Python, PHP, Go, Ruby structure
	mf        *legacy.Unit     // COBOL program/copybook, JCL job, BMS mapset
	csd       *legacy.Unit     // CICS resource definitions
	inner     []*fileResult    // files read inside a built artifact (WAR/EAR); not repository files
	isInner   bool
	libs      []model.Dependency // libraries embedded in a built artifact
	certs     []rawCert          // public certificates held by the file
	artifact  *artifactInfo
	dep       *deploy.Result // Dockerfile / Compose / Kubernetes / Helm
	diag      []model.Diagnostic
}

// Repo scans one repository root and returns its facts.
func Repo(root, name, group string, o Options) (*model.Facts, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	if name == "" {
		name = SystemName(abs)
	}
	ex := map[string]bool{}
	for _, d := range DefaultExcludedDirs {
		ex[d] = true
	}
	for _, d := range o.KeepDirs {
		delete(ex, d)
	}
	globs := append([]string{}, o.ExtraExclude...)
	if o.Config != nil {
		globs = append(globs, o.Config.Exclude...)
	}
	// Reconnaissance first: where the code is and which directories are copies.
	sv, err := survey.Run(abs, survey.Options{ExcludeDirs: ex, ExcludeGlobs: globs})
	if err != nil {
		return nil, err
	}
	var skip []string
	if !o.SVNAll {
		skip = sv.Skip
	}
	o.introspect = map[string]bool{}
	for _, b := range sv.Summary.BinaryOnly {
		for _, a := range b.Artifacts {
			if b.Path == "" {
				o.introspect[a] = true
			} else {
				o.introspect[b.Path+"/"+a] = true
			}
		}
	}
	sv.Summary.AllCopies = o.SVNAll && len(sv.Summary.Copies) > 0
	wr, err := walk(abs, walkOptions{excludeDirs: ex, excludeGlobs: globs, svnAll: o.SVNAll, skip: skip})
	if err != nil {
		return nil, err
	}
	f := &model.Facts{SchemaVersion: model.SchemaVersion,
		Tool: model.ToolInfo{Name: o.ToolName, Version: o.ToolVersion, RulesVersion: o.Catalog.Version}}
	f.Repo.Name, f.Repo.Group, f.Repo.ScannedAt = name, group, o.Now
	f.Repo.VCS, f.Repo.Branch, f.Repo.Revision = detectVCS(abs)
	f.Repo.Survey = &sv.Summary
	f.Repo.Layout = sv.Summary.Decision
	if o.SVNAll && len(sv.Summary.Copies) > 0 {
		f.Repo.Layout = "all copies analyzed (--layout all): " + sv.Summary.Decision
	}
	if len(wr.skipped) > 0 {
		if len(sv.Summary.Copies) == 0 {
			f.Repo.Layout = "svn-standard (trunk only)"
		}
		f.Repo.Notes = append(f.Repo.Notes, fmt.Sprintf("Skipped %d Subversion branches/tags directories (use --layout all to include): %s",
			len(wr.skipped), strings.Join(firstN(wr.skipped, 10), ", ")))
	}

	results := make([]*fileResult, len(wr.files))
	workers := o.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	var wg sync.WaitGroup
	jobs := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = analyzeFile(wr.files[i], o)
			}
		}()
	}
	for i := range wr.files {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for _, r := range results {
		if r != nil {
			results = append(results, r.inner...)
		}
	}

	o.root = abs
	assemble(f, results, o)
	return f, nil
}

func firstN(xs []string, n int) []string {
	if len(xs) > n {
		return append(append([]string{}, xs[:n]...), "…")
	}
	return xs
}

var reCtxPath = regexp.MustCompile(`(?m)^\s*server\.(?:servlet\.)?context-path\s*[=:]\s*(\S+)`)

func sourceSet(rel string) string {
	l := "/" + strings.ToLower(rel)
	switch {
	case strings.Contains(l, "/generated/") || strings.Contains(l, "/generated-sources/") || strings.Contains(l, "/gen-src/"):
		return "generated"
	case strings.Contains(l, "/src/test/") || strings.Contains(l, "/src/it/") || strings.Contains(l, "/test/") ||
		strings.Contains(l, "/tests/") || strings.Contains(l, "/__tests__/") || strings.Contains(l, "/spec/") ||
		strings.HasSuffix(l, "test.java") || strings.HasSuffix(l, "tests.cs") || strings.HasSuffix(l, ".spec.ts") ||
		strings.HasSuffix(l, ".test.js") || strings.HasSuffix(l, ".spec.js") || strings.HasSuffix(l, "_test.go") ||
		strings.HasPrefix(path.Base(l), "test_"):
		return "test"
	}
	return "main"
}

func analyzeFile(wf walkedFile, o Options) (res *fileResult) {
	// An extractor bug on one file must never abort a portfolio scan.
	defer func() {
		if p := recover(); p != nil {
			res = &fileResult{file: model.File{Path: wf.rel, Language: "Other", Category: "other", Bytes: wf.size, SourceSet: sourceSet(wf.rel)},
				diag: []model.Diagnostic{{File: wf.rel, Kind: "extractor-error", Message: fmt.Sprintf("file skipped after internal error: %v", p)}}}
		}
	}()
	r := &fileResult{}
	l, known := lang.Detect(wf.rel)
	r.file = model.File{Path: wf.rel, Bytes: wf.size, SourceSet: sourceSet(wf.rel)}
	if certCandidate(wf.rel, wf.size) {
		if b, err := os.ReadFile(wf.abs); err == nil {
			r.certs = extractCertificates(b)
		}
	}
	if !known {
		r.file.Language, r.file.Category = "Other", "other"
		if lang.IsArchive(wf.rel) {
			r.file.Language, r.file.Category = "Binary library", "library"
			low := strings.ToLower(wf.rel)
			if strings.HasSuffix(low, ".jar") {
				d := build.VendoredJarAt(wf.abs, wf.rel, wf.size)
				r.jar = &d
			}
			if o.introspect[wf.rel] {
				introspectArchive(r, wf.abs, wf.rel, wf.size, o)
			}
		}
		r.hits = o.Catalog.Apply(wf.rel, "", nil) // path-only rules
		return r
	}
	r.file.Language, r.file.Category = l.Name, l.Category
	if wf.size > o.MaxFileSize {
		r.diag = append(r.diag, model.Diagnostic{File: wf.rel, Kind: "too-big", Message: fmt.Sprintf("%d bytes exceeds --max-file-size", wf.size)})
		r.hits = o.Catalog.Apply(wf.rel, l.Name, nil)
		return r
	}
	raw, err := os.ReadFile(wf.abs)
	if err != nil {
		r.diag = append(r.diag, model.Diagnostic{File: wf.rel, Kind: "read-error", Message: err.Error()})
		return r
	}
	analyzeText(r, wf.rel, raw, l, o)
	return r
}

// analyzeText runs every extractor on the content of one text file. It is
// shared by repository files and by the files read inside built artifacts.
func analyzeText(r *fileResult, rel string, raw []byte, l lang.Language, o Options) {
	if isBinary(raw) {
		r.file.Language, r.file.Category = "Other", "other"
		return
	}
	text, enc := decode(raw)
	if rl := lang.Refine(l, rel, text); rl.Name != l.Name {
		l = rl
		r.file.Language, r.file.Category = l.Name, l.Category
	}
	if enc != "utf-8" {
		r.file.Encoding = enc
	}
	st := lang.Strip(text, l.Syntax)
	r.file.Code, r.file.Comment, r.file.Blank = st.CodeN, st.CommN, st.BlankN
	r.hits = o.Catalog.Apply(rel, l.Name, st)

	base := strings.ToLower(path.Base(rel))
	switch {
	case l.Name == "Java":
		r.jf = java.Parse(st)
	case l.Name == "C#":
		r.jf = java.ParseDialect(st, java.CSharp)
		r.minimal = minimalAPIs(st)
	case l.Name == "Kotlin":
		r.jf = java.ParseDialect(st, java.Kotlin)
	case lite.Supported(l.Name):
		r.lite = lite.Parse(l.Name, rel, st)
	case l.Name == "COBOL":
		ext := strings.ToLower(path.Ext(rel))
		r.mf = legacy.COBOL(st, strings.TrimSuffix(path.Base(rel), path.Ext(rel)), ext == ".cpy" || ext == ".cbx" || strings.Contains(strings.ToLower(rel), "/cpy"))
	case l.Name == "JCL":
		r.mf = legacy.JCL(st, strings.TrimSuffix(path.Base(rel), path.Ext(rel)))
	case l.Name == "Assembler (HLASM)":
		r.mf = legacy.Assembler(strings.TrimSuffix(path.Base(rel), path.Ext(rel)))
	case l.Name == "CICS BMS":
		r.mf = legacy.BMS(st, strings.TrimSuffix(path.Base(rel), path.Ext(rel)))
	case base == "pom.xml":
		if p := build.ParsePom(rel, text); p != nil {
			r.pom = p
		}
	case build.IsManifest(rel):
		r.manifest = build.Parse(rel, text)
	}
	if k := deploy.Detect(rel, l.Name, text); k != "" && r.file.SourceSet == "main" {
		r.dep = deploy.Parse(rel, k, text)
	}
	if l.Category != lang.CatDocs && legacy.HasCSD(text) {
		r.csd = legacy.CSD(st)
	}
	if descriptors.IsDescriptor(rel, text) {
		r.desc = descriptors.Parse(rel, text)
		if r.desc == nil {
			r.diag = append(r.diag, model.Diagnostic{File: rel, Kind: "xml-invalid", Message: "descriptor could not be parsed"})
		}
	}
	if l.Name == "Properties" || l.Name == "YAML" {
		if m := reCtxPath.FindStringSubmatch(st.Code); m != nil {
			r.ctxPath = strings.Trim(m[1], `"'`)
		}
	}
	if r.file.SourceSet == "main" || o.IncludeTests {
		r.contracts = otherContracts(rel, l, text, st)
		r.contracts = append(r.contracts, configContracts(rel, l, text)...)
	}
}

func findingID(ruleID, repo, file string) string {
	h := sha1.Sum([]byte(ruleID + "|" + repo + "|" + file))
	return hex.EncodeToString(h[:])[:16]
}

func assemble(f *model.Facts, results []*fileResult, o Options) {
	// ---- modules ----
	var poms []*build.Pom
	var jars []model.Dependency
	var ctxPaths []struct{ dir, path string }
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.pom != nil {
			poms = append(poms, r.pom)
		}
		if r.manifest != nil {
			f.Modules = append(f.Modules, *r.manifest)
		}
		if r.jar != nil && r.file.SourceSet == "main" {
			jars = append(jars, *r.jar)
		}
		jars = append(jars, r.libs...)
		if r.ctxPath != "" {
			ctxPaths = append(ctxPaths, struct{ dir, path string }{path.Dir(r.file.Path), r.ctxPath})
		}
		f.Diagnostics = append(f.Diagnostics, r.diag...)
	}
	f.Modules = append(build.ResolveMaven(poms), f.Modules...)
	f.Diagnostics = append(f.Diagnostics, build.ApplyLocks(o.root, f.Modules)...)
	// MSBuild project references by path -> module id
	byDesc := map[string]string{}
	for _, m := range f.Modules {
		byDesc[m.Descriptor] = m.ID
	}
	for i := range f.Modules {
		for j, d := range f.Modules[i].Deps {
			if strings.HasPrefix(d.ID, "path:") {
				if id, ok := byDesc[strings.TrimPrefix(d.ID, "path:")]; ok {
					f.Modules[i].Deps[j].ID = id
				}
			}
		}
	}
	moduleOf := func(rel string) string {
		best, bestLen := "", -1
		for _, m := range f.Modules {
			p := m.Path
			if (p == "" || rel == p || strings.HasPrefix(rel, p+"/")) && len(p) > bestLen {
				best, bestLen = m.ID, len(p)
			}
		}
		return best
	}
	modulePath := map[string]string{}
	for _, m := range f.Modules {
		modulePath[m.ID] = m.Path
	}
	if len(jars) > 0 {
		byID := map[string]int{}
		for i, m := range f.Modules {
			byID[m.ID] = i
		}
		for _, j := range jars {
			id := moduleOf(j.File)
			if id == "" {
				id = "repo:" + f.Repo.Name
				if _, ok := byID[id]; !ok {
					f.Modules = append(f.Modules, model.Module{ID: id, Build: "none", Name: f.Repo.Name, Descriptor: "(vendored libraries)"})
					byID[id] = len(f.Modules) - 1
					modulePath[id] = ""
				}
			}
			m := &f.Modules[byID[id]]
			m.Deps = append(m.Deps, j)
		}
	}

	// ---- public certificates ----
	now, _ := time.Parse(time.RFC3339, o.Now)
	for _, r := range results {
		if r == nil || len(r.certs) == 0 {
			continue
		}
		for _, c := range certificates(r.file.Path, r.certs, now) {
			f.Certificates = append(f.Certificates, c)
			if c.Status == "" || c.CA {
				continue
			}
			sev, title := "medium", "Expired certificate in the repository"
			if c.Status == "expiring" {
				sev, title = "low", "Certificate expiring within 90 days"
			}
			f.Findings = append(f.Findings, model.Finding{ID: findingID("SEC-CERT-"+strings.ToUpper(c.Status), f.Repo.Name, c.File+c.Alias+c.NotAfter),
				RuleID: "SEC-CERT-" + strings.ToUpper(c.Status), Title: title, Category: "Security", Technology: "Certificates",
				Severity: sev, Confidence: "high", Message: "Valid until " + c.NotAfter + ".", File: c.File, Lines: []int{1}, Count: 1,
				SourceSet: r.file.SourceSet})
		}
	}

	sort.SliceStable(f.Certificates, func(i, j int) bool {
		a, b := f.Certificates[i], f.Certificates[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Alias < b.Alias
	})

	// ---- built artifacts read from the inside ----
	for _, r := range results {
		if r == nil || r.artifact == nil {
			continue
		}
		a := r.artifact
		mod := moduleOf(a.rel)
		libs := len(r.libs)
		f.Diagnostics = append(f.Diagnostics, model.Diagnostic{File: a.rel, Kind: "artifact-read",
			Message: fmt.Sprintf("module without source code: built artifact read from the inside (%d compiled classes, %d embedded libraries, %d descriptors/configuration files)", a.classes, libs, len(r.inner))})
		for _, m := range f.Modules {
			if m.ID != mod || m.Version == "" || a.version == "" {
				continue
			}
			if a.version != m.Version && !strings.HasPrefix(a.version, m.Version+"-") {
				f.Diagnostics = append(f.Diagnostics, model.Diagnostic{File: a.rel, Kind: "artifact-version-mismatch",
					Message: fmt.Sprintf("built artifact version %s differs from the build descriptor version %s (%s)", a.version, m.Version, m.Descriptor)})
			}
		}
	}

	// ---- files, languages ----
	langs := map[string]*model.LangStat{}
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.isInner {
			continue // read inside a built artifact: not a repository file
		}
		r.file.Module = moduleOf(r.file.Path)
		f.Files = append(f.Files, r.file)
		ls := langs[r.file.Language]
		if ls == nil {
			ls = &model.LangStat{Language: r.file.Language, Category: r.file.Category}
			langs[r.file.Language] = ls
		}
		ls.Files++
		ls.Code += r.file.Code
		ls.Comment += r.file.Comment
		ls.Blank += r.file.Blank
	}
	for _, ls := range langs {
		f.Languages = append(f.Languages, *ls)
	}
	sort.Slice(f.Languages, func(i, j int) bool {
		if f.Languages[i].Code != f.Languages[j].Code {
			return f.Languages[i].Code > f.Languages[j].Code
		}
		return f.Languages[i].Language < f.Languages[j].Language
	})

	// ---- deployables ----
	deps := map[string]*model.Deployable{}
	getDep := func(mod string) *model.Deployable {
		if d, ok := deps[mod]; ok {
			return d
		}
		d := &model.Deployable{Module: mod}
		deps[mod] = d
		return d
	}
	for _, m := range f.Modules {
		switch strings.ToLower(m.Packaging) {
		case "war", "ear", "ejb", "exe", "winexe":
			d := getDep(m.ID)
			d.Kind, d.Descriptor = strings.ToLower(m.Packaging), m.Descriptor
		}
	}
	var jaxrsMaps []struct{ mod, pat string }
	for _, r := range results {
		if r == nil || r.desc == nil {
			continue
		}
		mod := moduleOf(r.file.Path)
		if r.desc.Deployable != nil {
			d := getDep(mod)
			if r.desc.Deployable.ContextRoot != "" {
				d.ContextRoot = r.desc.Deployable.ContextRoot
				d.RootSource = path.Base(r.file.Path)
			}
			if d.Kind == "" || r.desc.Deployable.Kind == "ear" {
				d.Kind = r.desc.Deployable.Kind
			}
			d.Includes = append(d.Includes, r.desc.Deployable.Includes...)
			if d.Descriptor == "" {
				d.Descriptor = r.desc.Deployable.Descriptor
			}
		}
		for _, p := range r.desc.ServletMaps["jaxrs"] {
			jaxrsMaps = append(jaxrsMaps, struct{ mod, pat string }{mod, p})
		}
	}
	for _, c := range ctxPaths {
		d := getDep(moduleOf(c.dir + "/x"))
		if d.Kind == "" {
			d.Kind = "service"
		}
		d.ContextRoot = c.path
		d.RootSource = "application properties"
	}
	// EAR web modules carry context roots for their WARs
	for _, d := range deps {
		for _, inc := range d.Includes {
			if !strings.HasPrefix(inc, "web:") {
				continue
			}
			parts := strings.SplitN(strings.TrimPrefix(inc, "web:"), "@", 2)
			if len(parts) != 2 || parts[1] == "" {
				continue
			}
			war := strings.TrimSuffix(parts[0], ".war")
			for _, m := range f.Modules {
				if m.Name != "" && strings.HasPrefix(war, m.Name) {
					if wd := getDep(m.ID); wd.ContextRoot == "" {
						wd.ContextRoot = parts[1]
						wd.RootSource = "application.xml"
						if wd.Kind == "" {
							wd.Kind = "war"
						}
					}
				}
			}
		}
	}
	for _, j := range jaxrsMaps {
		d := getDep(j.mod)
		d.AppPaths = appendUnique(d.AppPaths, strings.TrimSuffix(j.pat, "/*"))
	}

	// WARs without an explicit context root are deployed under their file name
	// (JBoss/WildFly, Tomcat, Jetty default).
	for _, m := range f.Modules {
		if strings.EqualFold(m.Packaging, "war") && m.Artifact != "" {
			if d := getDep(m.ID); d.ContextRoot == "" {
				d.ContextRoot = "/" + strings.TrimSuffix(m.Artifact, ".war")
				d.RootSource = "inferred from WAR name"
			}
		}
	}

	// A single module with more than one WEB-INF/web.xml is usually a WAR
	// assembled from several web roots (overlays / per-profile variants). The
	// context root is inferred once, so flag it for the architect to verify.
	webRoots := map[string][]string{}
	for _, r := range results {
		if r == nil || path.Base(r.file.Path) != "web.xml" {
			continue
		}
		mod := moduleOf(r.file.Path)
		webRoots[mod] = appendUnique(webRoots[mod], path.Dir(path.Dir(r.file.Path)))
	}
	for mod, roots := range webRoots {
		if len(roots) > 1 {
			sort.Strings(roots)
			f.Diagnostics = append(f.Diagnostics, model.Diagnostic{Kind: "multiple-web-roots", File: mod,
				Message: "module has " + strconv.Itoa(len(roots)) + " web roots (" + strings.Join(roots, ", ") +
					"); likely several deployment variants — verify the context root of each"})
		}
	}

	// ---- java semantics ----
	javaSemantics(f, results, o, moduleOf, getDep)
	otherSemantics(f, results, o, moduleOf, getDep)

	for _, d := range deps {
		if d.Kind == "" && d.ContextRoot == "" && len(d.AppPaths) == 0 {
			continue
		}
		if d.Kind == "" {
			d.Kind = "war"
		}
		f.Deployables = append(f.Deployables, *d)
	}
	sort.Slice(f.Deployables, func(i, j int) bool { return f.Deployables[i].Module < f.Deployables[j].Module })

	// ---- contracts from descriptors / other sources ----
	for _, r := range results {
		if r == nil {
			continue
		}
		if r.desc != nil && (r.file.SourceSet == "main" || o.IncludeTests) {
			mod := moduleOf(r.file.Path)
			for _, c := range r.desc.Contracts {
				// servlet / Struts mappings are relative to the web module context root
				if c.Kind == "http" && c.Direction == "exposes" {
					if d, ok := deps[mod]; ok && d.ContextRoot != "" {
						c.Key = "ANY " + model.JoinPath(d.ContextRoot, strings.TrimPrefix(c.Key, "ANY "))
						if d.RootSource == "inferred from WAR name" {
							c.Detail += " (context root inferred)"
						}
					}
				}
				f.Contracts = append(f.Contracts, c)
			}
		}
		f.Contracts = append(f.Contracts, r.contracts...)
	}

	// attach component-less contracts (URL / SQL literals) to the file's component
	compOfFile := map[string]string{}
	for _, c := range f.Components {
		if _, ok := compOfFile[c.File]; !ok && !strings.HasPrefix(c.Kind, "nested") {
			compOfFile[c.File] = c.FQN
		}
	}
	for i := range f.Contracts {
		if f.Contracts[i].Component == "" {
			f.Contracts[i].Component = compOfFile[f.Contracts[i].File]
		}
	}

	assembleDeployment(f, results, moduleOf)

	// Host names never leave this function in clear text.
	sealHosts(f, o.Sealer)

	// ---- findings ----
	for _, r := range results {
		if r == nil {
			continue
		}
		for _, h := range r.hits {
			if r.file.SourceSet != "main" && !o.IncludeTests && !strings.HasPrefix(h.Rule.ID, "SEC-") {
				continue
			}
			fd := model.Finding{ID: findingID(h.Rule.ID, f.Repo.Name, r.file.Path), RuleID: h.Rule.ID,
				Title: h.Rule.Title, Category: h.Rule.Category, Technology: h.Rule.Technology, Severity: h.Rule.Severity,
				Confidence: h.Rule.Confidence, Message: h.Rule.Message, AuthKind: h.Rule.AuthKind, File: r.file.Path, Lines: h.Lines, Count: h.Count,
				SourceSet: r.file.SourceSet}
			if h.Rule.ID == "SEC-KEYSTORE-FILE" && !r.isInner {
				if sev, title, msg, ok := refineKeystoreFinding(filepath.Join(o.root, filepath.FromSlash(r.file.Path)), r.file.Path); ok {
					fd.Severity, fd.Title, fd.Message, fd.Confidence = sev, title, msg, "high"
				}
			}
			f.Findings = append(f.Findings, fd)
		}
		if r.desc != nil {
			for _, t := range r.desc.Techs {
				f.Findings = append(f.Findings, model.Finding{ID: findingID(t.RuleID, f.Repo.Name, r.file.Path), RuleID: t.RuleID,
					Title: t.Title, Category: t.Category, Technology: t.Technology, Severity: "info", Confidence: "high",
					File: r.file.Path, Lines: []int{t.Line}, Count: 1, SourceSet: r.file.SourceSet})
			}
		}
	}
	// one finding per outdated runtime level (modules inherit it from parents)
	rtIdx := map[string]int{}
	for _, m := range f.Modules {
		sev, msg := runtimeRisk(m.Runtime)
		if sev == "" {
			continue
		}
		if i, ok := rtIdx[m.Runtime]; ok {
			f.Findings[i].Count++
			continue
		}
		rtIdx[m.Runtime] = len(f.Findings)
		f.Findings = append(f.Findings, model.Finding{ID: findingID("RUNTIME-LEVEL:"+m.Runtime, f.Repo.Name, ""), RuleID: "RUNTIME-LEVEL",
			Title: "Outdated runtime level: " + m.Runtime, Category: "Migration", Technology: m.Runtime, Severity: sev,
			Confidence: "high", Message: msg + " (count = modules)", File: m.Descriptor, Count: 1, SourceSet: "main"})
	}
	f.Contracts = dedupeContracts(f.Contracts)
	markClientTLS(f)
	classify(f)

	metrics(f)
	sortFacts(f)
}

// markClientTLS upgrades outbound HTTP/SOAP calls to mutual TLS when the same
// source file configures a client TLS identity (keystore / key managers), and
// records the authentication evident in the client file (Basic / Bearer).
func markClientTLS(f *model.Facts) {
	mtls, auth := map[string]bool{}, map[string]string{}
	for _, fd := range f.Findings {
		switch fd.RuleID {
		case "TLS-CLIENT-IDENTITY":
			mtls[fd.File] = true
		case "AUTH-BASIC-OUTBOUND":
			auth[fd.File] = "basic"
		case "AUTH-BEARER-OUTBOUND":
			if auth[fd.File] == "" {
				auth[fd.File] = "bearer"
			}
		case "AUTH-WS-SECURITY":
			if auth[fd.File] == "" {
				auth[fd.File] = "ws-security"
			}
		}
	}
	for i := range f.Contracts {
		c := &f.Contracts[i]
		if c.Direction != "calls" || (c.Kind != "http" && c.Kind != "soap") {
			continue
		}
		if mtls[c.File] && c.Transport != "plain" {
			c.Transport = "mtls"
			if c.Auth == "" {
				c.Auth = "client-cert"
			}
		}
		if c.Auth == "" {
			c.Auth = auth[c.File]
		}
	}
}

// dedupeContracts removes exact duplicates and generic JNDI entries already
// described more precisely (e.g. a JMS destination at the same location).
func dedupeContracts(cs []model.Contract) []model.Contract {
	precise := map[string]bool{}
	for _, c := range cs {
		if c.Kind != "jndi" {
			precise[c.File+"|"+c.Match] = true
		}
	}
	seen := map[string]bool{}
	var out []model.Contract
	for _, c := range cs {
		if c.Kind == "jndi" && precise[c.File+"|"+c.Match] {
			continue
		}
		k := c.Kind + "|" + c.Direction + "|" + c.Key + "|" + c.File + "|" + fmt.Sprint(c.Line)
		if c.Kind == "db-table" || c.Kind == "dataset" || c.Kind == "cics-file" {
			// one row per table, access type and file (data scripts repeat INSERTs)
			k = c.Kind + "|" + c.Direction + "|" + c.Key + "|" + c.File
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, c)
	}
	return out
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

var reJavaLevel = regexp.MustCompile(`^java\s+(?:1\.)?(\d+)`)
var reDotNet = regexp.MustCompile(`^\.NET\s+v?([1-4])\.`)

func runtimeRisk(rt string) (string, string) {
	if m := reJavaLevel.FindStringSubmatch(rt); m != nil {
		var v int
		fmt.Sscanf(m[1], "%d", &v)
		switch {
		case v <= 6:
			return "high", "Java 6 or older: no supported JDK can target it; upgrade path needed."
		case v == 7:
			return "medium", "Java 7: unsupported by modern JDKs; plan upgrade to 11/17/21."
		}
	}
	if reDotNet.MatchString(rt) {
		return "medium", ".NET Framework is Windows-only; modern .NET requires migration."
	}
	return "", ""
}

func sortFacts(f *model.Facts) {
	sort.Slice(f.Files, func(i, j int) bool { return f.Files[i].Path < f.Files[j].Path })
	sort.Slice(f.Components, func(i, j int) bool { return f.Components[i].FQN < f.Components[j].FQN })
	sort.Slice(f.Dependencies, func(i, j int) bool {
		a, b := f.Dependencies[i], f.Dependencies[j]
		if a.From != b.From {
			return a.From < b.From
		}
		return a.To < b.To
	})
	sort.Slice(f.Contracts, func(i, j int) bool {
		a, b := f.Contracts[i], f.Contracts[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	sevRank := map[string]int{"high": 0, "medium": 1, "low": 2, "info": 3}
	sort.Slice(f.Findings, func(i, j int) bool {
		a, b := f.Findings[i], f.Findings[j]
		if sevRank[a.Severity] != sevRank[b.Severity] {
			return sevRank[a.Severity] < sevRank[b.Severity]
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.File < b.File
	})
	sort.Slice(f.Diagnostics, func(i, j int) bool { return f.Diagnostics[i].File < f.Diagnostics[j].File })
	for i := range f.Modules {
		ds := f.Modules[i].Deps
		sort.Slice(ds, func(a, b int) bool { return ds[a].ID < ds[b].ID })
	}
}

// sealHosts replaces every captured host by its pseudonym, or removes it when
// no key is configured. It is the single exit point for host names.
func sealHosts(f *model.Facts, s *secure.Sealer) {
	for i := range f.Certificates {
		h := f.Certificates[i].Host
		f.Certificates[i].Host = ""
		if s != nil && h != "" {
			f.Certificates[i].Host = s.Token(h)
		}
	}
	for i := range f.Contracts {
		h := f.Contracts[i].Host
		f.Contracts[i].Host = ""
		if s == nil || h == "" || namespaceHosts[strings.ToLower(h)] {
			continue
		}
		f.Contracts[i].Host = s.Token(h)
	}
	for i := range f.Deployment.Nodes {
		var toks []string
		for _, h := range f.Deployment.Nodes[i].Hosts {
			if s != nil {
				toks = append(toks, s.Token(h))
			}
		}
		f.Deployment.Nodes[i].Hosts = toks
	}
}

// assembleDeployment merges per-file IaC results, resolves references and
// links deployment nodes to the code modules they build or run.
func assembleDeployment(f *model.Facts, results []*fileResult, moduleOf func(string) string) {
	var links []model.DeployLink
	ctx := map[string]string{}
	type agg struct {
		fd   deploy.Finding
		file string
		n    int
	}
	findings := map[string]*agg{}
	var order []string
	for _, r := range results {
		if r == nil || r.dep == nil {
			continue
		}
		if r.dep.Err != nil {
			f.Diagnostics = append(f.Diagnostics, model.Diagnostic{File: r.file.Path, Kind: "yaml-partial", Message: r.dep.Err.Error()})
		}
		f.Deployment.Nodes = append(f.Deployment.Nodes, r.dep.Nodes...)
		links = append(links, r.dep.Links...)
		for k, v := range r.dep.Contexts {
			ctx[k] = v
		}
		for _, fd := range r.dep.Findings {
			k := fd.RuleID + "|" + r.file.Path
			if a, ok := findings[k]; ok {
				a.n++
				continue
			}
			findings[k] = &agg{fd, r.file.Path, 1}
			order = append(order, k)
		}
	}
	f.Deployment.Links = deploy.Resolve(f.Deployment.Nodes, links)
	// module linking: build context first, then image name = module/artifact name
	byName := map[string]string{}
	for _, m := range f.Modules {
		if m.Name != "" {
			byName[strings.ToLower(m.Name)] = m.ID
		}
	}
	for i := range f.Deployment.Nodes {
		n := &f.Deployment.Nodes[i]
		if dir, ok := ctx[n.ID]; ok {
			n.Module = moduleOf(strings.TrimPrefix(dir+"/x", "./"))
			continue
		}
		for _, img := range n.Images {
			if id, ok := byName[strings.ToLower(deploy.ImageName(img))]; ok {
				n.Module = id
			}
		}
	}
	sort.Slice(f.Deployment.Nodes, func(i, j int) bool { return f.Deployment.Nodes[i].ID < f.Deployment.Nodes[j].ID })
	sort.Slice(f.Deployment.Links, func(i, j int) bool {
		a, b := f.Deployment.Links[i], f.Deployment.Links[j]
		return a.From+a.To+a.Kind < b.From+b.To+b.Kind
	})
	for _, k := range order {
		a := findings[k]
		f.Findings = append(f.Findings, model.Finding{ID: findingID(a.fd.RuleID, f.Repo.Name, a.file), RuleID: a.fd.RuleID, Title: a.fd.Title,
			Category: a.fd.Category, Technology: a.fd.Technology, Severity: a.fd.Severity, Confidence: "high", Message: a.fd.Message,
			File: a.file, Lines: []int{a.fd.Line}, Count: a.n, SourceSet: "main"})
	}
}
