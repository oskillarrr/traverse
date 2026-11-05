package report

import (
	"fmt"
	"html"
	"html/template"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// summaryHTML renders the executive-summary bullets for the HTML report,
// turning **bold** spans into <strong> and escaping everything else.
func summaryHTML(f *model.Facts, p *model.Portfolio, lang string) []template.HTML {
	return inlineHTML(summaryLines(f, p, lang))
}

// inlineHTML renders the **bold** and `code` spans of report lines as HTML;
// everything else is escaped.
func inlineHTML(lines []string) []template.HTML {
	var out []template.HTML
	for _, line := range lines {
		var b strings.Builder
		bold, code := false, false
		for len(line) > 0 {
			i := strings.IndexAny(line, "*`")
			if i < 0 {
				b.WriteString(html.EscapeString(line))
				break
			}
			switch {
			case line[i] == '`':
				b.WriteString(html.EscapeString(line[:i]))
				if code {
					b.WriteString("</code>")
				} else {
					b.WriteString("<code>")
				}
				code = !code
				line = line[i+1:]
			case strings.HasPrefix(line[i:], "**") && !code:
				b.WriteString(html.EscapeString(line[:i]))
				if bold {
					b.WriteString("</strong>")
				} else {
					b.WriteString("<strong>")
				}
				bold = !bold
				line = line[i+2:]
			default:
				b.WriteString(html.EscapeString(line[:i+1]))
				line = line[i+1:]
			}
		}
		if code {
			b.WriteString("</code>")
		}
		if bold {
			b.WriteString("</strong>")
		}
		out = append(out, template.HTML(b.String()))
	}
	return out
}

// sysAgg holds the figures the executive summary and the Markdown report share.
type sysAgg struct {
	code, files            int
	primaryLang            string
	primaryPct             int
	modules, components    int
	inbound, outbound      int
	apiIn, uiIn            int
	byKind                 map[string]int // direction+kind counts, inbound only, e.g. "in:http"
	tables                 int
	jndi                   int
	findBySev              map[string]int // high | medium | low
	vulnBySev              map[string]int
	topVuln                *model.Vulnerability
	riskTechs              []string // high/medium-severity technologies for this system
	biggestCycle           int
	cycles, layerIssues    int
	runtimeSev, runtimeMsg string
	deployables            []string
}

func aggregate(f *model.Facts, p *model.Portfolio) sysAgg {
	a := sysAgg{byKind: map[string]int{}, findBySev: map[string]int{}, vulnBySev: map[string]int{}}
	for _, l := range f.Languages {
		a.code += l.Code
	}
	a.files = len(f.Files)
	top := 0
	for _, l := range f.Languages {
		if l.Code > top {
			top, a.primaryLang = l.Code, l.Language
		}
	}
	if a.code > 0 {
		a.primaryPct = top * 100 / a.code
	}
	a.modules, a.components = len(f.Modules), len(f.Components)
	tbl := map[string]bool{}
	for _, c := range f.Contracts {
		switch {
		case c.Kind == "db-table":
			tbl[c.Key] = true
		case c.Direction == "exposes" && c.Kind != "program":
			a.inbound++
			a.byKind["in:"+c.Kind]++
			switch c.Role {
			case "api":
				a.apiIn++
			case "ui":
				a.uiIn++
			}
		case c.Direction == "calls" || c.Direction == "publishes" || c.Direction == "subscribes" || c.Kind == "database" && c.Direction == "uses":
			a.outbound++
			a.byKind["out:"+c.Kind]++
		case c.Kind == "jndi" && c.Direction == "uses":
			a.jndi++
		}
	}
	a.tables = len(tbl)
	for _, fd := range f.Findings {
		switch fd.Severity {
		case "high", "medium", "low":
			a.findBySev[fd.Severity]++
		}
	}
	for _, v := range f.Vulnerabilities {
		a.vulnBySev[v.Severity]++
	}
	if len(f.Vulnerabilities) > 0 {
		a.topVuln = &f.Vulnerabilities[0]
	}
	if p != nil {
		for _, t := range p.Technologies {
			if t.Systems[f.Repo.Name] > 0 && (t.Severity == "high" || t.Severity == "medium") {
				a.riskTechs = append(a.riskTechs, t.Technology+severityMark(t.Severity))
			}
		}
		sort.Strings(a.riskTechs)
	}
	a.cycles = len(f.Cycles)
	for _, c := range f.Cycles {
		if n := len(c.Members); n > a.biggestCycle {
			a.biggestCycle = n
		}
	}
	a.layerIssues = len(f.LayerIssues)
	for _, fd := range f.Findings {
		if fd.RuleID == "RUNTIME-LEVEL" {
			a.runtimeSev, a.runtimeMsg = fd.Severity, fd.Title
		}
	}
	for _, d := range f.Deployables {
		s := d.Kind
		if d.ContextRoot != "" {
			s += " " + d.ContextRoot
		}
		a.deployables = append(a.deployables, s)
	}
	return a
}

func severityMark(s string) string {
	if s == "high" {
		return " (!)"
	}
	return ""
}

// summaryLines builds the executive-summary bullets for a system.
func summaryLines(f *model.Facts, p *model.Portfolio, lang string) []string {
	a := aggregate(f, p)
	es := lang == "es"
	var out []string
	add := func(ess, en string, args ...any) { out = append(out, fmt.Sprintf(es2(es, ess, en), args...)) }

	if sv := f.Repo.Survey; sv != nil && (len(sv.Copies) > 0 || len(sv.BinaryOnly) > 0) {
		var parts []string
		for _, r := range sv.Roots {
			if r.Main {
				v := ""
				if r.Version != "" {
					v = " (" + r.Version + ")"
				}
				parts = append(parts, es2(es, "se analiza ", "analyzed ")+"`"+dispPath(r.Path)+"`"+v)
			}
		}
		if n := len(sv.Copies); n > 0 && !sv.AllCopies {
			parts = append(parts, fmt.Sprintf(es2(es, "%d copias del mismo proyecto omitidas", "%d copies of the same project left out"), n))
		}
		if n := len(sv.BinaryOnly); n > 0 {
			parts = append(parts, fmt.Sprintf(es2(es, "%d módulos sin código fuente analizados desde su artefacto", "%d modules without source code read from their artifact"), n))
		}
		add("**Estructura.** %s.", "**Structure.** %s.", strings.Join(parts, "; "))
	}

	add("**Tamaño y stack.** %s líneas de código en %d ficheros; lenguaje principal %s (%d%%). Módulos de build: %d · componentes: %d.",
		"**Size and stack.** %s lines of code across %d files; main language %s (%d%%). Build modules: %d · components: %d.",
		thousands(a.code), a.files, a.primaryLang, a.primaryPct, a.modules, a.components)

	if len(a.deployables) > 0 {
		add("**Despliegue.** %s.", "**Deployment.** %s.", strings.Join(a.deployables, ", "))
	}

	add("**Integraciones.** Expone %d interfaces (%s) y consume %d externas (%s); %d tablas de datos; %d recursos JNDI.",
		"**Integrations.** Exposes %d interfaces (%s) and consumes %d external ones (%s); %d data tables; %d JNDI resources.",
		a.inbound, kindBreakdown(a.byKind, "in:"), a.outbound, kindBreakdown(a.byKind, "out:"), a.tables, a.jndi)
	if pv := buildParties(f, p, lang); len(pv.Calls)+len(pv.CalledBy)+len(pv.Registered) > 0 {
		var names []string
		for i, r := range pv.Calls {
			if i == 8 {
				names = append(names, "…")
				break
			}
			names = append(names, r.Counterpart)
		}
		line := fmt.Sprintf(es2(es, "**Relaciones.** Llama a %d aplicaciones (%s); le llaman %d sistemas analizados",
			"**Relationships.** Calls %d applications (%s); called by %d analyzed systems"), len(pv.Calls), strings.Join(names, ", "), len(pv.CalledBy))
		if len(pv.Registered) > 0 {
			line += fmt.Sprintf(es2(es, "; %d certificados de aplicaciones o servidores de confianza", "; %d certificates of trusted applications or servers"), len(pv.Registered))
		}
		if pv.UnknownAPI > 0 {
			line += fmt.Sprintf(es2(es, "; %d API sin consumidor identificado", "; %d APIs without identified consumer"), pv.UnknownAPI)
		}
		out = append(out, line+".")
	}
	if a.apiIn > 0 || a.uiIn > 0 {
		add("**Interfaces.** API para otros sistemas: %d · pantallas: %d · sin clasificar: %d.",
			"**Interfaces.** API for other systems: %d · screens: %d · unclassified: %d.", a.apiIn, a.uiIn, a.inbound-a.apiIn-a.uiIn)
	}

	if a.topVuln != nil {
		fixed := es2(es, "sin versión corregida publicada", "no fixed version published")
		if len(a.topVuln.Fixed) > 0 {
			fixed = es2(es, "corregida en ", "fixed in ") + strings.Join(a.topVuln.Fixed, ", ")
		}
		add("**Vulnerabilidades.** %d críticas y %d altas de %d conocidas. La más grave: %s %s (%s, %s).",
			"**Vulnerabilities.** %d critical and %d high of %d known. Most severe: %s %s (%s, %s).",
			a.vulnBySev["critical"], a.vulnBySev["high"], len(f.Vulnerabilities),
			a.topVuln.Package, a.topVuln.Version, a.topVuln.ID, fixed)
	} else if f.VulnScan == nil {
		add("**Vulnerabilidades.** No evaluadas (ejecuta con `--vulndb` para cruzarlas con OSV).",
			"**Vulnerabilities.** Not evaluated (run with `--vulndb` to match against OSV).")
	}

	risk := fmt.Sprintf("%d %s · %d %s · %d %s", a.findBySev["high"], es2(es, "altos", "high"),
		a.findBySev["medium"], es2(es, "medios", "medium"), a.findBySev["low"], es2(es, "bajos", "low"))
	if len(a.riskTechs) > 0 {
		add("**Riesgos.** Hallazgos: %s. Tecnología en riesgo: %s.", "**Risks.** Findings: %s. Technology at risk: %s.",
			risk, strings.Join(a.riskTechs, ", "))
	} else {
		add("**Riesgos.** Hallazgos: %s.", "**Risks.** Findings: %s.", risk)
	}
	if a.runtimeMsg != "" {
		add("**Runtime.** %s.", "**Runtime.** %s.", a.runtimeMsg)
	}

	add("**Acoplamiento.** %d ciclos de paquetes (el mayor, %d paquetes) y %d violaciones de capa.",
		"**Coupling.** %d package cycles (largest, %d packages) and %d layer violations.",
		a.cycles, a.biggestCycle, a.layerIssues)

	sec := buildSecurity(f, p, lang)
	if mechs := mechNames(sec.Mechanisms); mechs != "" || sec.MTLS || len(sec.Weak) > 0 {
		parts := []string{}
		if sec.NoUserAuth {
			parts = append(parts, es2(es, "login de usuarios no visible en el código", "user login not visible in the code"))
		}
		if mechs != "" {
			parts = append(parts, mechs)
		}
		if sec.MTLS {
			parts = append(parts, es2(es, "mTLS detectado", "mTLS detected"))
		}
		if sec.UpstreamAPIs > 0 {
			parts = append(parts, es2(es, plural(sec.UpstreamAPIs, "interfaz", "interfaces")+" con mTLS en un intermediario (inferido)",
				plural(sec.UpstreamAPIs, "interface", "interfaces")+" with mTLS at an intermediary (inferred)"))
		}
		if len(sec.Weak) > 0 {
			parts = append(parts, es2(es, "puntos débiles: ", "weak spots: ")+strings.Join(sec.Weak, "; "))
		}
		add("**Seguridad.** %s.", "**Security.** %s.", strings.Join(parts, " · "))
	}
	if n := len(sec.Externals); n > 0 {
		apps, ext := map[string]bool{}, map[string]bool{}
		for _, e := range sec.Externals {
			apps[e.Counterpart] = true
			if e.Scope == es2(es, "externo", "external") {
				ext[e.Counterpart] = true
			}
		}
		_ = n
		add("**Terceros.** %d aplicaciones conectadas (%d externas al portfolio), %d integraciones con ellas.",
			"**Third parties.** %d connected applications (%d outside the portfolio), %d integrations with them.", len(apps), len(ext), len(sec.Externals))
	}

	for _, d := range f.Diagnostics {
		if d.Kind == "multiple-web-roots" {
			add("**Atención.** %s.", "**Attention.** %s.", d.Message)
		}
	}
	return out
}

func kindBreakdown(m map[string]int, prefix string) string {
	var ks []string
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			ks = append(ks, strings.TrimPrefix(k, prefix))
		}
	}
	sort.Slice(ks, func(i, j int) bool {
		if ci, cj := m[prefix+ks[i]], m[prefix+ks[j]]; ci != cj {
			return ci > cj
		}
		return ks[i] < ks[j]
	})
	var parts []string
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("%s ×%d", k, m[prefix+k]))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

func thousands(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// portfolioSummary builds the executive-summary bullets for the whole portfolio.
func portfolioSummary(all []*model.Facts, p *model.Portfolio, lang string) []string {
	es := lang == "es"
	var out []string
	add := func(ess, en string, args ...any) { out = append(out, fmt.Sprintf(es2(es, ess, en), args...)) }

	code := 0
	for _, s := range p.Systems {
		code += s.Code
	}
	lo := 0
	for _, l := range p.Links {
		if l.Confidence == "low" {
			lo++
		}
	}
	add("**Alcance.** %d sistemas, %s líneas de código en total.", "**Scope.** %d systems, %s lines of code in total.",
		len(p.Systems), thousands(code))
	add("**Integraciones.** %d relaciones entre sistemas (%d de confianza baja, a validar) y %d tablas compartidas.",
		"**Integrations.** %d inter-system relationships (%d low-confidence, to validate) and %d shared tables.",
		len(p.Links), lo, len(p.SharedTables))
	if len(p.Vulnerabilities) > 0 {
		crit, high := 0, 0
		for _, v := range p.Vulnerabilities {
			switch v.Severity {
			case "critical":
				crit++
			case "high":
				high++
			}
		}
		add("**Vulnerabilidades.** %d distintas en el portfolio (%d críticas, %d altas).",
			"**Vulnerabilities.** %d distinct across the portfolio (%d critical, %d high).", len(p.Vulnerabilities), crit, high)
	}
	// systems with the most high risks
	type sv struct {
		name string
		high int
	}
	var svs []sv
	for _, s := range p.Systems {
		svs = append(svs, sv{s.Name, s.Risks["high"]})
	}
	sort.Slice(svs, func(i, j int) bool {
		if svs[i].high != svs[j].high {
			return svs[i].high > svs[j].high
		}
		return svs[i].name < svs[j].name
	})
	if len(svs) > 0 && svs[0].high > 0 {
		var top []string
		for _, s := range svs {
			if s.high == 0 || len(top) >= 3 {
				break
			}
			top = append(top, fmt.Sprintf("%s (%d)", s.name, s.high))
		}
		add("**Sistemas con más riesgos altos.** %s.", "**Systems with the most high risks.** %s.", strings.Join(top, ", "))
	}
	return out
}

func mechNames(ms []authRow) string {
	var out []string
	seen := map[string]bool{}
	for _, m := range ms {
		if m.Category == "Authentication" && !seen[m.Mechanism] {
			seen[m.Mechanism] = true
			out = append(out, m.Mechanism)
		}
	}
	if len(out) > 6 {
		out = out[:6]
	}
	return strings.Join(out, ", ")
}
