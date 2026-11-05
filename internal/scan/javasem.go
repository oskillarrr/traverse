package scan

import (
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/java"
	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/sqlparse"
)

var depRank = map[string]int{"extends": 6, "inject": 5, "import": 4, "fqn": 3, "wildcard": 2, "same-package": 1}

var httpVerbs = map[string]string{
	"GET": "GET", "POST": "POST", "PUT": "PUT", "DELETE": "DELETE", "PATCH": "PATCH", "HEAD": "HEAD", "OPTIONS": "OPTIONS",
	"GetMapping": "GET", "PostMapping": "POST", "PutMapping": "PUT", "DeleteMapping": "DELETE", "PatchMapping": "PATCH",
}

var reReqMethod = regexp.MustCompile(`RequestMethod\.(\w+)`)
var reMDBDest = regexp.MustCompile(`propertyName\s*=\s*"(?:destination|destinationLookup)"\s*,\s*propertyValue\s*=\s*"([^"]+)"|propertyValue\s*=\s*"([^"]+)"\s*,\s*propertyName\s*=\s*"(?:destination|destinationLookup)"`)
var reURLLit = regexp.MustCompile(`^https?://[^\s"]+$`)
var rePathLit = regexp.MustCompile(`^/[A-Za-z0-9_\-./{}]+$`)

type endpoint struct {
	mod, verb, framework, classPath, methodPath, comp, file, dir, conf string
	role                                                               string // api | ui | "" (unknown)
	line                                                               int
}

func annNames(as []java.Annotation) []string {
	var out []string
	for _, a := range as {
		out = append(out, a.Name)
	}
	return out
}

func javaSemantics(f *model.Facts, results []*fileResult, o Options, moduleOf func(string) string, getDep func(string) *model.Deployable) {
	cls := NewClassifier(o.Config)
	index := map[string]int{}
	byPkg := map[string]map[string]string{}
	entityTable := map[string]string{}
	var eps []endpoint
	include := func(r *fileResult) bool {
		return r != nil && r.jf != nil && (r.file.SourceSet != "test" || o.IncludeTests)
	}

	// pass 1: components, entity tables, application paths
	for _, r := range results {
		if !include(r) {
			continue
		}
		jf := r.jf
		mod := moduleOf(r.file.Path)
		for _, t := range jf.Types {
			stereo := annNames(t.Annotations)
			layer, reason := cls.Classify(jf.Package, t.Name, stereo, t.Extends)
			kind := t.Kind
			if !t.TopLevel {
				kind = "nested-" + kind
			}
			c := model.Component{FQN: t.FQN, Name: t.Name, Kind: kind, Language: r.file.Language, Package: jf.Package,
				SourceSet: r.file.SourceSet, Module: mod, Layer: layer, LayerReason: reason, Stereotypes: stereo,
				Extends: t.Extends, File: r.file.Path, Line: t.Line}
			if _, dup := index[t.FQN]; dup {
				f.Diagnostics = append(f.Diagnostics, model.Diagnostic{File: r.file.Path, Kind: "duplicate-type", Message: t.FQN + " is declared more than once"})
				continue
			}
			index[t.FQN] = len(f.Components)
			f.Components = append(f.Components, c)
			if t.TopLevel {
				if byPkg[jf.Package] == nil {
					byPkg[jf.Package] = map[string]string{}
				}
				byPkg[jf.Package][t.Name] = t.FQN
			}
			if a, ok := java.Has(t.Annotations, "Entity"); ok {
				table := t.Name
				conf := "low"
				if tb, ok := java.Has(t.Annotations, "Table"); ok {
					if n := tb.First("name"); n != "" {
						table, conf = n, "high"
						if s := tb.First("schema"); s != "" {
							table = s + "." + n
						}
					}
				}
				entityTable[t.Name] = table
				if alias := a.First("name", ""); alias != "" {
					entityTable[alias] = table
				}
				addTable(f, table, "uses", t.FQN, "JPA entity mapping", conf, r.file.Path, t.Line)
			}
			if a, ok := java.Has(t.Annotations, "ApplicationPath"); ok {
				d := getDep(mod)
				d.AppPaths = appendUnique(d.AppPaths, a.First("value", ""))
			}
		}
	}

	resolve := func(jf *java.File, name string) string {
		if strings.Contains(name, ".") {
			if _, ok := index[name]; ok {
				return name
			}
			name = name[strings.LastIndexByte(name, '.')+1:]
		}
		for _, im := range jf.Imports {
			if !im.Wildcard && !im.Static && strings.HasSuffix(im.Name, "."+name) {
				if _, ok := index[im.Name]; ok {
					return im.Name
				}
				return ""
			}
		}
		if fq, ok := byPkg[jf.Package][name]; ok {
			return fq
		}
		for _, im := range jf.Imports {
			if im.Wildcard && !im.Static {
				if fq, ok := byPkg[im.Name][name]; ok {
					return fq
				}
			}
		}
		return ""
	}
	topOf := func(fqn string) string {
		for {
			c, ok := index[fqn]
			if !ok {
				return fqn
			}
			comp := f.Components[c]
			if !strings.HasPrefix(comp.Kind, "nested-") {
				return fqn
			}
			fqn = fqn[:strings.LastIndexByte(fqn, '.')]
		}
	}

	// pass 2: dependencies, endpoints, contracts
	for _, r := range results {
		if !include(r) {
			continue
		}
		jf := r.jf
		var primary string
		for _, t := range jf.Types {
			if t.TopLevel {
				primary = t.FQN
				break
			}
		}
		if primary == "" {
			continue
		}
		mod := moduleOf(r.file.Path)
		best := map[string]model.CodeDep{}
		add := func(to, kind, conf string, line int) {
			if to == "" {
				return
			}
			to = topOf(to)
			if to == primary {
				return
			}
			if prev, ok := best[to]; ok && depRank[prev.Kind] >= depRank[kind] {
				return
			}
			best[to] = model.CodeDep{From: primary, To: to, Kind: kind, Confidence: conf, File: r.file.Path, Line: line}
		}
		explicit := map[string]bool{}
		for _, im := range jf.Imports {
			if im.Wildcard || im.Static {
				continue
			}
			explicit[im.Name[strings.LastIndexByte(im.Name, '.')+1:]] = true
			if _, ok := index[im.Name]; ok {
				add(im.Name, "import", "high", im.Line)
			}
		}
		for ref, line := range jf.Refs {
			if explicit[ref] {
				continue
			}
			if fq, ok := byPkg[jf.Package][ref]; ok {
				add(fq, "same-package", "medium", line)
				continue
			}
			for _, im := range jf.Imports {
				if im.Wildcard && !im.Static {
					if fq, ok := byPkg[im.Name][ref]; ok {
						add(fq, "wildcard", "medium", line)
						break
					}
				}
			}
		}
		for ref, line := range jf.FQNRefs {
			if _, ok := index[ref]; ok {
				add(ref, "fqn", "high", line)
			}
		}
		for _, t := range jf.Types {
			for _, e := range t.Extends {
				add(resolve(jf, e), "extends", "high", t.Line)
			}
			for _, fd := range t.Fields {
				if _, ok := java.Has(fd.Annotations, "Inject", "EJB", "Autowired", "Resource", "PersistenceContext"); ok {
					add(resolve(jf, fd.Type), "inject", "high", fd.Line)
				}
			}
			typeContracts(f, r, mod, t, &eps)
		}
		for _, d := range best {
			f.Dependencies = append(f.Dependencies, d)
		}
		// literals: JNDI names, SQL tables, outbound URLs
		httpClient := false
		for _, h := range r.hits {
			if h.Rule.ID == "JAVA-HTTP-CLIENT" {
				httpClient = true
			}
		}
		for _, l := range jf.Literals {
			v := l.Value
			switch {
			case strings.HasPrefix(v, "java:"):
				addContract(f, "jndi", "uses", v, model.NormalizeJNDI(v), primary, "string literal", "medium", r.file.Path, l.Line)
			case httpClient && reURLLit.MatchString(v) && model.LiteralSegments(model.NormalizePath(pathOfURL(v))) > 0:
				host, p := splitHostPath(v)
				addContract(f, "http", "calls", "ANY "+model.NormalizePath(p), model.NormalizePath(p), primary, "URL literal", "medium", r.file.Path, l.Line)
				f.Contracts[len(f.Contracts)-1].Host = host
				f.Contracts[len(f.Contracts)-1].Transport = model.URLTransport(v)
			case httpClient && rePathLit.MatchString(v) && model.LiteralSegments(model.NormalizePath(v)) >= 2:
				addContract(f, "http", "calls", "ANY "+model.NormalizePath(v), model.NormalizePath(v), primary, "path literal", "low", r.file.Path, l.Line)
			default:
				for _, tr := range sqlparse.Tables(v) {
					table, detail := tr.Table, "SQL in string literal"
					if t, ok := entityTable[tr.Table]; ok {
						table, detail = t, "JPQL entity "+tr.Table
					}
					addTable(f, table, tr.Op, primary, detail, "medium", r.file.Path, l.Line)
				}
			}
		}
		for _, l := range jf.Lookups {
			if !strings.HasPrefix(l.Value, "java:") {
				addContract(f, "jndi", "uses", l.Value, model.NormalizeJNDI(l.Value), primary, "lookup()", "high", r.file.Path, l.Line)
			}
		}
	}

	// compose endpoint paths with deployment information
	for _, e := range eps {
		d := getDep(e.mod)
		app := ""
		if e.framework == "jaxrs" && len(d.AppPaths) > 0 {
			app = d.AppPaths[0]
		}
		var key string
		if e.dir == "exposes" {
			key = e.verb + " " + model.JoinPath(d.ContextRoot, app, e.classPath, e.methodPath)
		} else {
			key = e.verb + " " + model.JoinPath(e.classPath, e.methodPath)
		}
		match := model.NormalizePath(model.JoinPath(e.classPath, e.methodPath))
		detail := e.framework
		if e.dir == "exposes" && d.ContextRoot == "" && (d.Kind == "war" || d.Kind == "ear") {
			detail += " (context root unknown)"
		} else if e.dir == "exposes" && d.RootSource == "inferred from WAR name" {
			detail += " (context root inferred)"
		}
		addContract(f, "http", e.dir, key, match, e.comp, detail, e.conf, e.file, e.line)
		if e.dir == "exposes" && len(f.Contracts) > 0 && f.Contracts[len(f.Contracts)-1].Key == key {
			f.Contracts[len(f.Contracts)-1].Role = e.role
		}
	}
}

func typeContracts(f *model.Facts, r *fileResult, mod string, t *java.Type, eps *[]endpoint) {
	file := r.file.Path
	client := false
	if _, ok := java.Has(t.Annotations, "FeignClient", "RegisterRestClient"); ok {
		client = true
	}
	dir := "exposes"
	if client {
		dir = "calls"
	}
	classPath, framework := "", ""
	if r.file.Language == "C#" {
		aspnetContracts(r, mod, t, eps)
		return
	}
	if a, ok := java.Has(t.Annotations, "Path"); ok {
		classPath, framework = a.First("value", ""), "jaxrs"
	}
	if a, ok := java.Has(t.Annotations, "RequestMapping"); ok {
		classPath, framework = a.First("value", "path", ""), "spring"
	}
	if a, ok := java.Has(t.Annotations, "FeignClient"); ok {
		classPath = a.First("path")
	}
	_, restClass := java.Has(t.Annotations, "RestController")
	_, mvcClass := java.Has(t.Annotations, "Controller")
	_, classBody := java.Has(t.Annotations, "ResponseBody")
	for _, m := range t.Methods {
		verb, paths, fw := "", []string{}, framework
		role := ""
		switch {
		case restClass || classBody || framework == "jaxrs":
			role = "api"
		case mvcClass:
			role = "ui" // @Controller methods render views unless they say otherwise
		}
		if _, ok := java.Has(m.Annotations, "ResponseBody"); ok {
			role = "api"
		}
		for _, a := range m.Annotations {
			if v, ok := httpVerbs[a.Name]; ok {
				verb = v
				if strings.HasSuffix(a.Name, "Mapping") {
					fw = "spring"
					paths = append(paths, a.Values("value", "path", "")...)
				} else if fw == "" {
					fw = "jaxrs"
				}
			}
			if strings.Contains(a.Args, "application/json") || strings.Contains(a.Args, "APPLICATION_JSON") || strings.Contains(a.Args, "application/xml") {
				role = "api"
			}
			if a.Name == "RequestMapping" {
				fw = "spring"
				paths = append(paths, a.Values("value", "path", "")...)
				if mm := reReqMethod.FindStringSubmatch(a.Args); mm != nil {
					verb = mm[1]
				} else if verb == "" {
					verb = "ANY"
				}
			}
			if a.Name == "Path" {
				paths = append(paths, a.First("value", ""))
			}
			if a.Name == "JmsListener" {
				d := a.First("destination")
				addContract(f, "jms", "subscribes", d, model.NormalizeJNDI(d), t.FQN, "@JmsListener", "high", file, a.Line)
			}
			if a.Name == "KafkaListener" {
				for _, tp := range a.Values("topics", "value", "") {
					addContract(f, "kafka", "subscribes", tp, tp, t.FQN, "@KafkaListener", "high", file, a.Line)
				}
			}
		}
		if verb == "" {
			continue // sub-resource locator or plain method
		}
		if fw == "jaxrs" {
			role = "api"
		}
		if len(paths) == 0 {
			paths = []string{""}
		}
		for _, p := range paths {
			*eps = append(*eps, endpoint{mod: mod, verb: verb, framework: fw, classPath: classPath, methodPath: p,
				comp: t.FQN, file: file, dir: dir, conf: "high", line: m.Line, role: role})
		}
	}
	if a, ok := java.Has(t.Annotations, "WebServlet"); ok {
		for _, p := range a.Values("urlPatterns", "value", "") {
			*eps = append(*eps, endpoint{mod: mod, verb: "ANY", framework: "servlet", methodPath: p,
				comp: t.FQN, file: file, dir: "exposes", conf: "high", line: a.Line})
		}
	}
	if a, ok := java.Has(t.Annotations, "WebService"); ok && t.Kind != "interface" {
		name := a.First("serviceName")
		if name == "" {
			name = a.First("name")
		}
		if name == "" {
			name = t.Name + "Service"
		}
		addContract(f, "soap", "exposes", name, strings.ToLower(name), t.FQN, "@WebService", "high", file, a.Line)
	}
	if a, ok := java.Has(t.Annotations, "WebServiceClient"); ok {
		name := a.First("name")
		addContract(f, "soap", "calls", name, strings.ToLower(name), t.FQN, "@WebServiceClient (generated client)", "high", file, a.Line)
	}
	if a, ok := java.Has(t.Annotations, "MessageDriven"); ok {
		dest := a.First("mappedName")
		if m := reMDBDest.FindStringSubmatch(a.Args); m != nil {
			dest = m[1] + m[2]
		}
		if dest != "" {
			addContract(f, "jms", "subscribes", dest, model.NormalizeJNDI(dest), t.FQN, "message-driven bean", "high", file, a.Line)
		}
	}
	if _, ok := java.Has(t.Annotations, "Remote"); ok && t.Kind == "interface" {
		addContract(f, "ejb", "exposes", t.Name, strings.ToLower(t.Name), t.FQN, "@Remote interface", "medium", file, t.Line)
	}
	for _, fd := range t.Fields {
		if a, ok := java.Has(fd.Annotations, "Resource"); ok {
			n := a.First("lookup", "mappedName", "name")
			if n == "" {
				continue
			}
			kind := "jndi"
			switch fd.Type {
			case "Queue", "Topic", "Destination":
				kind = "jms"
			}
			addContract(f, kind, "uses", n, model.NormalizeJNDI(n), t.FQN, "@Resource "+fd.Type, "high", file, a.Line)
		}
		if a, ok := java.Has(fd.Annotations, "EJB"); ok {
			if n := a.First("lookup", "mappedName"); n != "" {
				addContract(f, "ejb", "calls", n, model.NormalizeJNDI(n), t.FQN, "@EJB lookup", "medium", file, a.Line)
			}
		}
		for _, a := range fd.Annotations {
			if a.Name == "JoinTable" || a.Name == "CollectionTable" {
				if n := a.First("name"); n != "" {
					addTable(f, n, "uses", t.FQN, "@"+a.Name, "high", file, a.Line)
				}
			}
		}
	}
}

func addContract(f *model.Facts, kind, dir, key, match, comp, detail, conf, file string, line int) {
	if strings.TrimSpace(key) == "" {
		return
	}
	f.Contracts = append(f.Contracts, model.Contract{Kind: kind, Direction: dir, Key: key, Match: match, Component: comp,
		Detail: detail, Confidence: conf, File: file, Line: line})
}

func addTable(f *model.Facts, table, op, comp, detail, conf, file string, line int) {
	t := strings.ToUpper(strings.Trim(table, `"[]`+"`"))
	m := t
	if i := strings.LastIndexByte(m, '.'); i >= 0 {
		m = m[i+1:]
	}
	addContract(f, "db-table", op, t, m, comp, detail, conf, file, line)
}

func pathOfURL(u string) string {
	_, p := splitHostPath(u)
	return p
}

func splitHostPath(u string) (string, string) {
	rest := u[strings.Index(u, "://")+3:]
	if at := strings.LastIndex(rest, "@"); at >= 0 && at < strings.IndexAny(rest+"/", "/") {
		rest = rest[at+1:]
	}
	host, p := rest, "/"
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		host, p = rest[:j], rest[j:]
	}
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host, p
}

var csVerbs = map[string]string{"HttpGet": "GET", "HttpPost": "POST", "HttpPut": "PUT", "HttpDelete": "DELETE",
	"HttpPatch": "PATCH", "HttpHead": "HEAD", "HttpOptions": "OPTIONS", "AcceptVerbs": "ANY"}

var reCsToken = regexp.MustCompile(`(?i)\[(controller|action|area)\]`)

// aspnetContracts handles ASP.NET (Core, MVC, Web API) attribute routing and
// conventional routing for controllers without route attributes.
func aspnetContracts(r *fileResult, mod string, t *java.Type, eps *[]endpoint) {
	file := r.file.Path
	ctrl := strings.TrimSuffix(t.Name, "Controller")
	isController := strings.HasSuffix(t.Name, "Controller")
	for _, e := range t.Extends {
		switch e {
		case "Controller", "ControllerBase", "ApiController", "ODataController":
			isController = true
		}
	}
	if _, ok := java.Has(t.Annotations, "ApiController"); ok {
		isController = true
	}
	if !isController || t.Kind == "interface" {
		return
	}
	expand := func(p, action string) string {
		return reCsToken.ReplaceAllStringFunc(p, func(m string) string {
			switch strings.ToLower(strings.Trim(m, "[]")) {
			case "controller":
				return ctrl
			case "action":
				return action
			}
			return "{area}"
		})
	}
	var classPaths []string
	for _, a := range t.Annotations {
		if a.Name == "Route" || a.Name == "RoutePrefix" {
			classPaths = append(classPaths, a.Values("", "template", "Template")...)
		}
	}
	if len(classPaths) == 0 {
		classPaths = []string{""}
	}
	attributeRouted := len(classPaths) > 1 || classPaths[0] != ""
	for _, m := range t.Methods {
		verb := ""
		var paths []string
		for _, a := range m.Annotations {
			if v, ok := csVerbs[a.Name]; ok {
				verb = v
				paths = append(paths, a.Values("", "template", "Template")...)
				attributeRouted = true
			}
			if a.Name == "Route" {
				paths = append(paths, a.Values("", "template", "Template")...)
				attributeRouted = true
				if verb == "" {
					verb = "ANY"
				}
			}
		}
		if verb == "" {
			continue
		}
		if len(paths) == 0 {
			paths = []string{""}
		}
		for _, cp := range classPaths {
			for _, p := range paths {
				base := cp
				if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~/") {
					base, p = "", strings.TrimPrefix(p, "~")
				}
				*eps = append(*eps, endpoint{mod: mod, verb: verb, framework: "aspnet", classPath: expand(base, m.Name),
					methodPath: expand(p, m.Name), comp: t.FQN, file: file, dir: "exposes", conf: "high", line: m.Line})
			}
		}
	}
	if attributeRouted {
		return
	}
	// Conventional routing "{controller}/{action}/{id?}" (classic ASP.NET MVC).
	for _, m := range t.Methods {
		if !m.Public || m.Name == t.Name || m.Name == "Dispose" {
			continue
		}
		action := strings.TrimSuffix(m.Name, "Async")
		*eps = append(*eps, endpoint{mod: mod, verb: "ANY", framework: "aspnet-mvc conventional route", classPath: ctrl,
			methodPath: action, comp: t.FQN, file: file, dir: "exposes", conf: "low", line: m.Line})
	}
}

type route struct {
	verb, path, framework string
	line                  int
}

var reMinimal = regexp.MustCompile(`\.Map(Get|Post|Put|Delete|Patch|Methods)\s*\(\s*"([^"]*)"`)

func minimalAPIs(st *lang.Stripped) []route {
	var out []route
	for _, m := range reMinimal.FindAllStringSubmatchIndex(st.Code, -1) {
		v := strings.ToUpper(st.Code[m[2]:m[3]])
		if v == "METHODS" {
			v = "ANY"
		}
		out = append(out, route{v, st.Code[m[4]:m[5]], "aspnet minimal API", st.Line(m[0])})
	}
	return out
}
