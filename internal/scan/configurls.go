package scan

import (
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/model"
)

// Integrations declared in configuration: Java EE applications keep their
// endpoints in .properties, YAML, Spring XML and server descriptors, not in
// code. Every URL-valued entry becomes an outbound contract; the key names the
// probable counterpart. Values are never copied: hosts go through sealHosts.

// reCfgURL finds URLs with a scheme we can classify.
var reCfgURL = regexp.MustCompile(`(?i)\b(https?|wss?|ldaps?|smtps?|ftps?|sftp|amqps?|mqtt|stomp|jdbc:[a-z0-9\-]+(?::[a-z0-9\-]+)*):(//)?[^\s"'<>]+`)

// keys whose plain (non-URL) value is a mail server host
var reMailHostKey = regexp.MustCompile(`(?i)(^|[._-])(smtp|mail)[._-]?(smtp[._-])?(host|server|servidor)$`)

var genericKeyTokens = map[string]bool{
	"url": true, "uri": true, "urls": true, "endpoint": true, "endpoints": true, "host": true, "hostname": true, "server": true,
	"address": true, "addr": true, "base": true, "baseurl": true, "path": true, "location": true, "wsdl": true, "ws": true,
	"service": true, "services": true, "servicio": true, "servicios": true, "conexion": true, "connection": true, "servidor": true,
	"direccion": true, "dir": true, "prop": true, "props": true, "property": true, "properties": true, "config": true, "cfg": true,
	"app": true, "application": true, "spring": true, "datasource": true, "jdbc": true, "http": true, "https": true, "rest": true,
	"api": true, "client": true, "cliente": true, "remote": true, "remoto": true, "port": true, "puerto": true, "ip": true,
	"smtp": true, "mail": true, "ldap": true, "provider": true, "context": true, "value": true, "name": true, "target": true,
	"default": true, "int": true, "pre": true, "pro": true, "des": true, "dev": true, "prod": true, "test": true, "local": true,
	"global": true, "env": true, "es": true, "gob": true, "gov": true, "edu": true, "co": true, "io": true, "com": true, "org": true, "net": true, "xml": true, "bean": true,
	"entry": true, "key": true, "param": true, "parameter": true, "init": true, "web": true, "webservice": true, "soap": true,
	"wsdlurl": true, "serviceurl": true, "endpointurl": true, "addressurl": true, "id": true, "java": true, "jboss": true,
	"ds": true, "jndi": true, "environment": true, "containers": true, "container": true, "spec": true, "template": true,
	"values": true, "deploy": true, "deployment": true, "compose": true, "connection-url": true, "outbound": true, "socket": true, "binding": true, "destination": true,
}

var reKeySplit = regexp.MustCompile(`[._\-\s/\[\]:]+|([a-z])([A-Z])`)

// counterpartFromKey extracts the meaningful words of a configuration key:
// "signer.ws.url" → "signer"; "pasarela.taxgw.endpoint" → "pasarela.taxgw".
func counterpartFromKey(key string) string {
	k := reKeySplit.ReplaceAllString(key, "$1 $2")
	var out []string
	for _, t := range strings.Fields(strings.ToLower(k)) {
		t = strings.TrimSuffix(strings.TrimSuffix(t, "client"), "service")
		if len(t) < 2 || genericKeyTokens[t] || isNumber(t) {
			continue
		}
		out = append(out, t)
		if len(out) == 2 {
			break
		}
	}
	return strings.Join(out, ".")
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

var reTestKey = regexp.MustCompile(`(?i)(^|[._-])(test|tests|mock|dummy|example|ejemplo|sample|local|localhost)([._-]|$)`)

// message bundles hold texts shown to users (links to web pages), and Maven
// settings or CI pipelines point to build tooling: none is a runtime integration.
var reMessageBundle = regexp.MustCompile(`(?i)(^|/)(mensajes?|messages?|applicationresources|i18n|labels?|literales|textos|errors?)([_-][a-z]{2}(_[a-z]{2})?)?\.properties$|(^|/)(mensajes|messages|i18n|literales)/`)
var reToolingFile = regexp.MustCompile(`(?i)(^|/)(settings[^/]*\.xml|jenkinsfile[^/]*|[^/]*\.groovy|\.gitlab-ci\.ya?ml|sonar-project\.properties|deploy-request\.properties)$|(^|/)pipes?/`)

// nonEndpoint reports URLs that identify documents or build tooling rather
// than a system the application talks to at run time.
var reDocURL = regexp.MustCompile(`(?i)\.(pdf|docx?|xlsx?|pptx?|odt|xsd|dtd|xsl|png|jpe?g|gif|svg|ico|css|js|html?|txt|zip|jar)$`)
var reToolingPath = regexp.MustCompile(`(?i)/(artifactory|nexus|repository|svn|jenkins|sonar|sonarqube|gitlab|maven2?|repo1|nexus3)(/|$)`)

func nonEndpoint(path string) bool {
	return reDocURL.MatchString(path) || reToolingPath.MatchString(path)
}

// configContracts turns URL-valued configuration entries into contracts.
func configContracts(rel string, l lang.Language, text string) []model.Contract {
	if reMessageBundle.MatchString(rel) || reToolingFile.MatchString(rel) {
		return nil
	}
	var out []model.Contract
	switch l.Name {
	case "Properties":
		forEachProperty(text, func(key, val string, line int) {
			out = append(out, cfgEntry(rel, key, val, line)...)
		})
	case "YAML":
		forEachYAML(text, func(key, val string, line int) {
			out = append(out, cfgEntry(rel, key, val, line)...)
		})
	case "XML":
		if e := strings.ToLower(rel[strings.LastIndexByte(rel, '.')+1:]); e == "xml" || e == "config" {
			out = append(out, xmlConfigURLs(rel, text)...)
		}
	}
	return out
}

func lineOf(text string, off int) int { return strings.Count(text[:off], "\n") + 1 }

// forEachProperty parses key=value / key: value / key value lines (Java
// .properties, with continuation lines), skipping comments.
func forEachProperty(text string, fn func(key, val string, line int)) {
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		start := i
		ln := strings.TrimRight(lines[i], "\r")
		for strings.HasSuffix(ln, "\\") && i+1 < len(lines) {
			i++
			ln = strings.TrimSuffix(ln, "\\") + strings.TrimSpace(strings.TrimRight(lines[i], "\r"))
		}
		t := strings.TrimSpace(ln)
		if t == "" || t[0] == '#' || t[0] == '!' {
			continue
		}
		sep := strings.IndexAny(t, "=: \t")
		if sep <= 0 {
			continue
		}
		val := strings.TrimSpace(t[sep+1:])
		if t[sep] == ' ' || t[sep] == '\t' {
			val = strings.TrimSpace(strings.TrimLeft(val, "=:"))
		}
		fn(t[:sep], val, start+1)
	}
}

var reYAMLLine = regexp.MustCompile(`^(\s*)(-\s+)?([A-Za-z0-9_.\-"']+)\s*:\s*(.*)$`)

// forEachYAML walks "key: value" lines building dotted keys from indentation.
func forEachYAML(text string, fn func(key, val string, line int)) {
	type lvl struct {
		indent int
		key    string
	}
	var stack []lvl
	for i, raw := range strings.Split(text, "\n") {
		ln := strings.TrimRight(raw, "\r")
		if t := strings.TrimSpace(ln); t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		m := reYAMLLine.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		indent := len(m[1])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		key := strings.Trim(m[3], `"'`)
		full := key
		if len(stack) > 0 {
			full = stack[len(stack)-1].key + "." + key
		}
		val := strings.TrimSpace(m[4])
		if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		val = strings.Trim(val, `"'`)
		if val == "" || val == "|" || val == ">" {
			stack = append(stack, lvl{indent, full})
			continue
		}
		fn(full, val, i+1)
	}
}

// cfgEntry classifies one configuration value.
func cfgEntry(rel, key, val string, line int) []model.Contract {
	if val == "" {
		return nil
	}
	var out []model.Contract
	cp := counterpartFromKey(key)
	conf := "medium"
	if reTestKey.MatchString(key) {
		conf = "low"
	}
	for _, m := range reCfgURL.FindAllStringIndex(val, -1) {
		if c, ok := urlContract(val[m[0]:m[1]], "configuration: "+key, cp, conf, rel, line); ok {
			out = append(out, c)
		}
	}
	if len(out) == 0 && reMailHostKey.MatchString(key) && isHostLike(val) {
		out = append(out, model.Contract{Kind: "mail", Direction: "calls", Key: "smtp", Match: "", Detail: "configuration: " + key,
			Confidence: conf, Host: val, Counterpart: "mail", File: rel, Line: line})
	}
	return out
}

var reHostLike = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.\-]*[A-Za-z0-9]$`)

func isHostLike(s string) bool { return reHostLike.MatchString(s) && !strings.Contains(s, "${") }

var reOracleHost = regexp.MustCompile(`(?i)\(\s*HOST\s*=\s*([^)\s]+)`)
var reOracleSvc = regexp.MustCompile(`(?i)\(\s*(SERVICE_NAME|SID)\s*=\s*([^)\s]+)`)

// urlContract builds the contract for one URL found in configuration.
func urlContract(u, detail, cp, conf, rel string, line int) (model.Contract, bool) {
	u = strings.TrimRight(u, ".,;)")
	if strings.HasPrefix(u, "${") || strings.Contains(u, "://${") {
		return model.Contract{}, false // unresolved placeholder: nothing objective to report
	}
	lower := strings.ToLower(u)
	c := model.Contract{Direction: "calls", Detail: detail, Confidence: conf, Counterpart: cp, File: rel, Line: line,
		Transport: model.URLTransport(u)}
	switch {
	case strings.HasPrefix(lower, "jdbc:"):
		c.Kind, c.Direction = "database", "uses"
		vendor := strings.SplitN(lower[5:], ":", 2)[0]
		c.Key, c.Transport = vendor, ""
		host, svc := jdbcTarget(u)
		if host != "" {
			c.Host = host
			if svc != "" {
				c.Host = host + "/" + svc // same database service ⇒ same pseudonym across systems
			}
		}
		if strings.Contains(lower, "tcps") || strings.Contains(lower, "ssl=true") || strings.Contains(lower, "encrypt=true") || strings.Contains(lower, "sslmode=require") {
			c.Transport = "tls"
		}
		if cp == "" {
			c.Counterpart = "database"
		}
		return c, true
	case strings.HasPrefix(lower, "ldap"):
		c.Kind, c.Key = "ldap", "ldap"
	case strings.HasPrefix(lower, "smtp"):
		c.Kind, c.Key = "mail", "smtp"
	case strings.HasPrefix(lower, "ftp") || strings.HasPrefix(lower, "sftp"):
		c.Kind, c.Key = "file-transfer", strings.SplitN(lower, ":", 2)[0]
	case strings.HasPrefix(lower, "amqp") || strings.HasPrefix(lower, "mqtt") || strings.HasPrefix(lower, "stomp"):
		c.Kind, c.Key = "messaging", strings.SplitN(lower, ":", 2)[0]
	default: // http(s), ws(s)
		if !strings.Contains(u, "://") {
			return model.Contract{}, false
		}
		host, p := splitHostPath(u)
		if namespaceHosts[strings.ToLower(host)] || host == "" {
			return model.Contract{}, false
		}
		if q := strings.IndexAny(p, "?#"); q >= 0 {
			p = p[:q]
		}
		if nonEndpoint(p) {
			return model.Contract{}, false
		}
		np := model.NormalizePath(p)
		c.Kind, c.Key, c.Host = "http", "ANY "+np, host
		if model.LiteralSegments(np) > 0 {
			c.Match = np
		}
		if strings.HasSuffix(strings.ToLower(p), "wsdl") || strings.Contains(lower, "?wsdl") {
			c.Kind = "soap"
			c.Key = np
		}
		return c, true
	}
	if strings.Contains(u, "://") {
		host, _ := splitHostPath(u)
		c.Host = host
	}
	return c, true
}

// jdbcTarget returns host and database/service name of a JDBC URL.
func jdbcTarget(u string) (host, svc string) {
	if m := reOracleHost.FindStringSubmatch(u); m != nil {
		host = m[1]
		if s := reOracleSvc.FindStringSubmatch(u); s != nil {
			svc = s[2]
		}
		return
	}
	rest := u
	if i := strings.Index(rest, "@"); i >= 0 { // oracle thin: @host:port:SID or @//host:port/SERVICE
		rest = strings.TrimPrefix(rest[i+1:], "//")
		parts := strings.FieldsFunc(rest, func(r rune) bool { return r == ':' || r == '/' })
		if len(parts) > 0 {
			host = parts[0]
		}
		if len(parts) > 2 {
			svc = parts[len(parts)-1]
		}
		return
	}
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
		hp := rest
		if j := strings.IndexAny(rest, "/;?"); j >= 0 {
			hp, rest = rest[:j], rest[j:]
		} else {
			rest = ""
		}
		host = hp
		if k := strings.LastIndexByte(host, ':'); k >= 0 {
			host = host[:k]
		}
		if strings.HasPrefix(rest, "/") {
			svc = strings.SplitN(strings.TrimPrefix(rest, "/"), "?", 2)[0]
			svc = strings.SplitN(svc, ";", 2)[0]
		}
		if svc == "" {
			if m := regexp.MustCompile(`(?i)databaseName=([^;]+)`).FindStringSubmatch(rest); m != nil {
				svc = m[1]
			}
		}
	}
	return
}

// XML configuration: URL-valued attributes and text, plus WildFly/JBoss
// datasources and outbound socket bindings.
var reXMLTagStart = regexp.MustCompile(`<([A-Za-z_][\w:.\-]*)([^<>]*)$`)
var reXMLAttr = regexp.MustCompile(`([\w:.\-]+)\s*=\s*"[^"]*$`)
var reXMLTag = regexp.MustCompile(`<(/?)([A-Za-z_][\w:.\-]*)([^<>]*?)(/?)>`)

// xmlContext returns the name/id attributes of the elements enclosing offset.
func xmlContext(text string, off int) []string {
	var stack []string
	for _, m := range reXMLTag.FindAllStringSubmatchIndex(text[:off], -1) {
		closing, selfClosing := m[3] > m[2], m[9] > m[8]
		switch {
		case closing:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case selfClosing:
		default:
			id := ""
			if nm := reXMLNameAttr.FindStringSubmatch(text[m[6]:m[7]]); nm != nil {
				id = nm[1]
			}
			stack = append(stack, id)
		}
	}
	var out []string
	for i := len(stack) - 1; i >= 0 && len(out) < 2; i-- {
		if stack[i] != "" {
			out = append([]string{stack[i]}, out...)
		}
	}
	return out
}

var reXMLNameAttr = regexp.MustCompile(`\b(?:name|id|key)\s*=\s*"([^"]+)"`)
var reOutboundBinding = regexp.MustCompile(`(?s)<outbound-socket-binding\s+[^>]*name\s*=\s*"([^"]+)"[^>]*>\s*<remote-destination\s+[^>]*host\s*=\s*"([^"]+)"`)

func xmlConfigURLs(rel, text string) []model.Contract {
	low := strings.ToLower(text[:min(len(text), 4096)])
	if strings.Contains(low, "<wsdl:definitions") || strings.Contains(low, "<definitions") || strings.Contains(low, "<xs:schema") ||
		strings.Contains(low, "<xsd:schema") || strings.Contains(low, "<project") && strings.Contains(low, "maven") {
		return nil // contracts and build files are handled elsewhere
	}
	var out []model.Contract
	for _, m := range reCfgURL.FindAllStringIndex(text, -1) {
		before := text[:m[0]]
		if i := strings.LastIndex(before, "<!--"); i >= 0 && !strings.Contains(before[i:], "-->") {
			continue // inside a comment
		}
		key := ""
		if tm := reXMLTagStart.FindStringSubmatch(before); tm != nil { // inside a tag: attribute value
			attr := ""
			if am := reXMLAttr.FindStringSubmatch(before); am != nil {
				attr = am[1]
			}
			la := strings.ToLower(attr)
			if strings.HasPrefix(la, "xmlns") || strings.HasSuffix(la, "schemalocation") || la == "href" || la == "src" {
				continue
			}
			key = tm[1]
			if nm := reXMLNameAttr.FindStringSubmatch(tm[2]); nm != nil {
				key = nm[1] + "." + key
			}
			if attr != "" && attr != "value" {
				key += "." + attr
			}
			if ctx := xmlContext(text, m[0]); len(ctx) > 0 {
				key = strings.Join(ctx, ".") + "." + key
			}
		} else if j := strings.LastIndex(before, "<"); j >= 0 { // element text
			tag := before[j+1:]
			if k := strings.IndexAny(tag, " >/"); k >= 0 {
				attrs := tag[k:]
				tag = tag[:k]
				if nm := reXMLNameAttr.FindStringSubmatch(attrs); nm != nil {
					tag = nm[1] + "." + tag
				}
			}
			if strings.HasPrefix(tag, "!") || strings.HasPrefix(tag, "?") {
				continue
			}
			key = tag
			if ctx := xmlContext(text, j); len(ctx) > 0 {
				key = strings.Join(ctx, ".") + "." + key
			}
		}
		if c, ok := urlContract(text[m[0]:m[1]], "configuration: "+key, counterpartFromKey(key), "medium", rel, lineOf(text, m[0])); ok {
			out = append(out, c)
		}
	}
	for _, m := range reOutboundBinding.FindAllStringSubmatchIndex(text, -1) {
		name, host := text[m[2]:m[3]], text[m[4]:m[5]]
		if strings.Contains(host, "${") {
			continue
		}
		kind, key := "socket", "tcp"
		if ln := strings.ToLower(name); strings.Contains(ln, "mail") || strings.Contains(ln, "smtp") {
			kind, key = "mail", "smtp"
		}
		out = append(out, model.Contract{Kind: kind, Direction: "calls", Key: key, Detail: "configuration: outbound-socket-binding " + name,
			Confidence: "medium", Host: host, Counterpart: counterpartFromKey(name), File: rel, Line: lineOf(text, m[0])})
	}
	return out
}
