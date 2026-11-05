package report

import (
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// roleLabel renders the role of an inbound interface.
func roleLabel(lang, role string) string {
	switch role {
	case "api":
		return "API"
	case "ui":
		return es2(lang == "es", "pantalla", "screen")
	}
	return ""
}

// structureLines explains in plain words how the repository was read.
func structureLines(sv *model.Survey, lang string) []string {
	es := lang == "es"
	var out []string
	for _, r := range sv.Roots {
		s := es2(es, "Analizado: ", "Analyzed: ") + "`" + dispPath(r.Path) + "`"
		if r.Coord != "" {
			s += " (" + strings.TrimSpace(r.Coord+" "+r.Version) + ")"
		}
		if r.Main {
			s += es2(es, " — línea principal (", " — main line (") + mainReason(es, r.Reason) + ")"
		}
		out = append(out, s)
	}
	if n := len(sv.Copies); n > 0 {
		ident := 0
		for _, c := range sv.Copies {
			if c.Identical {
				ident++
			}
		}
		verb := es2(es, "omitidas", "left out")
		if sv.AllCopies {
			verb = es2(es, "analizadas también (--layout all)", "also analyzed (--layout all)")
		}
		out = append(out, es2(es, "Copias del mismo proyecto ", "Copies of the same project ")+verb+": "+itin(n)+
			es2(es, " (", " (")+itin(ident)+es2(es, " idénticas a la línea principal)", " identical to the main line)"))
	}
	for _, a := range sv.NonCode {
		if a.Kind == "other" {
			continue // images, fonts…: not worth a line
		}
		out = append(out, "`"+dispPath(a.Path)+"`: "+areaLabel(es, a.Kind)+" ("+itin(a.Files)+es2(es, " ficheros)", " files)"))
	}
	for _, bo := range sv.BinaryOnly {
		out = append(out, "`"+dispPath(bo.Path)+"`: "+es2(es, "módulo sin código fuente; se analiza el artefacto por dentro: ",
			"module without source code; the artifact is read from the inside: ")+strings.Join(bo.Artifacts, ", "))
	}
	return out
}

func dispPath(p string) string {
	if p == "" {
		return "."
	}
	return p
}

func mainReason(es bool, r string) string {
	switch r {
	case "trunk":
		return "trunk"
	case "highest-version":
		return es2(es, "versión más alta", "highest version")
	case "highest-version+newest":
		return es2(es, "versión más alta, la más reciente", "highest version, most recent")
	}
	return es2(es, "la más reciente", "most recent")
}

func areaLabel(es bool, k string) string {
	switch k {
	case "docs":
		return es2(es, "documentación (no es código)", "documentation (not code)")
	case "models":
		return es2(es, "modelos (no es código)", "models (not code)")
	case "binaries":
		return es2(es, "binarios", "binaries")
	case "data":
		return es2(es, "scripts SQL (modelo de datos, se analizan)", "SQL scripts (data model, analyzed)")
	case "runtime":
		return es2(es, "runtime incluido en el repositorio (JDK/servidor), omitido", "runtime committed to the repository (JDK/server), left out")
	}
	return es2(es, "sin código", "no code")
}

// ReportMarkdown renders the full static-analysis report of one system as a
// self-contained Markdown document: the same content as report.html, but
// readable and diff-friendly, and without arc42's "to complete" placeholders.
func ReportMarkdown(f *model.Facts, p *model.Portfolio, o Options) string {
	b := &md{lang: o.Lang}
	sys := f.Repo.Name
	b.p("# %s — %s\n", sys, b.t("Análisis de arquitectura", "Architecture analysis"))
	meta := []string{}
	if f.Repo.VCS != "" && f.Repo.VCS != "none" {
		meta = append(meta, f.Repo.VCS)
	}
	if f.Repo.Revision != "" {
		meta = append(meta, b.t("revisión ", "revision ")+f.Repo.Revision)
	}
	if f.Repo.Layout != "" {
		meta = append(meta, f.Repo.Layout)
	}
	meta = append(meta, b.t("generado ", "generated ")+f.Repo.ScannedAt, f.Tool.Name+" "+f.Tool.Version)
	b.p("_%s_\n", strings.Join(meta, " · "))
	if o.Classification != "" {
		b.p("> **%s**\n", o.Classification)
	}

	// Executive summary
	b.p("## %s\n", b.t("Resumen ejecutivo", "Executive summary"))
	for _, line := range summaryLines(f, p, o.Lang) {
		b.p("- %s", line)
	}
	b.p("")

	a := aggregate(f, p)

	// Repository structure (what was analyzed and what was left out)
	if sv := f.Repo.Survey; sv != nil {
		b.p("## %s\n", b.t("Estructura del repositorio", "Repository structure"))
		for _, line := range structureLines(sv, o.Lang) {
			b.p("- %s", line)
		}
		b.p("")
		if len(sv.Copies) > 0 {
			var crows [][]string
			for i, c := range sv.Copies {
				if i == 40 {
					crows = append(crows, []string{"…", b.t("(+", "(+") + itin(len(sv.Copies)-40) + ")", "", "", ""})
					break
				}
				rel := b.t("copia", "copy")
				if c.Identical {
					rel = b.t("idéntica", "identical")
				}
				crows = append(crows, []string{c.Path, c.Version, rel, itin(int(c.Overlap*100)) + "%", itin(c.Files)})
			}
			b.table([]string{b.t("Copia omitida", "Copy left out"), b.t("Versión", "Version"), b.t("Relación", "Relation"),
				b.t("Solape con la línea principal", "Overlap with main line"), b.t("Ficheros", "Files")}, crows)
		}
	}

	// Metrics
	b.p("## %s\n", b.t("Métricas", "Metrics"))
	metric := func(k, v string) { b.p("| %s | %s |", k, v) }
	b.p("| %s | %s |", b.t("Métrica", "Metric"), b.t("Valor", "Value"))
	b.p("|---|---|")
	metric(b.t("Líneas de código", "Lines of code"), thousands(a.code))
	metric(b.t("Ficheros", "Files"), itin(a.files))
	metric(b.t("Módulos de build", "Build modules"), itin(a.modules))
	metric(b.t("Componentes", "Components"), itin(a.components))
	metric(b.t("Interfaces expuestas", "Exposed interfaces"), itin(a.inbound))
	if a.apiIn+a.uiIn > 0 {
		metric(b.t("  · API para otros sistemas / pantallas", "  · API for other systems / screens"), itin(a.apiIn)+" / "+itin(a.uiIn))
	}
	metric(b.t("Llamadas salientes", "Outbound calls"), itin(a.outbound))
	metric(b.t("Tablas de datos", "Data tables"), itin(a.tables))
	metric(b.t("Recursos JNDI", "JNDI resources"), itin(a.jndi))
	metric(b.t("Ciclos de paquetes", "Package cycles"), itin(a.cycles))
	metric(b.t("Violaciones de capa", "Layer violations"), itin(a.layerIssues))
	metric(b.t("Hallazgos (alto/medio/bajo)", "Findings (high/medium/low)"),
		itin(a.findBySev["high"])+" / "+itin(a.findBySev["medium"])+" / "+itin(a.findBySev["low"]))
	if f.VulnScan != nil {
		metric(b.t("Vulnerabilidades (críticas/altas)", "Vulnerabilities (critical/high)"),
			itin(a.vulnBySev["critical"])+" / "+itin(a.vulnBySev["high"]))
	}
	b.p("")

	// Languages
	b.p("## %s\n", b.t("Lenguajes", "Languages"))
	langs := append([]model.LangStat(nil), f.Languages...)
	sort.Slice(langs, func(i, j int) bool { return langs[i].Code > langs[j].Code })
	var lrows [][]string
	for _, l := range langs {
		if l.Code == 0 && l.Files == 0 {
			continue
		}
		lrows = append(lrows, []string{l.Language, l.Category, itin(l.Files), thousands(l.Code)})
	}
	b.table([]string{b.t("Lenguaje", "Language"), b.t("Categoría", "Category"), b.t("Ficheros", "Files"), b.t("Código", "Code")}, lrows)

	// Technologies
	if p != nil {
		b.p("## %s\n", b.t("Tecnologías", "Technologies"))
		type tech struct {
			cat, name, sev string
			files          int
		}
		var ts []tech
		for _, t := range p.Technologies {
			if n := t.Systems[sys]; n > 0 {
				ts = append(ts, tech{t.Category, t.Technology, t.Severity, n})
			}
		}
		sort.Slice(ts, func(i, j int) bool {
			if ts[i].cat != ts[j].cat {
				return ts[i].cat < ts[j].cat
			}
			return ts[i].name < ts[j].name
		})
		var trows [][]string
		for _, t := range ts {
			sev := t.sev
			if sev == "info" {
				sev = "—"
			}
			trows = append(trows, []string{t.cat, t.name, sev, itin(t.files)})
		}
		b.table([]string{b.t("Área", "Area"), b.t("Tecnología", "Technology"), b.t("Severidad", "Severity"), b.t("Ficheros", "Files")}, trows)
	}

	// Integrations
	b.p("## %s\n", b.t("Integraciones", "Integrations"))
	rows := integrations(f, p, o.Lang)
	var irows [][]string
	for i, r := range rows {
		other := r.other
		if r.host != "" {
			other += " · " + r.host
		}
		irows = append(irows, []string{itin(i + 1), r.dir, r.kind, roleLabel(o.Lang, r.role), other, r.key, r.mode, transportLabel(o.Lang, r.transport), r.auth, shortName(r.comp), r.evidence, r.conf})
	}
	b.table([]string{"#", b.t("Dirección", "Direction"), b.t("Tipo", "Type"), b.t("Uso", "Use"), b.t("Contraparte", "Counterpart"),
		b.t("Clave", "Key"), b.t("Modo", "Mode"), b.t("Transporte", "Transport"), b.t("Auth", "Auth"), b.t("Componente", "Component"), b.t("Evidencia", "Evidence"), b.t("Confianza", "Confidence")}, irows)

	// Authentication & transport security
	sec := buildSecurity(f, p, o.Lang)
	b.p("## %s\n", b.t("Autenticación y seguridad de transporte", "Authentication and transport security"))
	if sec.Inbound != "" {
		b.p("- **%s** %s.\n", b.t("mTLS entrante:", "Inbound mTLS:"), sec.Inbound)
	}
	if sec.NoUserAuth {
		b.p("- **%s** %s\n", b.t("Login de usuarios:", "User login:"),
			b.t("no visible en el código (posiblemente delegado en la infraestructura: pasarela, SSO o servidor).",
				"not visible in the code (possibly delegated to the infrastructure: gateway, SSO or server)."))
	}
	var mrows [][]string
	for _, m := range sec.Mechanisms {
		name := m.Mechanism
		if m.Detail != "" {
			name += " (" + m.Detail + ")"
		}
		mrows = append(mrows, []string{m.KindLabel, name, itin(m.Files), m.Evidence})
	}
	b.table([]string{b.t("Tipo", "Kind"), b.t("Mecanismo", "Mechanism"), b.t("Ficheros", "Files"), b.t("Evidencia", "Evidence")}, mrows)
	if sec.MTLS {
		b.p("- %s\n", b.t("**mTLS / certificado de cliente** detectado.", "**mTLS / client certificate** detected."))
	}
	if len(sec.Weak) > 0 {
		b.p("**%s**\n", b.t("Puntos débiles", "Weak spots"))
		for _, w := range sec.Weak {
			b.p("- %s", w)
		}
		b.p("")
	}

	// Who it calls, who calls it
	pv := buildParties(f, p, o.Lang)
	es := o.Lang == "es"
	b.p("## %s\n", b.t("Relaciones con otras aplicaciones", "Relationships with other applications"))
	b.p("### %s (%d)\n", b.t("A quién llama", "What it calls"), len(pv.Calls))
	if len(pv.Calls) == 0 {
		b.p("%s\n", b.t("No se ha encontrado ninguna llamada saliente en el código ni en la configuración del repositorio.",
			"No outbound call found in the code or the repository configuration."))
	} else {
		var rows [][]string
		for _, r := range pv.Calls {
			rows = append(rows, []string{r.Counterpart, r.Scope, r.Protocols, itin(r.Endpoints), transportLabel(o.Lang, r.Transport), r.Auth, r.Envs, r.Evidence})
		}
		b.table([]string{b.t("Aplicación", "Application"), b.t("Ámbito", "Scope"), b.t("Protocolos", "Protocols"), b.t("Endpoints", "Endpoints"),
			b.t("Transporte", "Transport"), b.t("Auth", "Auth"), b.t("Entornos", "Environments"), b.t("Evidencia", "Evidence")}, rows)
	}
	b.p("### %s\n", b.t("Quién le llama", "Who calls it"))
	if len(pv.CalledBy) == 0 {
		b.p("- %s", b.t("Ningún sistema analizado le llama.", "No analyzed system calls it."))
	} else {
		var rows [][]string
		for _, r := range pv.CalledBy {
			rows = append(rows, []string{r.Counterpart, r.Protocols, itin(r.Endpoints), r.Evidence})
		}
		b.table([]string{b.t("Aplicación", "Application"), b.t("Protocolos", "Protocols"), b.t("Interfaces", "Interfaces"), b.t("Evidencia", "Evidence")}, rows)
	}
	if len(pv.SharesWith) > 0 {
		var names []string
		for _, r := range pv.SharesWith {
			names = append(names, r.Counterpart)
		}
		b.p("- %s %s", b.t("**Comparte datos con:**", "**Shares data with:**"), strings.Join(names, ", "))
	}
	if n := pv.UnknownAPI + pv.UnknownUI + pv.UnknownAny; n > 0 {
		b.p("- %s %d API · %d %s · %d %s. %s", b.t("**Sin consumidor identificado:**", "**No identified consumer:**"),
			pv.UnknownAPI, pv.UnknownUI, b.t("pantallas", "screens"), pv.UnknownAny, b.t("otras", "other"),
			b.t("Quién consume una interfaz solo se sabe si el consumidor también se analiza.",
				"Who consumes an interface is only known when the consumer is analyzed too."))
	}
	b.p("")
	if len(pv.Registered) > 0 {
		b.p("**%s**\n", b.t("Certificados de aplicaciones o servidores de confianza (clientes registrados o pares mTLS)",
			"Certificates of trusted applications or servers (registered clients or mTLS peers)"))
		var rows [][]string
		for _, c := range pv.Registered {
			src := c.File
			if c.Alias != "" {
				src += " (" + c.Alias + ")"
			}
			rows = append(rows, []string{certLabel(es, c), c.Issuer, c.NotAfter, certStatus(es, c.Status), src})
		}
		b.table([]string{b.t("Certificado", "Certificate"), b.t("Emisor", "Issuer"), b.t("Válido hasta", "Valid until"), b.t("Estado", "Status"), b.t("Origen", "Source")}, rows)
	}
	if len(pv.Identities) > 0 {
		b.p("**%s**\n", b.t("Identidad propia (certificados con clave privada)", "Own identity (certificates with private key)"))
		var rows [][]string
		for _, c := range pv.Identities {
			rows = append(rows, []string{certLabel(es, c), c.Issuer, c.NotAfter, certStatus(es, c.Status), c.File})
		}
		b.table([]string{b.t("Certificado", "Certificate"), b.t("Emisor", "Issuer"), b.t("Válido hasta", "Valid until"), b.t("Estado", "Status"), b.t("Fichero", "File")}, rows)
	}
	if len(pv.TrustedCAs) > 0 {
		b.p("- **%s** %s\n", b.t("Autoridades de certificación de confianza:", "Trusted certification authorities:"), strings.Join(pv.TrustedCAs, "; "))
	}

	// Third-party systems inventory
	b.p("## %s\n", b.t("Inventario de aplicaciones conectadas (terceros)", "Connected applications inventory (third parties)"))
	var xrows [][]string
	for _, e := range sec.Externals {
		hosts := ""
		if e.Hosts > 0 {
			hosts = itin(e.Hosts)
		}
		xrows = append(xrows, []string{e.Counterpart, e.Scope, e.Direction, e.Protocol, transportLabel(o.Lang, e.Transport), e.Auth, e.Endpoint, e.Envs, hosts, e.Evidence, e.Confidence})
	}
	b.table([]string{b.t("Contraparte", "Counterpart"), b.t("Ámbito", "Scope"), b.t("Dirección", "Direction"), b.t("Protocolo", "Protocol"),
		b.t("Transporte", "Transport"), b.t("Auth", "Auth"), b.t("Endpoint", "Endpoint"), b.t("Entornos", "Environments"), b.t("Hosts", "Hosts"), b.t("Evidencia", "Evidence"), b.t("Confianza", "Confidence")}, xrows)

	// Data access
	b.p("## %s\n", b.t("Acceso a datos", "Data access"))
	type tbl struct {
		read, write bool
		comps       map[string]bool
		files       map[string]bool
	}
	tables := map[string]*tbl{}
	var tnames []string
	for _, c := range f.Contracts {
		if c.Kind != "db-table" {
			continue
		}
		t := tables[c.Key]
		if t == nil {
			t = &tbl{comps: map[string]bool{}, files: map[string]bool{}}
			tables[c.Key] = t
			tnames = append(tnames, c.Key)
		}
		if c.Direction == "write" {
			t.write = true
		} else {
			t.read = true
		}
		if c.Component != "" {
			t.comps[shortName(c.Component)] = true
		}
		t.files[c.File] = true
	}
	sort.Strings(tnames)
	var drows [][]string
	for _, n := range tnames {
		t := tables[n]
		acc := b.t("lectura", "read")
		switch {
		case t.read && t.write:
			acc = b.t("lectura/escritura", "read/write")
		case t.write:
			acc = b.t("escritura", "write")
		}
		drows = append(drows, []string{n, acc, strings.Join(keysOf(t.comps), ", "), itin(len(t.files))})
	}
	b.table([]string{b.t("Tabla", "Table"), b.t("Acceso", "Access"), b.t("Componentes", "Components"), b.t("Ficheros", "Files")}, drows)

	// Deployment
	b.p("## %s\n", b.t("Despliegue", "Deployment"))
	var deprows [][]string
	for _, d := range f.Deployables {
		deprows = append(deprows, []string{shortName(d.Module), d.Kind, d.ContextRoot, d.RootSource})
	}
	b.table([]string{b.t("Módulo", "Module"), b.t("Tipo", "Kind"), "Context root", b.t("Origen", "Source")}, deprows)
	if len(f.Deployment.Nodes) > 0 {
		var nrows [][]string
		for _, n := range f.Deployment.Nodes {
			nrows = append(nrows, []string{n.Name, n.Platform, n.Kind, strings.Join(n.Images, ", "), fileLine(n.File, n.Line)})
		}
		b.table([]string{b.t("Nodo", "Node"), b.t("Plataforma", "Platform"), b.t("Tipo", "Kind"), b.t("Imágenes", "Images"), b.t("Evidencia", "Evidence")}, nrows)
	}

	// Vulnerabilities
	if f.VulnScan != nil {
		b.p("## %s\n", tr(o.Lang, "vulns"))
		b.p("%s (%s) · %s %s · %s: %d\n", tr(o.Lang, "vulnSource"), f.VulnScan.Database, tr(o.Lang, "vulnAsOf"), f.VulnScan.AsOf, tr(o.Lang, "vulnChecked"), f.VulnScan.Checked)
		var vrows [][]string
		for _, v := range f.Vulnerabilities {
			sev := tr(o.Lang, v.Severity)
			if v.Score > 0 {
				sev += " " + score(v.Score)
			}
			id := "[" + v.ID + "](" + OSVURL(v.ID) + ")"
			if v.Malicious {
				id += " ⚠"
			}
			vrows = append(vrows, []string{sev, id, v.Package + " " + v.Version, strings.Join(v.Fixed, ", "), v.Confidence, v.Summary})
		}
		if len(vrows) == 0 {
			b.p("%s\n", tr(o.Lang, "noVulns"))
		} else {
			b.table([]string{b.t("Severidad", "Severity"), "ID", b.t("Librería", "Library"), tr(o.Lang, "fixed"), b.t("Confianza", "Confidence"), tr(o.Lang, "summary")}, vrows)
		}
		if len(f.VulnScan.Unevaluated) > 0 {
			b.p("**%s (%d)**\n", tr(o.Lang, "unevaluated"), len(f.VulnScan.Unevaluated))
			var urows [][]string
			for i, u := range f.VulnScan.Unevaluated {
				if i >= 50 {
					break
				}
				urows = append(urows, []string{u.Package, u.Version, tr(o.Lang, "why_"+u.Reason)})
			}
			b.table([]string{b.t("Librería", "Library"), b.t("Versión", "Version"), b.t("Motivo", "Reason")}, urows)
		}
	}

	// Risks / findings
	b.p("## %s\n", b.t("Riesgos y hallazgos", "Risks and findings"))
	type ruleAgg struct {
		title, sev, cat string
		files, count    int
	}
	byRule := map[string]*ruleAgg{}
	var ruleIDs []string
	for _, fd := range f.Findings {
		if fd.Severity == "info" {
			continue
		}
		r := byRule[fd.RuleID]
		if r == nil {
			r = &ruleAgg{title: fd.Title, sev: fd.Severity, cat: fd.Category}
			byRule[fd.RuleID] = r
			ruleIDs = append(ruleIDs, fd.RuleID)
		}
		r.files++
		r.count += fd.Count
	}
	sort.Slice(ruleIDs, func(i, j int) bool {
		a, b := byRule[ruleIDs[i]], byRule[ruleIDs[j]]
		if ra, rb := sevOrder(a.sev), sevOrder(b.sev); ra != rb {
			return ra > rb
		}
		if a.files != b.files {
			return a.files > b.files
		}
		return ruleIDs[i] < ruleIDs[j]
	})
	var frows [][]string
	for _, id := range ruleIDs {
		r := byRule[id]
		frows = append(frows, []string{tr(o.Lang, r.sev), r.title, r.cat, itin(r.files)})
	}
	b.table([]string{b.t("Severidad", "Severity"), b.t("Hallazgo", "Finding"), b.t("Categoría", "Category"), b.t("Ficheros", "Files")}, frows)

	// Coupling
	b.p("## %s\n", b.t("Acoplamiento", "Coupling"))
	if len(f.Cycles) > 0 {
		b.p("**%s**\n", b.t("Ciclos de dependencias", "Dependency cycles"))
		var crows [][]string
		for _, c := range f.Cycles {
			members := strings.Join(c.Members, ", ")
			if len(members) > 300 {
				members = members[:300] + "…"
			}
			crows = append(crows, []string{c.Level, itin(len(c.Members)), members})
		}
		b.table([]string{b.t("Nivel", "Level"), b.t("Tamaño", "Size"), b.t("Miembros", "Members")}, crows)
	}
	if len(f.LayerIssues) > 0 {
		pair := map[string]int{}
		var pk []string
		for _, v := range f.LayerIssues {
			k := v.FromLayer + " → " + v.ToLayer
			if pair[k] == 0 {
				pk = append(pk, k)
			}
			pair[k]++
		}
		sort.Slice(pk, func(i, j int) bool {
			if pair[pk[i]] != pair[pk[j]] {
				return pair[pk[i]] > pair[pk[j]]
			}
			return pk[i] < pk[j]
		})
		var prows [][]string
		for _, k := range pk {
			prows = append(prows, []string{k, itin(pair[k])})
		}
		b.p("**%s**\n", b.t("Violaciones de capa", "Layer violations"))
		b.table([]string{b.t("De → a", "From → to"), b.t("Casos", "Cases")}, prows)
	}

	// Diagnostics
	if len(f.Diagnostics) > 0 {
		b.p("## %s\n", b.t("Diagnósticos", "Diagnostics"))
		var grows [][]string
		for _, d := range f.Diagnostics {
			grows = append(grows, []string{d.Kind, d.File, d.Message})
		}
		b.table([]string{b.t("Tipo", "Kind"), b.t("Fichero", "File"), b.t("Mensaje", "Message")}, grows)
	}

	b.p("---\n_%s %s._",
		b.t("Documento generado por análisis estático; sin ejecutar el código. Formatos abiertos acompañantes: arc42.md, CycloneDX, SARIF, CSV y diagramas (C4, ArchiMate, Mermaid).",
			"Generated by static analysis; the code is never executed. Companion open formats: arc42.md, CycloneDX, SARIF, CSV and diagrams (C4, ArchiMate, Mermaid)."),
		"")
	return b.String()
}

func itin(n int) string { return thousands(n) }

func keysOf(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sevOrder(s string) int {
	switch s {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	}
	return 1
}
