package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

type Options struct {
	Lang           string
	MaxNodes       int
	Classification string
	MinConfidence  string // low | medium | high: relationships drawn in diagrams
	HostsSealed    bool
	LinkBase       string // path from portfolio pages to the output root (area views)
	Area           string // set when writing an area view
}

// diagramPortfolio returns a copy of the portfolio without the relationships
// below the minimum confidence, for diagrams only (tables keep everything).
func diagramPortfolio(p *model.Portfolio, min string) *model.Portfolio {
	rank := map[string]int{"low": 1, "medium": 2, "high": 3}
	if rank[min] <= 1 {
		return p
	}
	q := *p
	q.Links = nil
	for _, l := range p.Links {
		if rank[l.Confidence] >= rank[min] {
			q.Links = append(q.Links, l)
		}
	}
	return &q
}

var reSlug = regexp.MustCompile(`[^a-z0-9._-]+`)

func Slug(s string) string {
	s = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "ç", "c", "à", "a", "è", "e", "ò", "o").
		Replace(strings.ToLower(s))
	s = strings.Trim(reSlug.ReplaceAllString(s, "-"), "-")
	if s == "" {
		s = "system"
	}
	return s
}

const css = `
:root{--bg:#f6f7f9;--card:#fff;--fg:#1d2733;--muted:#5b6675;--line:#e1e5eb;--accent:#1f4e8c;--accent-fg:#fff;
--hi:#c9372c;--md:#c77c02;--lo:#3d7a4f;--info:#5b6675;--chip:#eef1f5;--banner:#fff4d6;--bannerfg:#6b4e00}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]){--bg:#12161c;--card:#1a2029;--fg:#e4e8ee;--muted:#9aa5b4;
--line:#2c3440;--accent:#5b8fd6;--accent-fg:#0d1117;--hi:#ff6b5e;--md:#f0a43a;--lo:#6fc28a;--info:#9aa5b4;--chip:#232b36;--banner:#3a3018;--bannerfg:#f3d68a}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif}
header{background:var(--accent);color:var(--accent-fg);padding:22px 24px}header h1{margin:0 0 4px;font-size:22px}
header p{margin:0;opacity:.85;font-size:13px}header p.note{margin-top:6px;color:var(--accent-fg)}header a{color:var(--accent-fg);text-decoration:underline}.banner{background:var(--banner);color:var(--bannerfg);padding:6px 24px;font-weight:600;font-size:12px}
nav{position:sticky;top:0;z-index:5;background:var(--card);border-bottom:1px solid var(--line);padding:8px 24px;display:flex;gap:14px;flex-wrap:wrap}
nav a{color:var(--accent);text-decoration:none;font-size:13px;font-weight:600}main{max-width:1500px;margin:0 auto;padding:16px}
section{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:18px;margin:14px 0}
h2{font-size:17px;margin:0 0 12px}h3{font-size:14px;margin:16px 0 8px;color:var(--muted)}
.kpis{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px}
.kpi{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:12px}.kpi b{display:block;font-size:22px}
.kpi span{color:var(--muted);font-size:12px}.note{color:var(--muted);font-size:12px}
.summary ul{margin:6px 0;padding-left:20px}.summary li{margin:4px 0;line-height:1.5}
.scope{border-left:4px solid var(--md);padding:8px 12px;background:var(--chip);border-radius:4px;font-size:13px}
.tw{overflow:auto;max-height:560px;border:1px solid var(--line);border-radius:8px}
table{border-collapse:collapse;width:100%;font-size:12.5px;min-width:720px}td:first-child{min-width:150px}th,td{text-align:left;padding:6px 9px;border-bottom:1px solid var(--line);vertical-align:top}
th{position:sticky;top:0;background:var(--chip);cursor:pointer;user-select:none;white-space:nowrap}td{overflow-wrap:break-word}
td.n,th.n{text-align:right;font-variant-numeric:tabular-nums}
.f{margin:0 0 8px;padding:6px 9px;width:min(360px,100%);border:1px solid var(--line);border-radius:6px;background:var(--card);color:var(--fg)}
.sev{display:inline-block;padding:1px 8px;border-radius:10px;font-size:11px;font-weight:700;color:#fff}
.sev-high{background:var(--hi)}.sev-medium{background:var(--md)}.sev-low{background:var(--lo)}.sev-info{background:var(--info)}.sev-critical{background:#7a1020}.sev-unknown{background:#6b7280}
.chip{display:inline-block;background:var(--chip);border-radius:4px;padding:0 6px;margin:1px;font-size:11.5px}
.diagram{overflow:auto;border:1px solid var(--line);border-radius:8px;background:var(--bg);padding:8px}.diagram svg{max-width:none;height:auto}
.diagram text{font-family:system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;fill:var(--fg)}
.col{fill:var(--chip)}.colt{font-size:12px;font-weight:600;fill:var(--muted)!important}.nt{font-size:11px}.ns{font-size:10px;fill:var(--muted)!important}
.node{stroke-width:1}.e{fill:none;stroke:var(--muted);stroke-opacity:.5}.e.bad{stroke:var(--hi);stroke-opacity:.9}
.L-Presentation{fill:#e3edfb;stroke:#4a78c2}.L-Business{fill:#e5f4ea;stroke:#3b8a55}.L-Integration{fill:#fff1dc;stroke:#c98a1e}
.L-Persistence{fill:#f6e6f3;stroke:#a24d92}.L-Model{fill:#eef0f3;stroke:#7d8794}.L-Configuration{fill:#efeafb;stroke:#6f56b8}
.L-Cross-cutting{fill:#f1f1ec;stroke:#8b8a6a}.L-Unknown{fill:#fff;stroke:#a7b0bc}
.diagram .node+text,.diagram g text.nt,.diagram g text.ns{fill:#1d2733!important}
.sys{fill:var(--accent);stroke:var(--accent)}.diagram .syst{fill:var(--accent-fg);font-size:13px;font-weight:600}
.box{fill:var(--card);stroke:var(--line)}.bt{font-size:11px;font-weight:600}.bi{font-size:10px;fill:var(--muted)!important}
.lk{fill:none;stroke-width:1.6;stroke-opacity:.85}.lk.lo{stroke-opacity:.35;stroke-width:1;stroke-dasharray:2 4}.k-library{stroke:var(--muted);stroke-dasharray:5 4}.k-http{stroke:#2f6fd0}
.k-soap{stroke:#7d63c9}.k-jms,.k-kafka{stroke:#d98a17}.k-ejb{stroke:#3b8a55}.k-shared-db{stroke:var(--hi);stroke-dasharray:2 3}
.k-shared-datasource{stroke:var(--hi);stroke-dasharray:1 4}.k-deploys{stroke:#2a9d8f;stroke-dasharray:6 3}.lg{font-size:11px}
footer{color:var(--muted);font-size:12px;text-align:center;padding:24px}a{color:var(--accent)}
details summary{cursor:pointer;font-weight:600;color:var(--accent);margin:6px 0}
@media (max-width:700px){main{padding:8px}section{padding:12px}header{padding:16px}}
`

const js = `
document.querySelectorAll('input.f').forEach(function(i){i.addEventListener('input',function(){var t=document.getElementById(i.dataset.t);var v=i.value.toLowerCase();
t.querySelectorAll('tbody tr').forEach(function(r){r.style.display=r.textContent.toLowerCase().indexOf(v)>=0?'':'none'})})});
document.querySelectorAll('th').forEach(function(th){th.addEventListener('click',function(){var tb=th.closest('table').tBodies[0];var i=Array.prototype.indexOf.call(th.parentNode.children,th);
var asc=th.dataset.a!=='1';th.dataset.a=asc?'1':'0';var rows=Array.prototype.slice.call(tb.rows);rows.sort(function(a,b){var x=a.cells[i].textContent.trim(),y=b.cells[i].textContent.trim();
var nx=parseFloat(x),ny=parseFloat(y);var c=(!isNaN(nx)&&!isNaN(ny))?nx-ny:x.localeCompare(y);return asc?c:-c});rows.forEach(function(r){tb.appendChild(r)})})});
`

const csp = `default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:`

func funcs(lang string) template.FuncMap {
	return template.FuncMap{
		"t":     func(k string) string { return tr(lang, k) },
		"dir":   func(d string) string { return trDir(lang, d) },
		"join":  strings.Join,
		"osv":   OSVURL,
		"cvss":  score,
		"why":   func(r string) string { return tr(lang, "why_"+r) },
		"xport": func(t string) string { return transportLabel(lang, t) },
		"role":  func(r string) string { return roleLabel(lang, r) },
		"cp": func(c string) string {
			if c == "" {
				return ""
			}
			return counterpartLabel(lang == "es", c)
		},
		"certLabel":  func(c model.Certificate) string { return certLabel(lang == "es", c) },
		"certStatus": func(st string) string { return certStatus(lang == "es", st) },
		"pct100":     func(f float64) string { return itin(int(f*100+0.5)) + "%" },
		"linkConf":   func(l model.Link) string { return linkConfidence(lang == "es", l) },
		"lines":      joinInts,
		"loc": func(file string, line int) string {
			if line > 0 {
				return fmt.Sprintf("%s:%d", file, line)
			}
			return file
		},
		"showing": func(shown, total int) string { return fmt.Sprintf(tr(lang, "showing"), shown, total) },
		"pct":     func(f float64) string { return fmt.Sprintf("%.2f", f) },
		"k": func(n int) string {
			if n >= 10000 {
				return fmt.Sprintf("%dk", (n+500)/1000)
			}
			return fmt.Sprint(n)
		},
		"sevmap": func(m map[string]int) string { return fmt.Sprintf("%d / %d / %d", m["high"], m["medium"], m["low"]) },
		"cell": func(m map[string]int, k string) string {
			if v := m[k]; v > 0 {
				return fmt.Sprint(v)
			}
			return ""
		},
	}
}

type page struct {
	Lang, Title, Subtitle, Classification, CSS, JS, CSP string
	Generated                                           string
}

func (p page) CSSs() template.CSS { return template.CSS(p.CSS) }
func (p page) JSs() template.JS   { return template.JS(p.JS) }

// limited returns at most n items plus the original length.
type limited[T any] struct {
	Items []T
	Total int
	Cut   bool
}

func limit[T any](xs []T, n int) limited[T] {
	if len(xs) > n {
		return limited[T]{xs[:n], len(xs), true}
	}
	return limited[T]{xs, len(xs), false}
}

type techRow struct {
	Technology, Category, Severity string
	Files                          int
	Paths                          []string
}

type tableRow struct {
	Table, Ops string
	Refs       int
	Components []string
}

type repoView struct {
	page
	F                    *model.Facts
	ContextSVG, LayerSVG template.HTML
	DeploySVG            template.HTML
	DeployNames          map[string]string
	HostsSealed          bool
	DiagramNote          string
	Components           limited[model.Component]
	Contracts            limited[model.Contract]
	Risks                limited[model.Finding]
	Deps                 limited[model.CodeDep]
	Techs                []techRow
	Tables               []tableRow
	BuildDeps            []struct {
		Module string
		D      model.Dependency
	}
	LayerSummary []struct {
		Layer           string
		Types, Packages int
	}
	KCode, KHigh, KMedium, KExposed, KTables int
	Summary                                  []template.HTML
	SecMech                                  []authRow
	SecExternals                             []extRow
	SecMTLS                                  bool
	SecWeak                                  []string
	SecNoUser                                bool
	SecInbound                               string
	Parties                                  partiesView
	Structure                                []template.HTML
	Copies                                   []model.SurveyCopy
	CopiesMore                               int
	Back                                     string
}

const rowLimit = 2500

func buildRepoView(f *model.Facts, p *model.Portfolio, o Options, back string) *repoView {
	v := &repoView{F: f, Back: back}
	v.Summary = summaryHTML(f, p, o.Lang)
	sec := buildSecurity(f, p, o.Lang)
	v.SecMech, v.SecExternals, v.SecMTLS, v.SecWeak, v.SecNoUser = sec.Mechanisms, sec.Externals, sec.MTLS, sec.Weak, sec.NoUserAuth
	v.SecInbound = sec.Inbound
	v.Parties = buildParties(f, p, o.Lang)
	if sv := f.Repo.Survey; sv != nil {
		v.Structure = inlineHTML(structureLines(sv, o.Lang))
		v.Copies = sv.Copies
		if len(v.Copies) > 100 {
			v.CopiesMore, v.Copies = len(v.Copies)-100, v.Copies[:100]
		}
	}
	v.page = page{Lang: o.Lang, Title: f.Repo.Name, Classification: o.Classification, CSS: css, JS: js, CSP: csp, Generated: f.Repo.ScannedAt}
	ctx := ContextSVG(f, o.Lang)
	v.ContextSVG = template.HTML(ctx)
	ls, prefix := LayerSVG(f, o.MaxNodes, o.Lang)
	v.LayerSVG = template.HTML(ls)
	if prefix == "" {
		prefix = "—"
	}
	v.DiagramNote = fmt.Sprintf(tr(o.Lang, "diagramNote"), o.MaxNodes, prefix)
	v.DeploySVG = template.HTML(DeploymentSVG(f, o.Lang))
	v.HostsSealed = o.HostsSealed
	v.DeployNames = map[string]string{}
	for _, n := range f.Deployment.Nodes {
		v.DeployNames[n.ID] = n.Name + " (" + n.Kind + ")"
	}
	var mainComps []model.Component
	layerTypes, layerPkgs := map[string]int{}, map[string]map[string]bool{}
	for _, c := range f.Components {
		mainComps = append(mainComps, c)
		if c.SourceSet == "main" && !strings.HasPrefix(c.Kind, "nested") {
			layerTypes[c.Layer]++
			if layerPkgs[c.Layer] == nil {
				layerPkgs[c.Layer] = map[string]bool{}
			}
			layerPkgs[c.Layer][c.Package] = true
		}
	}
	for _, l := range []string{"Presentation", "Business", "Integration", "Persistence", "Model", "Configuration", "Cross-cutting", "Unknown"} {
		if layerTypes[l] > 0 {
			v.LayerSummary = append(v.LayerSummary, struct {
				Layer           string
				Types, Packages int
			}{l, layerTypes[l], len(layerPkgs[l])})
		}
	}
	v.Components = limit(mainComps, rowLimit)
	var contracts []model.Contract
	tables := map[string]*tableRow{}
	tableOps := map[string]map[string]bool{}
	for _, c := range f.Contracts {
		if c.Kind == "db-table" {
			t := tables[c.Match]
			if t == nil {
				t = &tableRow{Table: c.Key}
				tables[c.Match] = t
				tableOps[c.Match] = map[string]bool{}
			}
			t.Refs++
			tableOps[c.Match][c.Direction] = true
			if c.Component != "" && len(t.Components) < 6 && !containsStr(t.Components, c.Component) {
				t.Components = append(t.Components, c.Component)
			}
			continue
		}
		contracts = append(contracts, c)
		if c.Direction == "exposes" {
			v.KExposed++
		}
	}
	for k, t := range tables {
		var ops []string
		for op := range tableOps[k] {
			ops = append(ops, trDir(o.Lang, op))
		}
		sort.Strings(ops)
		t.Ops = strings.Join(ops, ", ")
		v.Tables = append(v.Tables, *t)
	}
	sort.Slice(v.Tables, func(i, j int) bool { return v.Tables[i].Table < v.Tables[j].Table })
	v.KTables = len(v.Tables)
	v.Contracts = limit(contracts, rowLimit)
	var risks []model.Finding
	techs := map[string]*techRow{}
	for _, fd := range f.Findings {
		if fd.Severity != "info" {
			risks = append(risks, fd)
			if fd.SourceSet == "main" {
				switch fd.Severity {
				case "high":
					v.KHigh++
				case "medium":
					v.KMedium++
				}
			}
		}
		if fd.Technology != "" && fd.Category != "Security" && fd.SourceSet == "main" {
			t := techs[fd.Technology]
			if t == nil {
				t = &techRow{Technology: fd.Technology, Category: fd.Category, Severity: fd.Severity}
				techs[fd.Technology] = t
			}
			t.Files++
			if len(t.Paths) < 3 {
				t.Paths = append(t.Paths, fd.File)
			}
		}
	}
	v.Risks = limit(risks, rowLimit)
	for _, t := range techs {
		v.Techs = append(v.Techs, *t)
	}
	sort.Slice(v.Techs, func(i, j int) bool {
		if v.Techs[i].Category != v.Techs[j].Category {
			return v.Techs[i].Category < v.Techs[j].Category
		}
		return v.Techs[i].Technology < v.Techs[j].Technology
	})
	v.Deps = limit(f.Dependencies, rowLimit)
	for _, m := range f.Modules {
		for _, d := range m.Deps {
			v.BuildDeps = append(v.BuildDeps, struct {
				Module string
				D      model.Dependency
			}{m.Name, d})
		}
	}
	for _, l := range f.Languages {
		if l.Category == "code" || l.Category == "markup" {
			v.KCode += l.Code
		}
	}
	return v
}

type portfolioView struct {
	page
	Base         string // path from the page to the output root ("" or "../../")
	AreaOwn      int    // systems of the area (area views)
	Areas        []areaLink
	P            *model.Portfolio
	LandscapeSVG template.HTML
	Slugs        map[string]string
	TechSystems  []string
	KCode, KHigh int
}

type areaLink struct{ Name, Href string }

// Write produces all outputs under outDir.
func Write(outDir string, all []*model.Facts, p *model.Portfolio, o Options) error {
	if o.MaxNodes <= 0 {
		o.MaxNodes = 60
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	repoT := template.Must(template.New("repo").Funcs(funcs(o.Lang)).Parse(repoTemplate))
	portT := template.Must(template.New("port").Funcs(funcs(o.Lang)).Parse(portfolioTemplate))
	slugs := map[string]string{}
	used := map[string]bool{}
	for _, f := range all {
		s := Slug(f.Repo.Name)
		for i := 2; used[s]; i++ {
			s = fmt.Sprintf("%s-%d", Slug(f.Repo.Name), i)
		}
		used[s] = true
		slugs[f.Repo.Name] = s
		dir := filepath.Join(outDir, "systems", s)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(dir, "facts.json"), f); err != nil {
			return err
		}
		if err := repoCSVs(dir, f); err != nil {
			return err
		}
		v := buildRepoView(f, p, o, "../../index.html")
		if err := writeTemplate(filepath.Join(dir, "report.html"), repoT, v); err != nil {
			return err
		}
		os.WriteFile(filepath.Join(dir, "context.svg"), []byte(Standalone(string(v.ContextSVG))), 0o644)
		if v.LayerSVG != "" {
			os.WriteFile(filepath.Join(dir, "architecture.svg"), []byte(Standalone(string(v.LayerSVG))), 0o644)
		}
		if v.DeploySVG != "" {
			os.WriteFile(filepath.Join(dir, "deployment.svg"), []byte(Standalone(string(v.DeploySVG))), 0o644)
		}
		os.WriteFile(filepath.Join(dir, "architecture.mmd"), []byte(MermaidLayers(f, o.MaxNodes, o.Lang)), 0o644)
		os.WriteFile(filepath.Join(dir, "workspace.dsl"), []byte(StructurizrRepo(f, o.Lang)), 0o644)
		tool := model.ToolInfo{Name: "traverse", Version: f.Tool.Version, RulesVersion: f.Tool.RulesVersion}
		os.WriteFile(filepath.Join(dir, "arc42.md"), []byte(Arc42System(f, p, o)), 0o644)
		os.WriteFile(filepath.Join(dir, "report.md"), []byte(ReportMarkdown(f, p, o)), 0o644)
		if err := save(filepath.Join(dir, "sbom.cdx.json"))(CycloneDX(f, tool)); err != nil {
			return err
		}
		if err := save(filepath.Join(dir, "findings.sarif"))(SARIF([]*model.Facts{f}, tool)); err != nil {
			return err
		}
	}
	var areas []areaLink
	if gs := groups(p); len(gs) > 1 {
		for _, g := range gs {
			areas = append(areas, areaLink{Name: g, Href: "areas/" + Slug(g) + "/index.html"})
		}
		for _, g := range gs {
			ap, keep := areaPortfolio(p, g)
			var sub []*model.Facts
			for _, f := range all {
				if keep[f.Repo.Name] {
					sub = append(sub, f)
				}
			}
			ao := o
			ao.LinkBase, ao.Area = "../../", g
			if err := writePortfolio(filepath.Join(outDir, "areas", Slug(g)), sub, ap, ao, portT, slugs, nil); err != nil {
				return err
			}
		}
	}
	if err := writeCSV(outDir, "validation-sample.csv", validationHeader(o.Lang), validationSample(all, p, o.Lang)); err != nil {
		return err
	}
	return writePortfolio(outDir, all, p, o, portT, slugs, areas)
}

// writePortfolio writes the portfolio-level outputs of a (possibly cut) portfolio.
func writePortfolio(outDir string, all []*model.Facts, p *model.Portfolio, o Options, portT *template.Template, slugs map[string]string, areas []areaLink) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "portfolio.json"), p); err != nil {
		return err
	}
	if err := portfolioCSVs(outDir, p); err != nil {
		return err
	}
	pv := &portfolioView{P: p, Slugs: slugs, Base: o.LinkBase, Areas: areas}
	for _, s := range p.Systems {
		if o.Area != "" && s.Group == o.Area {
			pv.AreaOwn++
		}
	}
	title := tr(o.Lang, "portfolio")
	if o.Area != "" {
		title += " — " + o.Area
	}
	pv.page = page{Lang: o.Lang, Title: title, Classification: o.Classification, CSS: css, JS: js, CSP: csp, Generated: p.GeneratedAt}
	dp := diagramPortfolio(p, o.MinConfidence)
	land := LandscapeSVG(dp, o.Lang)
	pv.LandscapeSVG = template.HTML(land)
	for _, s := range p.Systems {
		pv.TechSystems = append(pv.TechSystems, s.Name)
		pv.KCode += s.Code
		pv.KHigh += s.Risks["high"]
	}
	os.WriteFile(filepath.Join(outDir, "landscape.svg"), []byte(Standalone(land)), 0o644)
	os.WriteFile(filepath.Join(outDir, "landscape.mmd"), []byte(MermaidLandscape(dp)), 0o644)
	os.WriteFile(filepath.Join(outDir, "workspace.dsl"), []byte(StructurizrLandscape(dp)), 0o644)
	os.WriteFile(filepath.Join(outDir, "landscape.puml"), []byte(PlantUMLLandscape(dp)), 0o644)
	os.WriteFile(filepath.Join(outDir, "portfolio.md"), []byte(Arc42Portfolio(all, p, o)), 0o644)
	if err := save(filepath.Join(outDir, "findings.sarif"))(SARIF(all, p.Tool)); err != nil {
		return err
	}
	if err := save(filepath.Join(outDir, "architecture.archimate.xml"))(ArchiMate(all, dp, o.Lang)); err != nil {
		return err
	}
	if o.Classification != "" {
		os.WriteFile(filepath.Join(outDir, "CLASSIFICATION.txt"), []byte(o.Classification+"\n"), 0o644)
	}
	return writeTemplate(filepath.Join(outDir, "index.html"), portT, pv)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeTemplate(path string, t *template.Template, data any) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return t.Execute(fh, data)
}
