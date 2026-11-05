package scan

import (
	"path"
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// classify runs after every extractor: it tells API from user screens, sorts
// authentication evidence by kind and names the other side of each
// integration from what the code itself says (no external catalog).
func classify(f *model.Facts) {
	classifyRoles(f)
	classifyAuth(f)
	markUpstreamMTLS(f)
	nameCounterparts(f)
}

// UpstreamMTLSPrefix marks rules whose matches show that a client certificate
// reaches the application in an HTTP header: an intermediary (proxy, API
// gateway, load balancer) terminates mutual TLS in front of it.
const UpstreamMTLSPrefix = "TLS-UPSTREAM-"

// reCrossCutting names classes that apply to every request of a module
// (servlet filters, interceptors, middleware, security configuration).
var reCrossCutting = regexp.MustCompile(`(?i)(filter|interceptor|middleware|handler|aspect|security|auth|config|base)[^/]*$`)

// markUpstreamMTLS sets the transport of inbound endpoints to "mtls-upstream"
// when the code that serves them reads the client certificate from a header.
// A header read in a controller marks that controller's endpoints; in a
// filter or interceptor, every API endpoint of the same module.
func markUpstreamMTLS(f *model.Facts) {
	files := map[string]bool{}
	for _, fd := range f.Findings {
		if strings.HasPrefix(fd.RuleID, UpstreamMTLSPrefix) && fd.RuleID != UpstreamMTLSPrefix+"CERT-HEADER-CONFIG" && fd.SourceSet == "main" {
			files[fd.File] = true
		}
	}
	if len(files) == 0 {
		return
	}
	inbound := func(c *model.Contract) bool {
		return c.Direction == "exposes" && (c.Kind == "http" || c.Kind == "soap")
	}
	mark := func(c *model.Contract) {
		if c.Transport == "" || c.Transport == "tls" {
			c.Transport = "mtls-upstream"
		}
		if c.Auth == "" {
			c.Auth = "client-cert (header)"
		}
		if c.Role == "" {
			c.Role = "api" // a caller that presents a certificate is a system, not a person
		}
	}
	served := map[string]bool{}
	for i := range f.Contracts {
		if c := &f.Contracts[i]; inbound(c) && files[c.File] {
			mark(c)
			served[c.File] = true
		}
	}
	for file := range files {
		if served[file] || !reCrossCutting.MatchString(path.Base(file)) {
			continue
		}
		mod := moduleOf(f, file)
		for i := range f.Contracts {
			c := &f.Contracts[i]
			if inbound(c) && (c.Role == "api" || c.Kind == "soap") && moduleOf(f, c.File) == mod {
				mark(c)
			}
		}
	}
}

// moduleOf returns the path of the deepest build module containing file.
func moduleOf(f *model.Facts, file string) string {
	best := ""
	for _, m := range f.Modules {
		p := m.Path
		if p == "" || p == "." {
			if best == "" {
				best = "."
			}
			continue
		}
		if (file == p || strings.HasPrefix(file, p+"/")) && len(p) > len(best) {
			best = p
		}
	}
	return best
}

var reAPIPath = regexp.MustCompile(`(?i)(^|/)(ws|api|apis|rest|services|service|servicios|soap|rpc|graphql|odata)(/|$)`)

func classifyRoles(f *model.Facts) {
	for i := range f.Contracts {
		c := &f.Contracts[i]
		if c.Direction != "exposes" || c.Role != "" {
			continue
		}
		low := strings.ToLower(c.Detail)
		switch {
		case c.Kind == "soap" || c.Kind == "ejb" || c.Kind == "jms" || c.Kind == "kafka":
			c.Role = "api"
		case c.Kind != "http":
		case strings.Contains(low, "struts") || strings.Contains(low, "jsf") || strings.Contains(low, "faces"):
			c.Role = "ui"
		case reAPIPath.MatchString(pathOf(c.Key)):
			c.Role = "api"
		}
	}
}

func pathOf(key string) string {
	if i := strings.IndexByte(key, ' '); i >= 0 {
		return key[i+1:]
	}
	return key
}

// descriptorAuthKind covers the findings the engine derives from standard
// server descriptors (web.xml, server.xml, standalone.xml); catalog rules
// declare their own kind in the rule data.
var descriptorAuthKind = map[string]string{
	"DESC-WEB-MTLS": "transport", "DESC-WEB-TLS": "transport", "DESC-TOMCAT-MTLS": "transport", "DESC-TOMCAT-TLS": "transport",
	"DESC-WILDFLY-MTLS": "transport", "DESC-WILDFLY-TLS": "transport", "DESC-WILDFLY-REALM-MTLS": "transport", "DESC-TLS-WEAK": "transport",
}

var reLoginFile = regexp.MustCompile(`(?i)(login|logon|signin|sign-in|acceso|autentica|authentica|identifica)`)

func classifyAuth(f *model.Facts) {
	wsSecFiles := map[string]bool{}
	for _, fd := range f.Findings {
		if fd.RuleID == "AUTH-WS-SECURITY" {
			wsSecFiles[fd.File] = true
		}
	}
	defer func() {
		for i := range f.Findings {
			if fd := &f.Findings[i]; fd.RuleID == "JAVA-JAAS" && wsSecFiles[fd.File] {
				fd.AuthKind = "service" // callback handlers of a SOAP client, not user login
			}
		}
	}()
	for i := range f.Findings {
		fd := &f.Findings[i]
		k := fd.AuthKind // declared by the rule (catalog data, not code)
		if k == "" {
			k = descriptorAuthKind[fd.RuleID]
		}
		if k == "" && (fd.Category == "Authentication" || fd.Category == "Transport Security") {
			k = "transport"
			if fd.Category == "Authentication" {
				k = "user"
			}
		}
		// e-signature on a login screen or controller is certificate login
		if k == "signature" && reLoginFile.MatchString(path.Base(fd.File)) {
			k = "user"
		}
		fd.AuthKind = k
	}
}

// tokens that say nothing about who is on the other side
var genericCodeTokens = map[string]bool{
	"negocio": true, "service": true, "services": true, "servicio": true, "servicios": true, "impl": true, "client": true,
	"clients": true, "cliente": true, "clientes": true, "ws": true, "rest": true, "api": true, "integration": true, "integracion": true,
	"external": true, "externo": true, "externos": true, "proxy": true, "util": true, "utils": true, "common": true, "core": true,
	"dao": true, "repository": true, "model": true, "modelo": true, "domain": true, "app": true, "web": true, "controller": true,
	"business": true, "es": true, "com": true, "org": true, "gob": true, "gov": true, "edu": true, "net": true, "soap": true, "generated": true, "stub": true,
	"stubs": true, "port": true, "binding": true, "wsdl": true, "jaxws": true, "cxf": true, "axis": true, "remote": true, "remoto": true,
	"connector": true, "conector": true, "adapter": true, "adaptador": true, "infra": true, "infrastructure": true, "main": true,
	"java": true, "src": true, "http": true, "comun": true, "comunes": true,
}

var reCamelWS = regexp.MustCompile(`(?i)(ws|client|cliente|service|servicio|impl|port|proxy|stub|soap|rest|api)+$`)

// counterpartFromCode derives a name from the package and class of the
// component that makes the call: orders.business.payrollWS.TimeEntriesImplService → "payroll".
func counterpartFromCode(component, ownName string) string {
	if component == "" {
		return ""
	}
	parts := strings.Split(component, ".")
	own := strings.ToLower(ownName)
	pick := func(t string) string {
		t = reCamelWS.ReplaceAllString(t, "")
		lt := strings.ToLower(t)
		if len(lt) < 3 || genericCodeTokens[lt] || lt == own {
			return ""
		}
		return lt
	}
	// packages from the most specific, then the class name
	for i := len(parts) - 2; i >= 1; i-- {
		if p := pick(parts[i]); p != "" {
			return p
		}
	}
	// the class name only when it says it is a client of something
	cls := parts[len(parts)-1]
	if m := reClientPrefix.FindStringSubmatch(cls); m != nil {
		return pick(m[2])
	}
	if reClientClass.MatchString(cls) {
		return pick(cls)
	}
	return ""
}

var reClientPrefix = regexp.MustCompile(`^(Cliente|Client|Proxy|Conector|Connector|Pasarela)([A-Z]\w+)$`)

var reClientClass = regexp.MustCompile(`(?i)(client|cliente|ws|proxy|stub|remote|remoto|connector|conector|adapter|adaptador|gateway|pasarela)(impl)?$`)

// well-known vendor platform modules (JBoss/WildFly modules, OSGi bundles):
// everything else is the organization's own shared component.
var vendorModulePrefixes = []string{"org.", "com.sun.", "com.oracle.", "com.ibm.", "com.microsoft.", "javax.", "jakarta.",
	"sun.", "java.", "io.", "net.sf.", "com.fasterxml.", "com.google.", "ch.qos.", "asm.", "gnu.", "com.h2database",
	"com.mysql", "deployment.", "com.github.", "nu.xom", "net.jcip", "com.zaxxer", "com.thoughtworks."}

func vendorModule(m string) bool {
	m = strings.ToLower(m)
	for _, p := range vendorModulePrefixes {
		if strings.HasPrefix(m, p) {
			return true
		}
	}
	return false
}

var dictionarySchemas = map[string]bool{"SYS": true, "SYSTEM": true, "DBA": true, "ALL": true, "USER": true, "V$": true,
	"INFORMATION_SCHEMA": true, "PG_CATALOG": true, "DBO": true, "SYSIBM": true, "SYSCAT": true, "PUBLIC": true}

var reVersionSuffix = regexp.MustCompile(`[-_]v?\d+(\.\d+)*([-_.][A-Za-z0-9]+)*$`)

// ownRoots are the names under which this system is reached over HTTP: its
// context roots (without version suffixes), module names and its own name.
func ownRoots(f *model.Facts) map[string]bool {
	out := map[string]bool{strings.ToLower(f.Repo.Name): true}
	add := func(s string) {
		s = strings.ToLower(strings.Trim(s, "/"))
		if s == "" || strings.Contains(s, "/") {
			return
		}
		out[s] = true
		if t := reVersionSuffix.ReplaceAllString(s, ""); t != "" {
			out[t] = true
		}
	}
	for _, d := range f.Deployables {
		add(d.ContextRoot)
	}
	for _, m := range f.Modules {
		if m.Packaging == "war" || m.Packaging == "ear" {
			add(m.Name)
		}
	}
	return out
}

// generic first path segments that do not name a system
var genericPathSeg = map[string]bool{"ws": true, "api": true, "apis": true, "rest": true, "services": true, "service": true,
	"servicios": true, "soap": true, "app": true, "web": true, "v1": true, "v2": true, "v3": true, "public": true, "private": true,
	"internal": true, "es": true, "en": true, "index": true, "home": true, "inicio": true, "static": true, "resources": true}

// firstSegment returns the first literal path segment of a contract key.
func firstSegment(key string) string {
	p := pathOf(key)
	for _, seg := range strings.Split(p, "/") {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") || strings.HasPrefix(seg, "$") || strings.HasPrefix(seg, ":") {
			return ""
		}
		return strings.ToLower(seg)
	}
	return ""
}

func nameCounterparts(f *model.Facts) {
	own := strings.ToLower(f.Repo.Name)
	roots := ownRoots(f)
	// schemas this repository owns: those it creates, plus one named like the system
	ownSchemas := map[string]bool{strings.ToUpper(f.Repo.Name): true}
	for _, c := range f.Contracts {
		if c.Kind == "db-table" && c.Direction == "defines" {
			if i := strings.LastIndexByte(c.Key, '.'); i > 0 {
				ownSchemas[c.Key[:i]] = true
			}
		}
	}
	for i := range f.Contracts {
		c := &f.Contracts[i]
		// a configuration key that starts with the system's own name names
		// the system itself, not the other side
		if cp := c.Counterpart; cp == own || strings.HasPrefix(cp, own+".") {
			c.Counterpart = strings.TrimPrefix(strings.TrimPrefix(cp, own), ".")
			if c.Counterpart == "" && c.Kind == "database" {
				c.Counterpart = "database"
			}
		}
		if c.Counterpart != "" {
			continue
		}
		switch {
		case c.Kind == "db-table" && c.Direction != "defines":
			if j := strings.LastIndexByte(c.Key, '.'); j > 0 {
				schema := c.Key[:j]
				if !ownSchemas[schema] && !dictionarySchemas[schema] && !strings.HasPrefix(schema, "V$") {
					c.Counterpart = "schema:" + strings.ToLower(schema) // data owned by another system
				}
			}
		case c.Kind == "platform-module" && c.Direction == "uses":
			if !vendorModule(c.Key) {
				c.Counterpart = "module:" + c.Key
			}
		case c.Kind == "jndi" && c.Direction == "uses":
			lk := strings.ToLower(c.Key)
			switch {
			case strings.Contains(lk, "mail"):
				c.Counterpart = "mail"
			case strings.Contains(lk, "jdbc/") || strings.Contains(lk, "datasource") || strings.HasSuffix(lk, "ds"):
				c.Counterpart = "database"
			}
		case c.Direction == "calls" || (c.Kind == "soap" && c.Direction == "defines"):
			if cp := counterpartFromCode(c.Component, own); cp != "" {
				c.Counterpart = cp
			} else if c.Kind == "soap" {
				c.Counterpart = strings.ToLower(reCamelWS.ReplaceAllString(c.Key, ""))
			}
		}
	}
	// HTTP calls: the first path segment is usually the context root of the
	// target application (/signhub-signature/…, /docstore/…); when it is
	// one of this system's own roots, the call is the system calling itself.
	for i := range f.Contracts {
		c := &f.Contracts[i]
		if c.Kind != "http" || c.Direction != "calls" {
			continue
		}
		seg := firstSegment(c.Key)
		switch {
		case seg != "" && roots[seg]:
			c.Counterpart = "self"
		case seg != "" && !genericPathSeg[seg] && len(seg) >= 3:
			c.Counterpart = seg
		}
	}
	// a package-derived name that is just a variant of the system's own name
	// (notif in notifier) says nothing: fall back to the service name
	for i := range f.Contracts {
		c := &f.Contracts[i]
		if c.Kind == "soap" && c.Counterpart != "" && (strings.Contains(own, c.Counterpart) || strings.Contains(c.Counterpart, own)) {
			c.Counterpart = strings.ToLower(reCamelWS.ReplaceAllString(c.Key, ""))
		}
	}
}
