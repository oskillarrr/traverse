package report

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/portfolio"
)

// arc42 documents (https://arc42.org) in Markdown, pre-filled with the facts
// that can be derived from source code. Sections that need people are kept
// with an explicit "to complete" marker so the document stays honest.

type md struct {
	strings.Builder
	lang string
}

func (b *md) t(es, en string) string {
	if b.lang == "es" {
		return es
	}
	return en
}

func (b *md) p(format string, a ...any) { fmt.Fprintf(&b.Builder, format+"\n", a...) }

func (b *md) todo(es, en string) {
	b.p("> ✍️ **%s** %s\n", b.t("Por completar:", "To complete:"), b.t(es, en))
}

func cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	if s == "" {
		return "—"
	}
	return s
}

func (b *md) table(head []string, rows [][]string) {
	if len(rows) == 0 {
		b.p("_%s_\n", b.t("No se ha detectado nada.", "Nothing detected."))
		return
	}
	b.p("| %s |", strings.Join(head, " | "))
	seps := make([]string, len(head))
	for i := range seps {
		seps[i] = "---"
	}
	b.p("|%s|", strings.Join(seps, "|"))
	for _, r := range rows {
		cs := make([]string, len(r))
		for i, c := range r {
			cs[i] = cell(c)
		}
		b.p("| %s |", strings.Join(cs, " | "))
	}
	b.p("")
}

var reMID = regexp.MustCompile(`[^A-Za-z0-9_]`)

func mid(s string) string { return "n_" + reMID.ReplaceAllString(s, "_") }

func mlabel(s string) string {
	return strings.NewReplacer(`"`, "'", "[", "(", "]", ")", "{", "(", "}", ")", "|", "/", "<", "‹", ">", "›").Replace(s)
}

// integration modes: how two parts are coupled
func mode(lang, kind string) string {
	es := lang == "es"
	switch kind {
	case "http", "soap", "ejb", "cics-transaction", "program", "jndi", "ldap", "socket":
		if es {
			return "síncrona"
		}
		return "synchronous"
	case "jms", "kafka", "cics-queue":
		if es {
			return "asíncrona (mensajería)"
		}
		return "asynchronous (messaging)"
	case "db-table", "db-schema", "shared-db", "dataset", "shared-dataset", "cics-file", "shared-datasource", "database":
		if es {
			return "por datos compartidos"
		}
		return "through shared data"
	case "library":
		if es {
			return "en compilación (librería)"
		}
		return "build time (library)"
	case "deploys":
		if es {
			return "despliegue"
		}
		return "deployment"
	case "mail", "file-transfer":
		if es {
			return "asíncrona (correo / ficheros)"
		}
		return "asynchronous (mail / files)"
	case "messaging":
		if es {
			return "asíncrona (mensajería)"
		}
		return "asynchronous (messaging)"
	case "platform-module":
		if es {
			return "en ejecución (módulo de plataforma)"
		}
		return "runtime (platform module)"
	}
	return kind
}

type integ struct {
	dir, kind, other, key, mode, comp, evidence, conf, host string
	transport, auth                                         string
	role                                                    string // api | ui for inbound interfaces
	named                                                   bool   // counterpart named by the code (not a portfolio system)
}

// counterpartLabel renders the counterpart the code names.
func counterpartLabel(es bool, cp string) string {
	switch {
	case strings.HasPrefix(cp, "schema:"):
		return es2(es, "esquema ", "schema ") + strings.ToUpper(strings.TrimPrefix(cp, "schema:"))
	case strings.HasPrefix(cp, "module:"):
		return es2(es, "módulo ", "module ") + strings.TrimPrefix(cp, "module:")
	case cp == "mail":
		return es2(es, "servidor de correo", "mail server")
	case cp == "database":
		return es2(es, "base de datos", "database")
	case cp == "self":
		return es2(es, "(propio sistema)", "(this system)")
	}
	return cp
}

// integrations builds the integration catalog of one system from its own
// contracts and from the portfolio relationships (counterparts).
func integrations(f *model.Facts, p *model.Portfolio, lang string) []integ {
	es := lang == "es"
	in, out, shared, ext := "entrante", "saliente", "compartida", "externo (fuera del portfolio)"
	if !es {
		in, out, shared, ext = "inbound", "outbound", "shared", "external (outside the portfolio)"
	}
	sys := f.Repo.Name
	var rows []integ
	used := map[int]bool{}
	ev := func(c model.Contract) string {
		if c.Line > 0 {
			return fmt.Sprintf("%s:%d", c.File, c.Line)
		}
		return c.File
	}
	// 1. relationships with other systems of the portfolio
	for _, l := range p.Links {
		if l.From != sys && l.To != sys {
			continue
		}
		other, dir := l.To, out
		if l.To == sys {
			other, dir = l.From, in
		}
		if strings.HasPrefix(l.Kind, "shared") {
			dir = shared
		}
		if l.Kind == "deploys" && l.To == sys {
			dir = in
		}
		for _, k := range l.Keys {
			evidence := ""
			comp := ""
			xport, xauth, xrole := "", "", ""
			for i, c := range f.Contracts {
				if used[i] {
					continue
				}
				ok := false
				switch {
				case l.Kind == "http" || l.Kind == "soap":
					probe := model.Contract{Kind: c.Kind, Key: k, Match: model.NormalizePath(pathOf(k))}
					ok = c.Kind == l.Kind && ((l.From == sys && c.Direction != "exposes" && portfolio.MatchScore(c, probe) >= 0) || (l.To == sys && c.Direction == "exposes" && c.Key == k))
				case strings.HasPrefix(l.Kind, "shared"):
					ok = strings.EqualFold(c.Match, k) || strings.EqualFold(c.Match, model.NormalizeJNDI(k))
				case l.Kind == "library":
					ok = false
				default:
					ok = c.Kind == l.Kind && (c.Key == k || c.Match == k || c.Match == model.NormalizeJNDI(k))
				}
				if ok {
					evidence, comp = ev(c), c.Component
					xport, xauth = c.Transport, c.Auth
					if c.Direction == "exposes" {
						xrole = c.Role
					}
					used[i] = true
					break
				}
			}
			if l.Kind == "library" {
				for _, m := range f.Modules {
					for _, d := range m.Deps {
						if d.ID == k && evidence == "" {
							evidence = fmt.Sprintf("%s:%d", d.File, d.Line)
							comp = m.Name
						}
					}
				}
			}
			rows = append(rows, integ{dir, l.Kind, other, k, mode(lang, l.Kind), comp, evidence, linkConfidence(lang == "es", l), "", xport, xauth, xrole, false})
		}
	}
	// 2. own contracts not explained by a portfolio relationship
	exposedSeen := map[string]bool{}
	type schemaAgg struct {
		ops    map[string]bool
		tables map[string]bool
		ev     string
	}
	schemas := map[string]*schemaAgg{}
	for i, c := range f.Contracts {
		if used[i] {
			continue
		}
		other, named := ext, false
		if c.Counterpart != "" {
			other, named = counterpartLabel(es, c.Counterpart), true
		}
		switch {
		case c.Direction == "exposes" && c.Kind != "program":
			if exposedSeen[c.Kind+c.Key] {
				continue
			}
			exposedSeen[c.Kind+c.Key] = true
			rows = append(rows, integ{in, c.Kind, es2(es, "(consumidor no identificado)", "(no identified consumer)"), c.Key, mode(lang, c.Kind), c.Component, ev(c), c.Confidence, "", c.Transport, c.Auth, c.Role, false})
		case c.Kind == "db-table" && strings.HasPrefix(c.Counterpart, "schema:"):
			a := schemas[c.Counterpart]
			if a == nil {
				a = &schemaAgg{ops: map[string]bool{}, tables: map[string]bool{}, ev: ev(c)}
				schemas[c.Counterpart] = a
			}
			a.ops[c.Direction] = true
			a.tables[c.Key] = true
		case c.Kind == "platform-module" && c.Counterpart != "":
			rows = append(rows, integ{out, c.Kind, other, c.Key, mode(lang, "library"), c.Component, ev(c), c.Confidence, "", "", "", "", true})
		case c.Direction == "calls" || c.Direction == "publishes" || c.Direction == "subscribes" ||
			(c.Direction == "uses" && (c.Kind == "jms" || c.Kind == "kafka" || c.Kind == "database")):
			if c.Kind == "cics-transaction" {
				continue // internal transfer
			}
			d := out
			if c.Direction == "subscribes" {
				d = in
			}
			rows = append(rows, integ{d, c.Kind, other, c.Key, mode(lang, c.Kind), c.Component, ev(c), c.Confidence, c.Host, c.Transport, c.Auth, "", named})
		case c.Kind == "jndi" && c.Direction == "uses":
			o := es2(es, "recurso del servidor", "server resource")
			if c.Counterpart == "mail" {
				o, named = counterpartLabel(es, "mail"), true
			}
			rows = append(rows, integ{out, "jndi", o, c.Key, mode(lang, "jndi"), c.Component, ev(c), c.Confidence, "", c.Transport, c.Auth, "", named})
		}
	}
	for cp, a := range schemas {
		var ops []string
		for _, op := range []string{"read", "write", "uses"} {
			if a.ops[op] {
				ops = append(ops, op)
			}
		}
		key := fmt.Sprintf("%d %s (%s)", len(a.tables), es2(es, "tablas", "tables"), strings.Join(ops, "/"))
		rows = append(rows, integ{out, "db-schema", counterpartLabel(es, cp), key, mode(lang, "db-table"), "", a.ev, "medium", "", "", "", "", true})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.dir != b.dir {
			return a.dir < b.dir
		}
		if a.other != b.other {
			return a.other < b.other
		}
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		return a.key < b.key
	})
	return rows
}

func es2(es bool, a, b string) string {
	if es {
		return a
	}
	return b
}

// Arc42System renders the arc42 document of one system.
func Arc42System(f *model.Facts, p *model.Portfolio, o Options) string {
	b := &md{lang: o.Lang}
	b.p("# %s — %s\n", f.Repo.Name, b.t("Documentación de arquitectura (arc42)", "Architecture documentation (arc42)"))
	if o.Classification != "" {
		b.p("**%s**\n", o.Classification)
	}
	b.table([]string{b.t("Dato", "Item"), b.t("Valor", "Value")}, [][]string{
		{b.t("Sistema", "System"), f.Repo.Name},
		{b.t("Grupo", "Group"), f.Repo.Group},
		{b.t("Control de versiones", "Version control"), strings.TrimSpace(f.Repo.VCS + " " + f.Repo.Branch + " " + f.Repo.Revision)},
		{b.t("Generado", "Generated"), f.Repo.ScannedAt + " · Traverse " + f.Tool.Version + " · rules " + f.Tool.RulesVersion},
	})
	b.p("> %s\n", b.t(
		"Documento generado a partir del código con la plantilla [arc42](https://arc42.org). Las secciones marcadas con ✍️ requieren conocimiento del negocio o decisiones que no están en el código. Todo lo extraído es evidencia estática: valídalo antes de darlo por definitivo.",
		"Generated from source code using the [arc42](https://arc42.org) template. Sections marked ✍️ need business knowledge or decisions that are not in the code. Everything extracted is static evidence: validate it before treating it as final."))

	// ---- 1
	b.p("## 1. %s\n", b.t("Introducción y objetivos", "Introduction and goals"))
	b.todo("propósito del sistema, requisitos principales, objetivos de calidad y partes interesadas.", "purpose, key requirements, quality goals and stakeholders.")
	var langs []string
	code := 0
	for _, l := range f.Languages {
		if (l.Category == "code" || l.Category == "markup") && l.Code > 0 {
			code += l.Code
			if len(langs) < 6 {
				langs = append(langs, fmt.Sprintf("%s (%d)", l.Language, l.Code))
			}
		}
	}
	b.p("%s: %d %s · %s: %s · %s: %d · %s: %d.\n", b.t("Tamaño", "Size"), code, b.t("líneas de código", "lines of code"),
		b.t("lenguajes", "languages"), strings.Join(langs, ", "), b.t("módulos", "modules"), len(f.Modules), b.t("componentes", "components"), len(f.Components))
	if sv := f.Repo.Survey; sv != nil && len(structureLines(sv, o.Lang)) > 0 {
		b.p("**%s**\n", b.t("Alcance del análisis (estructura del repositorio)", "Analysis scope (repository structure)"))
		for _, l := range structureLines(sv, o.Lang) {
			b.p("- %s", l)
		}
		b.p("")
	}

	// ---- 2
	b.p("## 2. %s\n", b.t("Restricciones", "Constraints"))
	b.todo("restricciones organizativas, legales y de negocio.", "organisational, legal and business constraints.")
	b.p("### 2.1 %s\n", b.t("Restricciones técnicas detectadas", "Detected technical constraints"))
	var rows [][]string
	seen := map[string]bool{}
	for _, m := range f.Modules {
		if m.Runtime != "" && !seen[m.Runtime] {
			seen[m.Runtime] = true
			rows = append(rows, []string{b.t("Runtime", "Runtime"), m.Runtime, m.Descriptor})
		}
	}
	for _, fd := range f.Findings {
		if fd.SourceSet == "main" && (fd.Category == "Platform" || fd.Category == "Database") && !seen[fd.Technology] {
			seen[fd.Technology] = true
			rows = append(rows, []string{fd.Category, fd.Technology, fd.File})
		}
	}
	b.table([]string{b.t("Tipo", "Kind"), b.t("Restricción", "Constraint"), b.t("Evidencia", "Evidence")}, rows)

	// ---- 3
	integ := integrations(f, p, o.Lang)
	b.p("## 3. %s\n", b.t("Contexto y alcance", "Context and scope"))
	b.p("### 3.1 %s\n", b.t("Contexto de negocio", "Business context"))
	b.todo("usuarios, áreas y procesos de negocio a los que da servicio.", "users, business areas and processes it serves.")
	b.p("### 3.2 %s\n", b.t("Contexto técnico", "Technical context"))
	b.p("```mermaid\nflowchart LR")
	b.p("  %s[\"%s\"]", mid("SYS"), mlabel(f.Repo.Name))
	others := map[string]bool{}
	for _, r := range integ {
		node := r.other
		if strings.HasPrefix(node, b.t("externo", "external")) {
			node = b.t("Externo ", "External ") + strings.ToUpper(r.kind)
		}
		id := mid(node)
		if !others[id] {
			others[id] = true
			b.p("  %s[\"%s\"]", id, mlabel(node))
		}
	}
	edges := map[string]bool{}
	for _, r := range integ {
		node := r.other
		if strings.HasPrefix(node, b.t("externo", "external")) {
			node = b.t("Externo ", "External ") + strings.ToUpper(r.kind)
		}
		a, c := mid("SYS"), mid(node)
		if r.dir == b.t("entrante", "inbound") {
			a, c = c, a
		}
		arrow := "-->"
		if r.conf == "low" || strings.HasPrefix(r.kind, "shared") || r.kind == "library" {
			arrow = "-.->"
		}
		k := a + c + r.kind
		if !edges[k] {
			edges[k] = true
			b.p("  %s %s|\"%s\"| %s", a, arrow, mlabel(r.kind), c)
		}
	}
	b.p("```\n")
	b.p("### 3.3 %s\n", b.t("Catálogo de integraciones", "Integration catalog"))
	b.p("%s\n", b.t(
		"Cada fila es una integración con su evidencia en el código. **Dirección**: entrante (otros usan este sistema), saliente (este sistema usa otros) o compartida (acoplamiento por datos). **Modo**: síncrona, asíncrona, por datos compartidos, en compilación o despliegue.",
		"Each row is an integration with its evidence in the code. **Direction**: inbound (others use this system), outbound (this system uses others) or shared (coupling through data). **Mode**: synchronous, asynchronous, shared data, build time or deployment."))
	rows = nil
	for i, r := range integ {
		if i >= 400 {
			break
		}
		cp := r.other
		if r.host != "" {
			cp += " · " + r.host
		}
		rows = append(rows, []string{fmt.Sprint(i + 1), r.dir, r.kind, cp, r.key, r.mode, r.comp, r.evidence, r.conf})
	}
	b.table([]string{"#", b.t("Dirección", "Direction"), b.t("Tipo", "Kind"), b.t("Contraparte", "Counterpart"), b.t("Clave", "Key"),
		b.t("Modo", "Mode"), b.t("Componente", "Component"), b.t("Evidencia", "Evidence"), b.t("Confianza", "Confidence")}, rows)
	if len(integ) > 400 {
		b.p("_%s_\n", fmt.Sprintf(b.t("Se muestran 400 de %d integraciones; `contracts.csv` las contiene todas.", "Showing 400 of %d integrations; `contracts.csv` has all of them."), len(integ)))
	}
	summary := map[string]map[string]int{}
	for _, r := range integ {
		if summary[r.other] == nil {
			summary[r.other] = map[string]int{}
		}
		summary[r.other][r.kind]++
	}
	rows = nil
	for _, o2 := range sortedKeys(summary) {
		var ks []string
		for _, k := range sortedKeys(summary[o2]) {
			ks = append(ks, fmt.Sprintf("%s ×%d", k, summary[o2][k]))
		}
		rows = append(rows, []string{o2, strings.Join(ks, ", ")})
	}
	b.p("**%s**\n", b.t("Resumen por contraparte", "Summary by counterpart"))
	b.table([]string{b.t("Contraparte", "Counterpart"), b.t("Integraciones", "Integrations")}, rows)
	b.todo("acuerdos de nivel de servicio, volumetría, seguridad (autenticación, cifrado) y propietario de cada integración.", "service levels, volumes, security (authentication, encryption) and owner of each integration.")

	// ---- 4
	b.p("## 4. %s\n", b.t("Estrategia de solución", "Solution strategy"))
	b.todo("decisiones fundamentales y su motivación.", "fundamental decisions and their rationale.")
	techByCat := map[string][]string{}
	for _, fd := range f.Findings {
		if fd.SourceSet == "main" && fd.Technology != "" && fd.Category != "Security" {
			if !containsStr(techByCat[fd.Category], fd.Technology) {
				techByCat[fd.Category] = append(techByCat[fd.Category], fd.Technology)
			}
		}
	}
	rows = nil
	for _, c := range sortedKeys(techByCat) {
		sort.Strings(techByCat[c])
		rows = append(rows, []string{c, strings.Join(techByCat[c], ", ")})
	}
	b.p("%s:\n", b.t("Tecnologías detectadas", "Detected technologies"))
	b.table([]string{b.t("Área", "Area"), b.t("Tecnologías", "Technologies")}, rows)

	// ---- 5
	b.p("## 5. %s\n", b.t("Vista de bloques", "Building block view"))
	b.p("### 5.1 %s\n", b.t("Nivel 1: contenedores (módulos de build y unidades de despliegue)", "Level 1: containers (build modules and deployment units)"))
	depKind := map[string]model.Deployable{}
	for _, d := range f.Deployables {
		depKind[d.Module] = d
	}
	rows = nil
	for _, m := range f.Modules {
		d := depKind[m.ID]
		rows = append(rows, []string{m.Name, m.Build, m.Packaging, m.Runtime, strings.ToUpper(d.Kind), d.ContextRoot, m.Descriptor})
	}
	b.table([]string{b.t("Módulo", "Module"), "Build", b.t("Empaquetado", "Packaging"), "Runtime", b.t("Despliega", "Deploys"), "Context root", b.t("Descriptor", "Descriptor")}, rows)
	if len(f.Modules) > 1 {
		b.p("```mermaid\nflowchart LR")
		for _, m := range f.Modules {
			b.p("  %s[\"%s\"]", mid(m.ID), mlabel(m.Name))
		}
		for _, m := range f.Modules {
			for _, d := range m.Deps {
				for _, m2 := range f.Modules {
					if d.ID == m2.ID && m2.ID != m.ID && !d.Managed {
						b.p("  %s --> %s", mid(m.ID), mid(m2.ID))
					}
				}
			}
		}
		b.p("```\n")
	}
	b.p("### 5.2 %s\n", b.t("Nivel 2: componentes por capa", "Level 2: components by layer"))
	byLayer := map[string][]model.Component{}
	for _, c := range f.Components {
		if c.SourceSet == "main" && !strings.HasPrefix(c.Kind, "nested") {
			byLayer[c.Layer] = append(byLayer[c.Layer], c)
		}
	}
	rows = nil
	for _, l := range []string{"Presentation", "Business", "Integration", "Persistence", "Model", "Configuration", "Cross-cutting", "Unknown"} {
		cs := byLayer[l]
		if len(cs) == 0 {
			continue
		}
		var names []string
		for i, c := range cs {
			if i >= 12 {
				names = append(names, "…")
				break
			}
			names = append(names, c.Name)
		}
		rows = append(rows, []string{tr(o.Lang, l), fmt.Sprint(len(cs)), strings.Join(names, ", ")})
	}
	b.table([]string{b.t("Capa", "Layer"), b.t("Componentes", "Components"), b.t("Ejemplos", "Examples")}, rows)
	if mmd := MermaidLayers(f, 30, o.Lang); len(f.PackageStats) > 0 {
		b.p("```mermaid\n%s```\n", strings.TrimPrefix(mmd, "%% Generated by Traverse. Edit freely; regenerate to refresh.\n"))
	}
	b.p("%s: %d · %s: %d. %s\n", b.t("Ciclos de dependencias", "Dependency cycles"), len(f.Cycles), b.t("Violaciones de capas", "Layer violations"), len(f.LayerIssues),
		b.t("Detalle en `cycles.csv` y `layer-violations.csv`.", "Details in `cycles.csv` and `layer-violations.csv`."))
	b.todo("responsabilidad de cada bloque y sus interfaces internas relevantes.", "responsibility of each block and its relevant internal interfaces.")

	// ---- 6
	b.p("## 6. %s\n", b.t("Vista de ejecución", "Runtime view"))
	b.p("%s\n", b.t("Escenarios **derivados estáticamente**: desde cada punto de entrada se siguen las dependencias de código hasta las integraciones de salida alcanzables. No reflejan el orden real de ejecución ni las condiciones.",
		"Scenarios **derived statically**: from each entry point, code dependencies are followed to the reachable outbound integrations. They do not reflect actual execution order or conditions."))
	scen := scenarios(f, 15)
	if len(scen) == 0 {
		b.p("_%s_\n", b.t("No se ha detectado nada.", "Nothing detected."))
	}
	for _, s := range scen {
		b.p("- **%s** → %s", s[0], s[1])
	}
	b.p("")
	b.todo("escenarios de negocio principales, flujos de error y procesos batch con su planificación.", "main business scenarios, error flows and batch processes with their schedule.")

	// ---- 7
	b.p("## 7. %s\n", b.t("Vista de despliegue", "Deployment view"))
	rows = nil
	for _, d := range f.Deployables {
		rows = append(rows, []string{d.Module, strings.ToUpper(d.Kind), d.ContextRoot, d.RootSource, strings.Join(d.AppPaths, ", "), strings.Join(d.Includes, ", ")})
	}
	b.p("### 7.1 %s\n", b.t("Unidades de despliegue", "Deployment units"))
	b.table([]string{b.t("Módulo", "Module"), b.t("Tipo", "Kind"), "Context root", b.t("Origen", "Source"), b.t("Rutas de aplicación", "Application paths"), b.t("Incluye", "Includes")}, rows)
	b.p("### 7.2 %s\n", b.t("Infraestructura como código", "Infrastructure as code"))
	rows = nil
	for _, n := range f.Deployment.Nodes {
		rows = append(rows, []string{n.Platform, n.Kind, n.Name, strings.Join(append(append([]string{}, n.Images...), n.Base...), ", "),
			strings.Join(n.Ports, ", "), n.Replicas + n.Schedule, strings.Join(n.Hosts, ", "), n.Module, fmt.Sprintf("%s:%d", n.File, n.Line)})
	}
	b.table([]string{b.t("Plataforma", "Platform"), b.t("Tipo", "Kind"), b.t("Nombre", "Name"), b.t("Imágenes", "Images"), b.t("Puertos", "Ports"),
		b.t("Réplicas / planificación", "Replicas / schedule"), "Hosts", b.t("Módulo", "Module"), b.t("Evidencia", "Evidence")}, rows)
	if len(f.Deployment.Links) > 0 {
		b.p("```mermaid\nflowchart LR")
		for _, n := range f.Deployment.Nodes {
			b.p("  %s[\"%s\"]", mid(n.ID), mlabel(n.Name+" · "+n.Kind))
		}
		for _, l := range f.Deployment.Links {
			b.p("  %s -->|\"%s\"| %s", mid(l.From), mlabel(l.Kind), mid(l.To))
		}
		b.p("```\n")
	}
	platform := []string{}
	for _, c := range f.Contracts {
		if c.Kind == "platform-module" || (c.Kind == "jndi" && c.Direction == "uses") {
			platform = append(platform, c.Key)
		}
	}
	if len(platform) > 0 {
		sort.Strings(platform)
		b.p("%s: %s\n", b.t("Recursos que debe proporcionar el servidor (JNDI, módulos)", "Resources the server must provide (JNDI, modules)"), strings.Join(uniqueStrs(platform), ", "))
	}
	b.todo("entornos, nodos físicos o virtuales, redes, balanceadores y configuración por entorno.", "environments, physical or virtual nodes, networks, load balancers and per-environment configuration.")

	// ---- 8
	b.p("## 8. %s\n", b.t("Conceptos transversales", "Cross-cutting concepts"))
	concepts := map[string][]string{}
	for _, fd := range f.Findings {
		if fd.SourceSet != "main" || fd.Technology == "" {
			continue
		}
		switch fd.Category {
		case "Security", "Persistence", "Logging", "Scheduling", "Configuration", "Database":
			if !containsStr(concepts[fd.Category], fd.Technology) {
				concepts[fd.Category] = append(concepts[fd.Category], fd.Technology)
			}
		}
	}
	rows = nil
	for _, c := range sortedKeys(concepts) {
		rows = append(rows, []string{c, strings.Join(concepts[c], ", ")})
	}
	b.table([]string{b.t("Concepto", "Concept"), b.t("Mecanismos detectados", "Detected mechanisms")}, rows)
	if sec := buildSecurity(f, p, o.Lang); sec.Inbound != "" {
		b.p("**%s** %s.\n", b.t("mTLS entrante:", "Inbound mTLS:"), sec.Inbound)
	}
	b.todo("modelo de seguridad, gestión de errores, transacciones, auditoría, internacionalización.", "security model, error handling, transactions, auditing, internationalisation.")
	b.todo("requisitos de transporte impuestos por terceros (por ejemplo, un consumidor que exige mTLS) y el intermediario que los aplica (proxy, API gateway, balanceador).",
		"transport requirements imposed by third parties (for example, a consumer that requires mTLS) and the intermediary that enforces them (proxy, API gateway, load balancer).")

	// ---- 9
	b.p("## 9. %s\n", b.t("Decisiones de arquitectura", "Architecture decisions"))
	var adrs []string
	reADR := regexp.MustCompile(`(?i)(^|/)(adrs?|decisions?|architecture[-_ ]?decisions?)/[^/]+\.(md|adoc|txt|rst)$|(^|/)adr[-_]?\d+[^/]*\.(md|adoc)$`)
	for _, fl := range f.Files {
		if reADR.MatchString(fl.Path) {
			adrs = append(adrs, fl.Path)
		}
	}
	if len(adrs) > 0 {
		b.p("%s:\n", b.t("Registros de decisión (ADR) encontrados en el repositorio", "Decision records (ADR) found in the repository"))
		for _, a := range adrs {
			b.p("- `%s`", a)
		}
		b.p("")
	}
	b.todo("decisiones relevantes, alternativas consideradas y consecuencias (formato ADR recomendado).", "relevant decisions, alternatives and consequences (ADR format recommended).")

	// ---- 10
	b.p("## 10. %s\n", b.t("Requisitos de calidad", "Quality requirements"))
	b.todo("árbol de calidad y escenarios medibles.", "quality tree and measurable scenarios.")
	testLOC, mainLOC := 0, 0
	for _, fl := range f.Files {
		if fl.Category != "code" {
			continue
		}
		if fl.SourceSet == "test" {
			testLOC += fl.Code
		} else if fl.SourceSet == "main" {
			mainLOC += fl.Code
		}
	}
	unknown := len(byLayer["Unknown"])
	total := 0
	for _, cs := range byLayer {
		total += len(cs)
	}
	ratio := 0.0
	if mainLOC > 0 {
		ratio = float64(testLOC) / float64(mainLOC)
	}
	unk := 0.0
	if total > 0 {
		unk = 100 * float64(unknown) / float64(total)
	}
	b.p("%s:\n", b.t("Indicadores estáticos (orientativos)", "Static indicators (indicative)"))
	b.table([]string{b.t("Indicador", "Indicator"), b.t("Valor", "Value")}, [][]string{
		{b.t("Código de test / código principal", "Test code / main code"), fmt.Sprintf("%.2f (%d / %d LOC)", ratio, testLOC, mainLOC)},
		{b.t("Ciclos entre paquetes o módulos", "Package or module cycles"), fmt.Sprint(len(f.Cycles))},
		{b.t("Violaciones de capas", "Layer violations"), fmt.Sprint(len(f.LayerIssues))},
		{b.t("Componentes sin capa reconocible", "Components without a recognisable layer"), fmt.Sprintf("%.0f %%", unk)},
	})

	// ---- 11
	b.p("## 11. %s\n", b.t("Riesgos y deuda técnica", "Risks and technical debt"))
	rows = nil
	for _, fd := range f.Findings {
		if fd.Severity == "info" || fd.SourceSet != "main" {
			continue
		}
		if len(rows) >= 80 {
			break
		}
		rows = append(rows, []string{tr(o.Lang, fd.Severity), fd.Title, fd.Message, fmt.Sprintf("%s (%d)", fd.File, fd.Count), fd.RuleID})
	}
	b.table([]string{b.t("Severidad", "Severity"), b.t("Riesgo", "Risk"), b.t("Detalle", "Detail"), b.t("Evidencia", "Evidence"), b.t("Regla", "Rule")}, rows)
	if vs := f.VulnScan; vs != nil {
		b.p("### 11.1 %s\n", tr(o.Lang, "vulns"))
		b.p("%s (%s) · %s %s · %s: %d\n", tr(o.Lang, "vulnSource"), vs.Database, tr(o.Lang, "vulnAsOf"), vs.AsOf, tr(o.Lang, "vulnChecked"), vs.Checked)
		rows = nil
		for _, v := range f.Vulnerabilities {
			sev := tr(o.Lang, v.Severity)
			if v.Score > 0 {
				sev += " " + score(v.Score)
			}
			id := "[" + v.ID + "](" + OSVURL(v.ID) + ")"
			if len(v.Aliases) > 0 {
				id += " " + strings.Join(v.Aliases, ", ")
			}
			if v.Malicious {
				id += " ⚠ " + tr(o.Lang, "malicious")
			}
			rows = append(rows, []string{sev, id, v.Package + " " + v.Version, strings.Join(v.Fixed, ", "), v.Summary, v.Confidence, fileLine(v.File, v.Line)})
		}
		if len(rows) == 0 {
			b.p("%s\n", tr(o.Lang, "noVulns"))
		} else {
			b.table([]string{b.t("Severidad", "Severity"), "ID", b.t("Librería", "Library"), tr(o.Lang, "fixed"), tr(o.Lang, "summary"),
				b.t("Confianza", "Confidence"), b.t("Evidencia", "Evidence")}, rows)
		}
		if len(vs.Unevaluated) > 0 {
			b.p("**%s (%d)**\n", tr(o.Lang, "unevaluated"), len(vs.Unevaluated))
			rows = nil
			for _, u := range vs.Unevaluated {
				if len(rows) >= 60 {
					break
				}
				rows = append(rows, []string{u.Package, u.Version, tr(o.Lang, "why_"+u.Reason), u.File})
			}
			b.table([]string{b.t("Librería", "Library"), b.t("Versión declarada", "Declared version"), b.t("Motivo", "Reason"), b.t("Fichero", "File")}, rows)
		}
		b.p("_%s_\n", tr(o.Lang, "vulnNote"))
	}
	b.todo("impacto en el negocio, probabilidad y plan de mitigación de cada riesgo.", "business impact, likelihood and mitigation plan for each risk.")

	// ---- 12
	b.p("## 12. %s\n", b.t("Glosario", "Glossary"))
	b.p("%s:\n", b.t("Términos candidatos (entidades de dominio, tablas, transacciones)", "Candidate terms (domain entities, tables, transactions)"))
	terms := map[string]string{}
	for _, c := range f.Components {
		for _, s := range c.Stereotypes {
			if s == "Entity" || s == "Embeddable" {
				terms[c.Name] = b.t("Entidad", "Entity") + " (" + c.File + ")"
			}
		}
	}
	for _, c := range f.Contracts {
		switch c.Kind {
		case "db-table":
			if _, ok := terms[c.Match]; !ok {
				terms[c.Match] = b.t("Tabla", "Table")
			}
		case "cics-transaction":
			if c.Direction == "exposes" {
				terms[c.Key] = b.t("Transacción CICS", "CICS transaction") + " → " + c.Component
			}
		}
	}
	rows = nil
	for i, k := range sortedKeys(terms) {
		if i >= 80 {
			break
		}
		rows = append(rows, []string{k, terms[k], "✍️"})
	}
	b.table([]string{b.t("Término", "Term"), b.t("Origen", "Source"), b.t("Definición", "Definition")}, rows)
	return b.String()
}

func uniqueStrs(xs []string) []string {
	var out []string
	for i, x := range xs {
		if i == 0 || x != xs[i-1] {
			out = append(out, x)
		}
	}
	return out
}

// scenarios walks code dependencies from entry points (exposed contracts,
// subscriptions, batch jobs) to reachable outbound integrations.
func scenarios(f *model.Facts, max int) [][2]string {
	adj := map[string][]string{}
	for _, d := range f.Dependencies {
		adj[d.From] = append(adj[d.From], d.To)
	}
	outBy := map[string][]string{}
	for _, c := range f.Contracts {
		if c.Component == "" {
			continue
		}
		switch {
		case c.Direction == "calls" || c.Direction == "publishes" || (c.Direction == "uses" && (c.Kind == "jms" || c.Kind == "kafka")):
			outBy[c.Component] = append(outBy[c.Component], c.Kind+" "+c.Key)
		case c.Kind == "db-table" || c.Kind == "dataset" || c.Kind == "cics-file":
			outBy[c.Component] = append(outBy[c.Component], c.Kind+" "+c.Key+" ("+c.Direction+")")
		}
	}
	var out [][2]string
	seenEntry := map[string]bool{}
	entries := []model.Contract{}
	for _, c := range f.Contracts {
		if (c.Direction == "exposes" || c.Direction == "subscribes") && c.Component != "" && c.Kind != "program" {
			entries = append(entries, c)
		}
	}
	for _, c := range f.Components {
		if c.Kind == "job" {
			entries = append(entries, model.Contract{Kind: "batch job", Key: c.Name, Component: c.FQN})
		}
	}
	for _, e := range entries {
		if len(out) >= max || seenEntry[e.Component+e.Key] {
			continue
		}
		seenEntry[e.Component+e.Key] = true
		visited := map[string]bool{e.Component: true}
		queue := []string{e.Component}
		var path []string
		var targets []string
		for depth := 0; depth < 6 && len(queue) > 0; depth++ {
			var next []string
			for _, n := range queue {
				for _, t := range outBy[n] {
					if !containsStr(targets, t) {
						targets = append(targets, t)
					}
				}
				for _, m := range adj[n] {
					if !visited[m] {
						visited[m] = true
						next = append(next, m)
						if len(path) < 4 {
							path = append(path, shortName(m))
						}
					}
				}
			}
			queue = next
		}
		if len(targets) == 0 {
			continue
		}
		sort.Strings(targets)
		chain := shortName(e.Component)
		if len(path) > 0 {
			chain += " → " + strings.Join(path, " → ")
		}
		out = append(out, [2]string{e.Kind + " " + e.Key, chain + " ⇒ " + strings.Join(firstStrs(targets, 6), "; ")})
	}
	return out
}

func shortName(fqn string) string {
	if i := strings.LastIndexByte(fqn, '.'); i >= 0 {
		return fqn[i+1:]
	}
	return fqn
}

// Arc42Portfolio renders the portfolio-level document centred on integrations.
func Arc42Portfolio(all []*model.Facts, p *model.Portfolio, o Options) string {
	b := &md{lang: o.Lang}
	if o.Area != "" {
		b.p("# %s — %s\n", b.t("Portfolio de aplicaciones", "Application portfolio"), o.Area)
		b.p("> %s\n", b.t("Vista del área cortada del portfolio global: los sistemas de otras áreas solo aparecen si se relacionan con esta.",
			"Area view cut from the global portfolio: systems of other areas appear only when they are related to this one."))
	} else {
		b.p("# %s\n", b.t("Portfolio de aplicaciones — arquitectura e integraciones", "Application portfolio — architecture and integrations"))
	}
	if o.Classification != "" {
		b.p("**%s**\n", o.Classification)
	}
	b.p("> %s\n", b.t("Generado por Traverse a partir del código. Relaciones basadas en evidencias estáticas; las de confianza baja son candidatas a validar.",
		"Generated by Traverse from source code. Relationships are based on static evidence; low-confidence ones are candidates to validate."))
	b.p("## %s\n", b.t("Resumen ejecutivo", "Executive summary"))
	for _, line := range portfolioSummary(all, p, o.Lang) {
		b.p("- %s", line)
	}
	b.p("")
	b.p("## 1. %s\n", b.t("Catálogo de sistemas", "System catalog"))
	var rows [][]string
	for _, s := range p.Systems {
		rows = append(rows, []string{s.Name, s.Group, strings.Join(s.Languages, ", "), fmt.Sprint(s.Code), strings.Join(s.Runtimes, ", "),
			fmt.Sprint(s.Exposes), fmt.Sprint(s.Calls), fmt.Sprint(s.Tables), fmt.Sprintf("%d/%d/%d", s.Risks["high"], s.Risks["medium"], s.Risks["low"]),
			"[arc42](" + o.LinkBase + "systems/" + Slug(s.Name) + "/arc42.md)"})
	}
	b.table([]string{b.t("Sistema", "System"), b.t("Grupo", "Group"), b.t("Lenguajes", "Languages"), "LOC", "Runtime", b.t("Expone", "Exposes"),
		b.t("Consume", "Consumes"), b.t("Tablas", "Tables"), b.t("Riesgos A/M/B", "Risks H/M/L"), b.t("Documento", "Document")}, rows)
	b.p("## 2. %s\n", b.t("Mapa de integraciones", "Integration map"))
	dp := diagramPortfolio(p, o.MinConfidence)
	b.p("```mermaid\n%s```\n", strings.TrimPrefix(MermaidLandscape(dp), "%% Generated by Traverse\n"))
	b.p("## 3. %s\n", b.t("Matriz de integraciones", "Integration matrix"))
	b.p("%s\n", b.t("Filas: sistema origen (consumidor); columnas: sistema destino (proveedor).", "Rows: source (consumer); columns: target (provider)."))
	names := []string{}
	for _, s := range p.Systems {
		names = append(names, s.Name)
	}
	cellOf := map[string][]string{}
	for _, l := range p.Links {
		k := l.From + "→" + l.To
		cellOf[k] = append(cellOf[k], l.Kind)
	}
	rows = nil
	for _, a := range names {
		r := []string{"**" + a + "**"}
		for _, c := range names {
			r = append(r, strings.Join(cellOf[a+"→"+c], ", "))
		}
		rows = append(rows, r)
	}
	b.table(append([]string{""}, names...), rows)
	b.p("## 4. %s\n", b.t("Catálogo de integraciones entre sistemas", "Integration catalog between systems"))
	rows = nil
	for i, l := range p.Links {
		rows = append(rows, []string{fmt.Sprint(i + 1), l.From, l.To, l.Kind, mode(o.Lang, l.Kind), strings.Join(firstStrs(l.Keys, 8), "; "), fmt.Sprint(l.Count), linkConfidence(o.Lang == "es", l)})
	}
	b.table([]string{"#", b.t("Origen", "From"), b.t("Destino", "To"), b.t("Tipo", "Kind"), b.t("Modo", "Mode"), b.t("Claves", "Keys"), b.t("Evidencias", "Evidence"), b.t("Confianza", "Confidence")}, rows)
	b.todo("propietario, SLA, volumetría y seguridad de cada integración.", "owner, SLA, volume and security of each integration.")
	b.p("## 5. %s\n", b.t("Datos compartidos", "Shared data"))
	rows = nil
	for _, t := range p.SharedTables {
		rows = append(rows, []string{t.Table, t.Basis, strings.Join(t.Access, ", ")})
	}
	b.table([]string{b.t("Tabla", "Table"), b.t("Base de la evidencia", "Evidence basis"), b.t("Acceso por sistema", "Access by system")}, rows)
	b.p("## 6. %s\n", b.t("Librerías compartidas y versiones", "Shared libraries and versions"))
	rows = nil
	for _, l := range p.LibraryVersions {
		var vs []string
		for _, v := range sortedKeys(l.Versions) {
			vs = append(vs, v+": "+strings.Join(l.Versions[v], ", "))
		}
		internal := ""
		if l.Internal {
			internal = b.t("interna", "internal")
		}
		rows = append(rows, []string{l.Library, internal, strings.Join(vs, " · ")})
	}
	b.table([]string{b.t("Librería", "Library"), "", b.t("Versión → sistemas", "Version → systems")}, rows)
	b.p("## 7. %s\n", b.t("Dependencias fuera del portfolio", "Dependencies outside the portfolio"))
	rows = nil
	for _, e := range p.External {
		rows = append(rows, []string{e.Kind, e.Key, e.Host, strings.Join(e.Systems, ", ")})
	}
	b.table([]string{b.t("Tipo", "Kind"), b.t("Clave", "Key"), "Host", b.t("Sistemas", "Systems")}, rows)
	if o.HostsSealed {
		b.p("_%s_\n", tr(o.Lang, "hostsSealed"))
	}
	b.todo("identificar el sistema externo (organismo, proveedor) de cada dependencia.", "identify the external system (organisation, provider) of each dependency.")
	if p.VulnDB != "" {
		b.p("## 8. %s\n", tr(o.Lang, "vulns"))
		b.p("%s (%s) · %s %s\n", tr(o.Lang, "vulnSource"), p.VulnDB, tr(o.Lang, "vulnAsOf"), p.VulnAsOf)
		rows = nil
		for _, v := range p.Vulnerabilities {
			sev := tr(o.Lang, v.Severity)
			if v.Score > 0 {
				sev += " " + score(v.Score)
			}
			id := "[" + v.ID + "](" + OSVURL(v.ID) + ")"
			if len(v.Aliases) > 0 {
				id += " " + strings.Join(v.Aliases, ", ")
			}
			rows = append(rows, []string{sev, id, v.Package, strings.Join(v.Affected, ", "), strings.Join(v.Fixed, ", "), v.Summary})
		}
		if len(rows) == 0 {
			b.p("%s\n", tr(o.Lang, "noVulns"))
		} else {
			b.table([]string{b.t("Severidad", "Severity"), "ID", b.t("Librería", "Library"), tr(o.Lang, "affected"), tr(o.Lang, "fixed"), tr(o.Lang, "summary")}, rows)
		}
		b.todo("priorizar la actualización de cada librería y registrar las excepciones aceptadas.", "prioritise each library upgrade and record accepted exceptions.")
	}
	return b.String()
}

func pathOf(key string) string {
	if i := strings.IndexByte(key, ' '); i > 0 {
		return key[i+1:]
	}
	return key
}

func fileLine(f string, l int) string {
	if l > 0 {
		return fmt.Sprintf("%s:%d", f, l)
	}
	return f
}
