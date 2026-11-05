package descriptors

import (
	"path"
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/sqlparse"
)

// Tech is a technology fact found in a descriptor (becomes an info finding).
type Tech struct {
	RuleID, Title, Category, Technology string
	Line                                int
}

// Result collects what a descriptor contributes to the repository facts.
type Result struct {
	Kind        string // web.xml, persistence.xml ...
	Deployable  *model.Deployable
	Contracts   []model.Contract
	Techs       []Tech
	ServletMaps map[string][]string // servlet class -> url patterns (JAX-RS app paths)
}

var reJdbcVendor = regexp.MustCompile(`(?i)jdbc:(oracle|db2|sqlserver|jtds|mysql|mariadb|postgresql|informix-sqli|sybase|h2|hsqldb|derby|as400|teradata|sqlite)`)

// IsDescriptor reports whether a file name is handled by Parse.
func IsDescriptor(rel string, text string) bool {
	b := strings.ToLower(path.Base(rel))
	switch b {
	case "web.xml", "jboss-web.xml", "jboss-deployment-structure.xml", "persistence.xml", "application.xml",
		"ejb-jar.xml", "jboss-ejb3.xml", "jboss.xml", "struts-config.xml", "struts.xml", "faces-config.xml",
		"standalone.xml", "domain.xml", "standalone-full.xml", "standalone-full-ha.xml", "standalone-ha.xml",
		"server.xml", "server-config.xml":
		return true
	}
	if strings.HasSuffix(b, "-ds.xml") || strings.HasSuffix(b, "-jms.xml") || strings.HasSuffix(b, "-service.xml") || strings.HasSuffix(b, ".wsdl") {
		return true
	}
	if strings.HasSuffix(b, ".xml") {
		head := text
		if len(head) > 2000 {
			head = head[:2000]
		}
		lh := strings.ToLower(head)
		return strings.Contains(head, "<beans") ||
			strings.Contains(head, "<struts") || strings.Contains(lh, "struts configuration") ||
			strings.Contains(head, "<sqlMapConfig") || strings.Contains(head, "<sqlMap") ||
			strings.Contains(head, "<mapper") || strings.Contains(lh, "ibatis.com//dtd") ||
			strings.Contains(lh, "mybatis.org//dtd")
	}
	return false
}

// mapperTables extracts db-table contracts from the SQL inside iBATIS / MyBatis
// mapper statements. The mapper namespace (the DAO interface for MyBatis) is
// recorded as the component.
func mapperTables(doc *Node, add func(string, string, string, string, string, string, string, int)) {
	comp := doc.Attr["namespace"]
	for _, tag := range []string{"select", "insert", "update", "delete", "statement", "procedure", "sql"} {
		for _, st := range doc.All(tag) {
			for _, tr := range sqlparse.Tables(st.InnerText()) {
				full, base := sqlparse.Normalize(tr.Table)
				add("db-table", tr.Op, full, base, comp, "iBATIS/MyBatis mapper", "high", st.Line)
			}
		}
	}
}

// Parse dispatches on the descriptor name. Host names of addresses are kept in
// Contract.Host for the scanner to seal or drop (never full URLs or credentials).
func Parse(rel, text string) *Result {
	root, _ := ParseXML(text)
	doc := root.Root()
	if doc == nil {
		return nil
	}
	b := strings.ToLower(path.Base(rel))
	r := &Result{Kind: b, ServletMaps: map[string][]string{}}
	add := func(kind, dir, key, match, comp, detail, conf string, line int) {
		if strings.TrimSpace(key) == "" {
			return
		}
		r.Contracts = append(r.Contracts, model.Contract{Kind: kind, Direction: dir, Key: key, Match: match,
			Component: comp, Detail: detail, Confidence: conf, File: rel, Line: line})
	}
	tech := func(id, title, cat, techName string, line int) {
		r.Techs = append(r.Techs, Tech{id, title, cat, techName, line})
	}
	jndiUse := func(name, detail string, line int) {
		kind := "jndi"
		if strings.Contains(strings.ToLower(detail), "jms") || strings.Contains(name, "jms/") {
			kind = "jms"
		}
		add(kind, "uses", name, model.NormalizeJNDI(name), "", detail, "high", line)
	}

	switch {
	case b == "web.xml":
		if v := doc.Attr["version"]; v != "" {
			tech("DESC-SERVLET-SPEC", "Servlet specification "+v, "Web", "Servlet "+v, doc.Line)
		}
		classes := map[string]string{}
		for _, s := range doc.ChildrenNamed("servlet") {
			cls := s.T("servlet-class")
			classes[s.T("servlet-name")] = cls
			for _, p := range s.ChildrenNamed("init-param") {
				if p.T("param-name") == "javax.ws.rs.Application" || p.T("param-name") == "jakarta.ws.rs.Application" {
					classes[s.T("servlet-name")] = "jaxrs:" + cls
				}
			}
			if s.T("servlet-name") == "javax.ws.rs.core.Application" || s.T("servlet-name") == "jakarta.ws.rs.core.Application" {
				classes[s.T("servlet-name")] = "jaxrs:"
			}
		}
		for _, m := range doc.ChildrenNamed("servlet-mapping") {
			cls := classes[m.T("servlet-name")]
			for _, u := range m.ChildrenNamed("url-pattern") {
				pat := strings.TrimSpace(u.Text)
				isRS := strings.HasPrefix(cls, "jaxrs:") || strings.Contains(strings.ToLower(cls), "resteasy") ||
					strings.Contains(strings.ToLower(cls), "jersey") || strings.Contains(cls, ".ws.rs.")
				if isRS {
					r.ServletMaps["jaxrs"] = append(r.ServletMaps["jaxrs"], pat)
					continue
				}
				add("http", "exposes", "ANY "+pat, model.NormalizePath(pat), strings.TrimPrefix(cls, "jaxrs:"), "servlet-mapping", "high", u.Line)
			}
		}
		for _, x := range doc.ChildrenNamed("resource-ref") {
			jndiUse(x.T("res-ref-name"), x.T("res-type"), x.Line)
		}
		for _, x := range doc.ChildrenNamed("resource-env-ref") {
			jndiUse(x.T("resource-env-ref-name"), x.T("resource-env-ref-type"), x.Line)
		}
		for _, x := range doc.ChildrenNamed("message-destination-ref") {
			add("jms", "uses", x.T("message-destination-ref-name"), model.NormalizeJNDI(x.T("message-destination-ref-name")), "", x.T("message-destination-usage"), "high", x.Line)
		}
		for _, tag := range []string{"ejb-ref", "ejb-local-ref"} {
			for _, x := range doc.ChildrenNamed(tag) {
				add("ejb", "calls", x.T("ejb-ref-name"), model.NormalizeJNDI(x.T("ejb-ref-name")), "", tag, "medium", x.Line)
			}
		}
		if lc := doc.Child("login-config"); lc != nil {
			am := lc.T("auth-method")
			tech("DESC-WEB-AUTH", "Container authentication: "+am, "Authentication", "Container auth ("+am+")", lc.Line)
			if strings.EqualFold(am, "CLIENT-CERT") {
				tech("DESC-WEB-MTLS", "Client-certificate authentication (mutual TLS)", "Authentication", "X.509 client auth", lc.Line)
			}
		}
		for _, sc := range doc.All("transport-guarantee") {
			if g := strings.ToUpper(strings.TrimSpace(sc.Text)); g == "CONFIDENTIAL" || g == "INTEGRAL" {
				tech("DESC-WEB-TLS", "Transport guarantee "+g+" (TLS required)", "Transport Security", "TLS required", sc.Line)
				break
			}
		}
		if sc := doc.Child("security-constraint"); sc != nil {
			tech("DESC-WEB-SECCONS", "Declarative security constraints", "Security", "Servlet security", sc.Line)
		}
	case b == "jboss-web.xml":
		d := &model.Deployable{Kind: "war", ContextRoot: doc.T("context-root"), Descriptor: rel}
		r.Deployable = d
		if sd := doc.Child("security-domain"); sd != nil {
			tech("DESC-JBOSS-SECDOMAIN", "JBoss security domain", "Security", "JBoss security domain", sd.Line)
		}
		for _, x := range doc.ChildrenNamed("resource-ref") {
			if j := x.T("jndi-name"); j != "" {
				jndiUse(j, x.T("res-type"), x.Line)
			}
		}
		tech("DESC-JBOSS-WEB", "JBoss web descriptor", "Platform", "JBoss/WildFly", doc.Line)
	case b == "jboss-deployment-structure.xml":
		tech("DESC-JBOSS-DEPSTRUCT", "JBoss deployment structure (module classloading)", "Platform", "JBoss/WildFly", doc.Line)
		for _, m := range doc.All("module") {
			if n := m.Attr["name"]; n != "" {
				dir := "uses"
				if m.Parent != nil && m.Parent.Name == "exclusions" {
					dir = "excludes"
				}
				add("platform-module", dir, n, n, "", "JBoss module", "high", m.Line)
			}
		}
	case b == "persistence.xml":
		for _, pu := range doc.ChildrenNamed("persistence-unit") {
			for _, tag := range []string{"jta-data-source", "non-jta-data-source"} {
				if ds := pu.T(tag); ds != "" {
					add("jndi", "uses", ds, model.NormalizeJNDI(ds), "", "datasource ("+pu.Attr["name"]+")", "high", pu.Path(tag).Line)
				}
			}
			if p := pu.T("provider"); p != "" {
				tech("DESC-JPA-PROVIDER", "JPA provider "+p, "Persistence", providerName(p), pu.Line)
			}
			for _, prop := range pu.All("property") {
				n := prop.Attr["name"]
				if n == "hibernate.dialect" || n == "eclipselink.target-database" || n == "openjpa.jdbc.DBDictionary" {
					v := prop.Attr["value"]
					tech("DESC-DB-DIALECT", "Database dialect "+path.Ext("." + v)[1:], "Database", dbFromDialect(v), prop.Line)
				}
			}
		}
	case b == "application.xml":
		d := &model.Deployable{Kind: "ear", Descriptor: rel}
		for _, m := range doc.ChildrenNamed("module") {
			if w := m.Child("web"); w != nil {
				d.Includes = append(d.Includes, "web:"+w.T("web-uri")+"@"+w.T("context-root"))
			}
			for _, k := range []string{"ejb", "java", "connector"} {
				if v := m.T(k); v != "" {
					d.Includes = append(d.Includes, k+":"+v)
				}
			}
		}
		r.Deployable = d
	case b == "ejb-jar.xml" || b == "jboss-ejb3.xml" || b == "jboss.xml":
		for _, s := range doc.All("session") {
			if s.T("remote") != "" || s.T("home") != "" || s.T("business-remote") != "" {
				add("ejb", "exposes", s.T("ejb-name"), model.NormalizeJNDI(s.T("ejb-name")), s.T("ejb-class"), "remote session bean", "medium", s.Line)
			}
		}
		for _, mdb := range doc.All("message-driven") {
			dest := mdb.T("destination-jndi-name")
			for _, p := range mdb.All("activation-config-property") {
				n := p.T("activation-config-property-name")
				if n == "destination" || n == "destinationLookup" {
					dest = p.T("activation-config-property-value")
				}
			}
			if dest == "" {
				dest = mdb.T("message-destination-link")
			}
			add("jms", "subscribes", dest, model.NormalizeJNDI(dest), mdb.T("ejb-class"), "message-driven bean", "high", mdb.Line)
		}
	case doc.Name == "struts-config":
		tech("DESC-STRUTS1", "Struts 1 configuration", "Web", "Struts 1", doc.Line)
		for _, a := range doc.All("action") {
			if p := a.Attr["path"]; p != "" {
				add("http", "exposes", "ANY "+p+".do", model.NormalizePath(p+".do"), a.Attr["type"], "struts action", "high", a.Line)
			}
		}
	case doc.Name == "struts":
		// Struts 2: struts.xml and every file it <include>s (arbitrary names).
		tech("DESC-STRUTS2", "Struts 2 configuration", "Web", "Struts 2", doc.Line)
		for _, pk := range doc.All("package") {
			ns := pk.Attr["namespace"]
			for _, a := range pk.ChildrenNamed("action") {
				p := model.JoinPath(ns, a.Attr["name"]) + ".action"
				add("http", "exposes", "ANY "+p, model.NormalizePath(p), a.Attr["class"], "struts2 action", "high", a.Line)
			}
		}
	case doc.Name == "sqlMap" || doc.Name == "mapper":
		mybatis := doc.Name == "mapper" || strings.Contains(strings.ToLower(text), "mybatis")
		if mybatis {
			tech("DESC-MYBATIS", "MyBatis mapper", "Persistence", "MyBatis", doc.Line)
		} else {
			tech("DESC-IBATIS", "iBATIS SQL map", "Persistence", "iBATIS", doc.Line)
		}
		mapperTables(doc, add)
	case doc.Name == "sqlMapConfig":
		tech("DESC-IBATIS", "iBATIS SQL map configuration", "Persistence", "iBATIS", doc.Line)
		for _, ds := range doc.All("dataSource") {
			for _, pr := range ds.All("property") {
				n := strings.ToLower(pr.Attr["name"] + pr.Attr["value"])
				if strings.Contains(n, "jndi") && pr.Attr["value"] != "" {
					add("jndi", "uses", pr.Attr["value"], model.NormalizeJNDI(pr.Attr["value"]), "", "iBATIS datasource", "high", pr.Line)
				}
			}
		}
	case doc.Name == "configuration" && strings.Contains(strings.ToLower(text), "mybatis"):
		tech("DESC-MYBATIS", "MyBatis configuration", "Persistence", "MyBatis", doc.Line)
	case b == "faces-config.xml":
		v := doc.Attr["version"]
		tech("DESC-JSF", "JSF configuration "+v, "Web", "JSF "+v, doc.Line)
	case strings.HasSuffix(b, ".wsdl"):
		for _, s := range doc.ChildrenNamed("service") {
			add("soap", "defines", s.Attr["name"], strings.ToLower(s.Attr["name"]), "", "WSDL service", "high", s.Line)
			for _, a := range s.All("address") {
				if loc := a.Attr["location"]; loc != "" {
					host, p := splitURL(loc)
					add("http", "defines", "ANY "+p, model.NormalizePath(p), s.Attr["name"], "address path "+p, "medium", a.Line)
					r.Contracts[len(r.Contracts)-1].Host = host // sealed or dropped by the scanner
					r.Contracts[len(r.Contracts)-1].Transport = model.URLTransport(loc)
				}
			}
		}
	case doc.Name == "beans":
		parseSpring(doc, r, add, tech, jndiUse)
	default: // server configs: standalone.xml, *-ds.xml, *-jms.xml, *-service.xml
		parseServerConfig(doc, r, add, tech)
	}
	return r
}

func parseSpring(doc *Node, r *Result, add func(string, string, string, string, string, string, string, int),
	tech func(string, string, string, string, int), jndiUse func(string, string, int)) {
	tech("DESC-SPRING-XML", "Spring XML application context", "Framework", "Spring (XML config)", doc.Line)
	for _, l := range doc.All("jndi-lookup") {
		jndiUse(l.Attr["jndi-name"], "spring jee:jndi-lookup", l.Line)
	}
	for _, bn := range doc.All("bean") {
		cls := bn.Attr["class"]
		if strings.HasSuffix(cls, "JndiObjectFactoryBean") {
			for _, p := range bn.ChildrenNamed("property") {
				if p.Attr["name"] == "jndiName" {
					v := p.Attr["value"]
					if v == "" {
						v = p.T("value")
					}
					jndiUse(v, "spring JndiObjectFactoryBean", p.Line)
				}
			}
		}
		if strings.Contains(cls, "quartz") {
			tech("DESC-SPRING-QUARTZ", "Quartz scheduling (Spring XML)", "Scheduling", "Quartz", bn.Line)
		}
	}
	for _, e := range doc.All("endpoint") { // jaxws:endpoint
		if a := e.Attr["address"]; a != "" {
			_, p := splitURL(a)
			add("soap", "exposes", e.Attr["implementor"]+" "+p, model.NormalizePath(p), e.Attr["implementor"], "jaxws:endpoint", "high", e.Line)
		}
	}
	for _, c := range doc.All("client") { // jaxws:client / jaxrs:client
		if a := c.Attr["address"]; a != "" {
			host, p := splitURL(a)
			add("http", "calls", "ANY "+p, model.NormalizePath(p), c.Attr["serviceClass"], "client", "medium", c.Line)
			r.Contracts[len(r.Contracts)-1].Host = host
			r.Contracts[len(r.Contracts)-1].Transport = model.URLTransport(a)
		}
	}
	for _, l := range doc.All("listener") { // jms:listener
		if d := l.Attr["destination"]; d != "" {
			add("jms", "subscribes", d, model.NormalizeJNDI(d), l.Attr["ref"], "spring jms:listener", "high", l.Line)
		}
	}
}

func parseServerConfig(doc *Node, r *Result, add func(string, string, string, string, string, string, string, int),
	tech func(string, string, string, string, int)) {
	for _, tag := range []string{"datasource", "xa-datasource", "local-tx-datasource", "no-tx-datasource", "xa-tx-datasource"} {
		for _, ds := range doc.All(tag) {
			j := ds.Attr["jndi-name"]
			if j == "" {
				j = ds.T("jndi-name")
			}
			add("jndi", "defines", j, model.NormalizeJNDI(j), "", "datasource binding", "high", ds.Line)
			url := ds.T("connection-url")
			if url == "" {
				url = ds.T("xa-datasource-property")
			}
			if m := reJdbcVendor.FindStringSubmatch(url); m != nil {
				tech("DESC-DB-VENDOR", "Database "+strings.ToLower(m[1]), "Database", vendorName(m[1]), ds.Line)
			}
		}
	}
	for _, tag := range []string{"jms-queue", "jms-topic"} {
		for _, q := range doc.All(tag) {
			for _, e := range strings.Fields(q.Attr["entries"]) {
				add("jms", "defines", e, model.NormalizeJNDI(e), "", tag, "high", q.Line)
			}
			for _, e := range q.All("entry") {
				add("jms", "defines", e.Attr["name"], model.NormalizeJNDI(e.Attr["name"]), "", tag, "high", e.Line)
			}
		}
	}
	for _, mb := range doc.All("mbean") { // JBoss 4/5 destinations
		if strings.Contains(mb.Attr["name"], "destination") {
			for _, part := range strings.Split(mb.Attr["name"], ",") {
				if strings.HasPrefix(part, "name=") {
					add("jms", "defines", strings.TrimPrefix(part, "name="), strings.TrimPrefix(part, "name="), "", "JBoss MQ destination", "medium", mb.Line)
				}
			}
		}
	}
	if doc.Name == "server" || doc.Name == "domain" {
		tech("DESC-JBOSS-SERVERCFG", "Application server configuration in repository", "Platform", "JBoss/WildFly", doc.Line)
	}
	// Tomcat <Connector> TLS / client-cert
	for _, c := range doc.All("Connector") {
		ssl := strings.EqualFold(c.Attr["SSLEnabled"], "true") || c.Attr["sslProtocol"] != "" || c.Attr["scheme"] == "https" || c.Attr["keystoreFile"] != "" || c.Attr["SSLCertificateFile"] != ""
		ca := strings.ToLower(c.Attr["clientAuth"] + c.Attr["certificateVerification"] + c.Attr["SSLVerifyClient"])
		switch {
		case strings.Contains(ca, "true") || strings.Contains(ca, "want") || strings.Contains(ca, "require") || strings.Contains(ca, "optional"):
			tech("DESC-TOMCAT-MTLS", "Tomcat connector requires client certificate (mutual TLS)", "Authentication", "X.509 client auth", c.Line)
		case ssl:
			tech("DESC-TOMCAT-TLS", "Tomcat TLS connector", "Transport Security", "TLS", c.Line)
		}
		if p := weakTLSProtocol(c.Attr["sslProtocol"] + " " + c.Attr["sslEnabledProtocols"] + " " + c.Attr["protocols"]); p != "" {
			tech("DESC-TLS-WEAK", "Weak TLS protocol configured: "+p, "Transport Security", "Weak TLS", c.Line)
		}
	}
	// WildFly / Undertow https-listener and security-realm (mTLS)
	for _, l := range doc.All("https-listener") {
		if v := strings.ToLower(l.Attr["verify-client"]); strings.Contains(v, "require") || strings.Contains(v, "request") {
			tech("DESC-WILDFLY-MTLS", "Undertow https-listener verifies client certificate (mutual TLS)", "Authentication", "X.509 client auth", l.Line)
		} else {
			tech("DESC-WILDFLY-TLS", "Undertow https-listener (TLS)", "Transport Security", "TLS", l.Line)
		}
	}
	for _, sr := range doc.All("security-realm") {
		hasTrust := sr.Child("authentication") != nil || len(sr.All("truststore")) > 0
		hasID := sr.Child("server-identities") != nil || len(sr.All("keystore")) > 0
		if hasTrust && hasID {
			tech("DESC-WILDFLY-REALM-MTLS", "Security realm with keystore and truststore (mutual TLS capable)", "Authentication", "X.509 client auth", sr.Line)
		}
	}
}

// weakTLSProtocol returns the first obsolete protocol named in s, or "".
func weakTLSProtocol(s string) string {
	for _, w := range []string{"SSLv2", "SSLv3", "TLSv1.0", "TLSv1.1"} {
		if strings.Contains(s, w) {
			return w
		}
	}
	// bare "TLSv1" (not TLSv1.2/1.3)
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '+' }) {
		if f == "TLSv1" {
			return "TLSv1"
		}
	}
	return ""
}

func splitURL(u string) (host, p string) {
	u = strings.TrimSpace(u)
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 && at < strings.IndexAny(rest+"/", "/") {
			rest = rest[at+1:] // drop userinfo, never record it
		}
		if j := strings.IndexByte(rest, '/'); j >= 0 {
			return stripPort(rest[:j]), rest[j:]
		}
		return stripPort(rest), "/"
	}
	return "", u
}

func stripPort(h string) string {
	if i := strings.LastIndexByte(h, ':'); i >= 0 {
		return h[:i]
	}
	return h
}

func providerName(p string) string {
	switch {
	case strings.Contains(p, "hibernate"):
		return "Hibernate"
	case strings.Contains(p, "eclipse"):
		return "EclipseLink"
	case strings.Contains(p, "openjpa"):
		return "OpenJPA"
	case strings.Contains(p, "toplink"):
		return "TopLink"
	}
	return p
}

func dbFromDialect(v string) string {
	l := strings.ToLower(v)
	for _, k := range []string{"oracle", "db2", "sqlserver", "postgres", "mysql", "mariadb", "informix", "sybase", "h2", "hsql", "derby", "ingres", "teradata"} {
		if strings.Contains(l, k) {
			return vendorName(k)
		}
	}
	return v
}

func vendorName(k string) string {
	switch strings.ToLower(k) {
	case "oracle":
		return "Oracle Database"
	case "db2", "as400":
		return "IBM DB2"
	case "sqlserver", "jtds":
		return "SQL Server"
	case "postgresql", "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL"
	case "mariadb":
		return "MariaDB"
	case "informix", "informix-sqli":
		return "Informix"
	case "sybase":
		return "Sybase"
	case "h2", "hsql", "hsqldb", "derby":
		return "Embedded DB (" + k + ")"
	}
	return k
}
