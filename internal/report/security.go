package report

import (
	"path"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/survey"

	"github.com/oskillarrr/traverse/internal/model"
)

type authRow struct {
	Mechanism string
	Category  string // Authentication | Transport Security
	Kind      string // user | service | signature | transport
	KindLabel string
	Severity  string
	Files     int
	Evidence  string
	Detail    string // e.g. custom provider class names
}

type extRow struct {
	Counterpart string
	Scope       string // portfolio | external
	Direction   string
	Protocol    string
	Transport   string
	Auth        string
	Endpoint    string
	Host        string
	Hosts       int    // distinct (encrypted) hosts seen for this integration
	Envs        string // environments whose configuration declares it (int, pre, pro…)
	Evidence    string
	Confidence  string
	hostSet     map[string]bool
	envSet      map[string]bool
}

type securityView struct {
	Mechanisms []authRow // authentication & transport mechanisms detected
	NoUserAuth bool      // no evidence of user login in the code
	MTLS       bool
	// Inbound says how inbound mutual TLS shows in the code, as one sentence:
	// required by the server itself, inferred from a certificate received in
	// a header (an intermediary terminates it), or not visible.
	Inbound      string
	UpstreamAPIs int      // inbound endpoints that receive the client certificate in a header
	Weak         []string // weak spots: plaintext outbound, disabled verification, weak TLS
	Externals    []extRow // third-party systems inventory
}

// mtlsTech marks technologies that imply mutual TLS / client certificates.
func isMTLS(tech string) bool {
	t := strings.ToLower(tech)
	return strings.Contains(t, "client auth") || strings.Contains(t, "mtls") || strings.Contains(t, "mutual")
}

func buildSecurity(f *model.Facts, p *model.Portfolio, lang string) securityView {
	es := lang == "es"
	var sv securityView
	// 1. mechanisms from auth/transport findings, grouped by kind
	type agg struct {
		cat, sev, ev, kind string
		evRank             int
		files              int
		classes            []string
	}
	byTech := map[string]*agg{}
	var order []string
	for _, fd := range f.Findings {
		if fd.AuthKind == "" && fd.Category != "Authentication" && fd.Category != "Transport Security" {
			continue
		}
		key := fd.Technology
		if key == "" {
			key = fd.Title
		}
		kind := fd.AuthKind
		if kind == "" {
			kind = "transport"
		}
		k2 := kind + "\x00" + key
		ev := fd.File
		if len(fd.Lines) > 0 && fd.Lines[0] > 0 {
			ev = fileLine(fd.File, fd.Lines[0])
		}
		rank := evidenceRank(fd.File)
		a := byTech[k2]
		if a == nil {
			a = &agg{cat: fd.Category, sev: fd.Severity, ev: ev, evRank: rank, kind: kind}
			byTech[k2] = a
			order = append(order, k2)
		} else if rank < a.evRank {
			a.ev, a.evRank = ev, rank // code beats configuration, which beats logging setup
		}
		a.files += fd.Count
		if fd.RuleID == "AUTH-CUSTOM-PROVIDER" {
			a.classes = appendUniq(a.classes, strings.TrimSuffix(path.Base(fd.File), path.Ext(fd.File)))
		}
		if isMTLS(key) {
			sv.MTLS = true
		}
	}
	kindOrder := map[string]int{"user": 0, "service": 1, "signature": 2, "transport": 3}
	sort.SliceStable(order, func(i, j int) bool {
		ai, aj := byTech[order[i]], byTech[order[j]]
		if ai.kind != aj.kind {
			return kindOrder[ai.kind] < kindOrder[aj.kind]
		}
		return order[i] < order[j]
	})
	hasUser := false
	for _, k := range order {
		a := byTech[k]
		name := k[strings.IndexByte(k, 0)+1:]
		if a.kind == "user" {
			hasUser = true
		}
		sort.Strings(a.classes)
		sv.Mechanisms = append(sv.Mechanisms, authRow{Mechanism: name, Category: a.cat, Kind: a.kind, KindLabel: authKindLabel(lang, a.kind),
			Severity: a.sev, Files: a.files, Evidence: a.ev, Detail: strings.Join(a.classes, ", ")})
	}
	exposes := false
	for _, c := range f.Contracts {
		if c.Direction == "exposes" && c.Kind != "program" {
			exposes = true
			break
		}
	}
	sv.NoUserAuth = !hasUser && exposes // a library or batch has no login to show
	sv.Inbound, sv.UpstreamAPIs = inboundMTLS(f, exposes, es)
	// 2. weak spots
	plain := 0
	soapNoWSS := false
	for _, c := range f.Contracts {
		if c.Kind == "http" && (c.Direction == "calls") && c.Transport == "plain" {
			plain++
		}
	}
	hasWSS := false
	for _, fd := range f.Findings {
		if fd.Technology == "WS-Security" {
			hasWSS = true
		}
		if fd.Technology == "Insecure TLS" {
			sv.Weak = append(sv.Weak, es2(es, "Verificación TLS deshabilitada: "+fileLineList(fd), "TLS verification disabled: "+fileLineList(fd)))
		}
		if fd.Technology == "Weak TLS" {
			sv.Weak = append(sv.Weak, es2(es, "Protocolo TLS obsoleto: "+fileLineList(fd), "Obsolete TLS protocol: "+fileLineList(fd)))
		}
	}
	for _, c := range f.Contracts {
		if c.Kind == "soap" && (c.Direction == "exposes" || c.Direction == "calls") && !hasWSS {
			soapNoWSS = true
			break
		}
	}
	if plain > 0 {
		sv.Weak = append(sv.Weak, es2(es,
			plural(plain, "llamada saliente por HTTP sin cifrar", "llamadas salientes por HTTP sin cifrar"),
			plural(plain, "outbound call over plaintext HTTP", "outbound calls over plaintext HTTP")))
	}
	if soapNoWSS {
		sv.Weak = append(sv.Weak, es2(es, "Servicios SOAP sin WS-Security detectada", "SOAP services with no WS-Security detected"))
	}
	// 3. third-party inventory from the integration catalog: one row per
	// counterpart and endpoint, with the environments and hosts that declare it
	rows := integrations(f, p, lang)
	seen := map[string]*extRow{}
	var eorder []string
	extLabel := es2(es, "externo (fuera del portfolio)", "external (outside the portfolio)")
	selfLabel := counterpartLabel(es, "self")
	portfolioSystems := map[string]bool{}
	if p != nil {
		for _, s := range p.Systems {
			portfolioSystems[s.Name] = true
		}
	}
	for _, r := range rows {
		if r.other == es2(es, "(consumidor no identificado)", "(no identified consumer)") ||
			r.other == es2(es, "recurso del servidor", "server resource") || r.kind == "library" || r.other == selfLabel {
			continue
		}
		cp := r.other
		if strings.HasPrefix(cp, extLabel) && r.host != "" {
			cp = r.host // nothing in the code names it: the encrypted host is the only identity
		}
		scope := es2(es, "del portfolio", "portfolio")
		if !portfolioSystems[r.other] {
			scope = es2(es, "externo", "external")
		}
		ep := r.key
		key := cp + "|" + r.kind + "|" + r.dir + "|" + ep
		x := seen[key]
		if x == nil {
			x = &extRow{Counterpart: cp, Scope: scope, Direction: r.dir, Protocol: r.kind, Transport: r.transport, Auth: r.auth,
				Endpoint: ep, Host: r.host, Evidence: r.evidence, Confidence: r.conf, hostSet: map[string]bool{}, envSet: map[string]bool{}}
			seen[key] = x
			eorder = append(eorder, key)
		} else {
			if x.Transport == "" || r.transport == "plain" {
				x.Transport = r.transport // a plaintext variant must not be hidden
			}
			if x.Auth == "" {
				x.Auth = r.auth
			}
		}
		if r.host != "" {
			x.hostSet[r.host] = true
		}
		if e := survey.EnvOf(r.evidence); e != "" {
			x.envSet[e] = true
		}
	}
	for _, k := range eorder {
		x := seen[k]
		x.Hosts = len(x.hostSet)
		var envs []string
		for e := range x.envSet {
			envs = append(envs, e)
		}
		sort.Strings(envs)
		x.Envs = strings.Join(envs, ", ")
	}
	sort.Strings(eorder)
	for _, k := range eorder {
		sv.Externals = append(sv.Externals, *seen[k])
	}
	return sv
}

// serverMTLSRules are the rules showing that the application server itself
// asks callers for a client certificate.
var serverMTLSRules = map[string]bool{"AUTH-X509": true, "TLS-SPRING-BOOT-MTLS": true, "TLS-LIBERTY-MTLS": true,
	"DESC-WEB-MTLS": true, "DESC-TOMCAT-MTLS": true, "DESC-WILDFLY-MTLS": true, "DESC-WILDFLY-REALM-MTLS": true}

func inboundMTLS(f *model.Facts, exposes, es bool) (string, int) {
	if !exposes {
		return "", 0
	}
	server, upstream := false, false
	for _, fd := range f.Findings {
		server = server || serverMTLSRules[fd.RuleID]
		upstream = upstream || strings.HasPrefix(fd.RuleID, "TLS-UPSTREAM-")
	}
	n := 0
	for _, c := range f.Contracts {
		if c.Direction == "exposes" && c.Transport == "mtls-upstream" {
			n++
		}
	}
	var parts []string
	if server {
		parts = append(parts, es2(es, "lo exige el propio servidor de aplicaciones (ver mecanismos de transporte)",
			"required by the application server itself (see transport mechanisms)"))
	}
	if upstream {
		what := es2(es, "la aplicación recibe el certificado de cliente en una cabecera HTTP",
			"the application receives the client certificate in an HTTP header")
		if n > 0 {
			what = es2(es, plural(n, "interfaz recibe", "interfaces reciben")+" el certificado de cliente en una cabecera HTTP",
				plural(n, "interface receives", "interfaces receive")+" the client certificate in an HTTP header")
		}
		parts = append(parts, es2(es, "inferido: "+what+", señal de que un intermediario (proxy, API gateway, balanceador) termina el mTLS. Quién lo exige y qué consumidores lo usan no está en el código",
			"inferred: "+what+", a sign that an intermediary (proxy, API gateway, load balancer) terminates mTLS. Who requires it and which consumers use it is not in the code"))
	}
	if len(parts) == 0 {
		return es2(es, "no visible en el código. Si un intermediario (proxy, API gateway, balanceador) termina el TLS, puede exigir certificado de cliente sin que la aplicación lo vea",
			"not visible in the code. If an intermediary (proxy, API gateway, load balancer) terminates TLS, it may require a client certificate without the application seeing it"), n
	}
	return strings.Join(parts, "; "), n
}

// evidenceRank prefers source code over configuration, and configuration
// over logging setup (a logger category naming a library is weak evidence).
func evidenceRank(file string) int {
	low := strings.ToLower(path.Base(file))
	switch {
	case strings.HasPrefix(low, "log4j") || strings.HasPrefix(low, "logback") || strings.Contains(low, "logging"):
		return 3
	case strings.HasSuffix(low, ".java") || strings.HasSuffix(low, ".cs") || strings.HasSuffix(low, ".kt") ||
		strings.HasSuffix(low, ".jsp") || strings.HasSuffix(low, ".js") || strings.HasSuffix(low, ".ts"):
		return 0
	case strings.HasSuffix(low, ".xml") || strings.HasSuffix(low, ".properties") || strings.HasSuffix(low, ".yml") || strings.HasSuffix(low, ".yaml"):
		return 1
	}
	return 2
}

func authKindLabel(lang, k string) string {
	es := lang == "es"
	switch k {
	case "user":
		return es2(es, "Login de usuarios", "User login")
	case "service":
		return es2(es, "Entre servicios", "Service to service")
	case "signature":
		return es2(es, "Firma electrónica", "E-signature")
	}
	return es2(es, "Transporte (TLS/mTLS)", "Transport (TLS/mTLS)")
}

func appendUniq(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}

func fileLineList(fd model.Finding) string {
	if len(fd.Lines) > 0 && fd.Lines[0] > 0 {
		return fileLine(fd.File, fd.Lines[0])
	}
	return fd.File
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itin(n) + " " + many
}

// transportLabel renders a transport code for reports.
func transportLabel(lang, t string) string {
	switch t {
	case "tls":
		return "TLS"
	case "mtls":
		return "mTLS"
	case "mtls-upstream":
		if lang == "es" {
			return "mTLS en intermediario (inferido)"
		}
		return "mTLS at intermediary (inferred)"
	case "plain":
		if lang == "es" {
			return "sin cifrar"
		}
		return "plaintext"
	}
	return ""
}
