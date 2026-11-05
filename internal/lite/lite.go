// Package lite extracts module-level structure from languages where the
// architecture is expressed through files, imports and route declarations
// rather than through classes: JavaScript/TypeScript, Python, PHP, Go, Ruby.
package lite

import (
	"path"
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
)

type Import struct {
	Spec string // as written: "./x", "pkg.mod", "App\\Http\\X", "example.com/mod/pkg"
	Rel  int    // Python relative level (number of leading dots)
	Line int
	// Optional imports are candidates ("from pkg import name" where name may be
	// a submodule); when one resolves, the plain package import is redundant.
	Optional bool
}

type Route struct {
	Verb, Path, Framework string
	Line                  int
}

type Table struct {
	Name, Op, Detail string
	Line             int
}

type File struct {
	Namespace string   // PHP namespace
	Classes   []string // PHP / Python / TS class names (informative)
	Imports   []Import
	Routes    []Route
	Tables    []Table
}

// Supported reports whether the language has a lite extractor.
func Supported(language string) bool {
	switch language {
	case "JavaScript", "TypeScript", "Vue", "Python", "PHP", "Go", "Ruby":
		return true
	}
	return false
}

var (
	// JavaScript / TypeScript
	reJSImport  = regexp.MustCompile(`(?m)(?:^|[^\w.])(?:import\s+(?:[\w*{}\s,]+\s+from\s+)?|export\s+[\w*{}\s,]+\s+from\s+|require\s*\(\s*|import\s*\(\s*)['"]([^'"\n]+)['"]`)
	reJSRoute   = regexp.MustCompile(`\b(\w+)\.(get|post|put|delete|patch|all|options|head)\s*\(\s*['"` + "`" + `](/[^'"` + "`" + `\s]*)['"` + "`" + `]`)
	reNestCtrl  = regexp.MustCompile(`@Controller\(\s*(?:['"` + "`" + `]([^'"` + "`" + `]*)['"` + "`" + `]|\{[^}]*path\s*:\s*['"]([^'"]*)['"][^}]*\})?\s*\)`)
	reNestRoute = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|All|Options|Head)\(\s*(?:['"` + "`" + `]([^'"` + "`" + `]*)['"` + "`" + `])?\s*\)`)
	reTSEntity  = regexp.MustCompile(`@Entity\(\s*(?:['"](\w+)['"]|\{[^}]*name\s*:\s*['"](\w+)['"])`)
	reSeqDefine = regexp.MustCompile(`\.define\(\s*['"](\w+)['"]\s*,\s*\{`)
	reKnexTable = regexp.MustCompile(`\.createTable\(\s*['"](\w+)['"]`)
	reJSClass   = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?(?:abstract\s+)?class\s+(\w+)`)

	// Python
	rePyFrom    = regexp.MustCompile(`(?m)^[ \t]*from[ \t]+(\.*)([\w.]*)[ \t]+import[ \t]+(\([^)]*\)|[^\n]+)`)
	rePyImport  = regexp.MustCompile(`(?m)^[ \t]*import[ \t]+([\w.]+(?:[ \t]*,[ \t]*[\w.]+)*)`)
	rePyRoute   = regexp.MustCompile(`@(\w+)\.(route|get|post|put|delete|patch|api_route|websocket)\(\s*['"]([^'"]*)['"]([^)]*)\)`)
	rePyMethods = regexp.MustCompile(`methods\s*=\s*\[([^\]]*)\]`)
	rePyPrefix  = regexp.MustCompile(`(\w+)\s*=\s*(?:APIRouter|Blueprint)\(([^)]*)\)`)
	rePyPrefArg = regexp.MustCompile(`(?:url_)?prefix\s*=\s*['"]([^'"]*)['"]`)
	reDjango    = regexp.MustCompile(`\b(?:path|re_path|url)\(\s*r?['"]([^'"]*)['"]`)
	rePyTable   = regexp.MustCompile(`(?m)^\s*(?:db_table|__tablename__)\s*=\s*['"]([\w.]+)['"]`)
	rePyClass   = regexp.MustCompile(`(?m)^class\s+(\w+)`)

	// PHP
	rePHPNs     = regexp.MustCompile(`(?m)^\s*namespace\s+([\w\\]+)\s*;`)
	rePHPUse    = regexp.MustCompile(`(?m)^\s*use\s+([\w\\]+)(?:\s+as\s+\w+)?\s*;`)
	rePHPClass  = regexp.MustCompile(`(?m)^\s*(?:final\s+|abstract\s+|readonly\s+)*(?:class|interface|trait|enum)\s+(\w+)`)
	reLaravel   = regexp.MustCompile(`Route::(get|post|put|patch|delete|options|any|match|resource|apiResource|view|redirect)\(\s*(?:\[[^\]]*\]\s*,\s*)?['"]([^'"]*)['"]`)
	reSymfony   = regexp.MustCompile(`(?:#\[Route\(|@Route\()\s*(?:path\s*[:=]\s*)?['"]([^'"]*)['"]([^)]*)\)`)
	reSymMeth   = regexp.MustCompile(`methods\s*[:=]\s*[\[{]([^\]}]*)[\]}]`)
	rePHPTable  = regexp.MustCompile(`protected\s+\$table\s*=\s*['"]([\w.]+)['"]`)
	rePHPSchema = regexp.MustCompile(`Schema::(create|table)\(\s*['"](\w+)['"]`)

	// Go
	reGoImportBlk = regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
	reGoImportOne = regexp.MustCompile(`(?m)^\s*import\s+(?:\w+\s+)?"([^"]+)"`)
	reGoQuoted    = regexp.MustCompile(`"([^"]+)"`)
	reGoGroup     = regexp.MustCompile(`(\w+)\s*:?=\s*(\w+)\.(?:Group|Route|PathPrefix)\(\s*"([^"]*)"`)
	reGoRoute     = regexp.MustCompile(`\b(\w+)\.(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS|Any|Get|Post|Put|Delete|Patch|Handle|HandleFunc|Methods)\(\s*"(/[^"]*)"`)
	reGoTable     = regexp.MustCompile(`TableName\(\)\s*string\s*\{\s*return\s+"(\w+)"`)

	// Ruby on Rails
	reRbRoute     = regexp.MustCompile(`(?m)^\s*(get|post|put|patch|delete|match)\s+['"]([^'"]+)['"]`)
	reRbResources = regexp.MustCompile(`(?m)^\s*resources?\s+:(\w+)`)
	reRbTable     = regexp.MustCompile(`create_table\s+[:'"](\w+)`)
	reRbRequire   = regexp.MustCompile(`(?m)^\s*require_relative\s+['"]([^'"]+)['"]`)
)

// Parse extracts imports, routes and table mappings. Patterns run on text
// without comments (string literals kept).
func Parse(language, rel string, st *lang.Stripped) *File {
	f := &File{}
	code := st.Code
	switch language {
	case "JavaScript", "TypeScript", "Vue":
		for _, m := range reJSImport.FindAllStringSubmatchIndex(code, -1) {
			f.Imports = append(f.Imports, Import{Spec: code[m[2]:m[3]], Line: st.Line(m[0])})
		}
		prefix := ""
		if m := reNestCtrl.FindStringSubmatch(code); m != nil {
			prefix = m[1] + m[2]
			for _, r := range reNestRoute.FindAllStringSubmatchIndex(code, -1) {
				p := ""
				if r[4] >= 0 {
					p = code[r[4]:r[5]]
				}
				f.Routes = append(f.Routes, Route{strings.ToUpper(code[r[2]:r[3]]), joinPath(prefix, p), "nestjs", st.Line(r[0])})
			}
		}
		for _, m := range reJSRoute.FindAllStringSubmatchIndex(code, -1) {
			obj := code[m[2]:m[3]]
			if isJSNonRouter(obj) {
				continue
			}
			f.Routes = append(f.Routes, Route{verbOf(code[m[4]:m[5]]), code[m[6]:m[7]], "express-style router", st.Line(m[0])})
		}
		for _, m := range reTSEntity.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{sub(code, m, 1) + sub(code, m, 2), "uses", "TypeORM @Entity", st.Line(m[0])})
		}
		for _, m := range reSeqDefine.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "uses", "Sequelize model", st.Line(m[0])})
		}
		for _, m := range reKnexTable.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "defines", "migration createTable", st.Line(m[0])})
		}
		for _, m := range reJSClass.FindAllStringSubmatch(code, -1) {
			f.Classes = append(f.Classes, m[1])
		}
	case "Python":
		for _, m := range rePyFrom.FindAllStringSubmatchIndex(code, -1) {
			spec, rel, line := code[m[4]:m[5]], m[3]-m[2], st.Line(m[0])
			// "from pkg import mod" may import submodules: try each name first
			for _, n := range strings.Split(strings.Trim(code[m[6]:m[7]], "()"), ",") {
				n = strings.TrimSpace(strings.Split(strings.TrimSpace(n), " ")[0])
				if n != "" && n != "*" {
					full := n
					if spec != "" {
						full = spec + "." + n
					}
					f.Imports = append(f.Imports, Import{Spec: full, Rel: rel, Line: line, Optional: true})
				}
			}
			f.Imports = append(f.Imports, Import{Spec: spec, Rel: rel, Line: line})
		}
		for _, m := range rePyImport.FindAllStringSubmatchIndex(code, -1) {
			for _, part := range strings.Split(code[m[2]:m[3]], ",") {
				f.Imports = append(f.Imports, Import{Spec: strings.TrimSpace(part), Line: st.Line(m[0])})
			}
		}
		prefixes := map[string]string{}
		for _, m := range rePyPrefix.FindAllStringSubmatch(code, -1) {
			if p := rePyPrefArg.FindStringSubmatch(m[2]); p != nil {
				prefixes[m[1]] = p[1]
			}
		}
		for _, m := range rePyRoute.FindAllStringSubmatchIndex(code, -1) {
			obj, kind, p, rest := code[m[2]:m[3]], code[m[4]:m[5]], code[m[6]:m[7]], code[m[8]:m[9]]
			verb := strings.ToUpper(kind)
			fw := "fastapi/flask"
			if kind == "route" || kind == "api_route" {
				verb = "ANY"
				if mm := rePyMethods.FindStringSubmatch(rest); mm != nil {
					verb = strings.ToUpper(strings.Trim(strings.Split(mm[1], ",")[0], ` '"`))
				}
			}
			if kind == "websocket" {
				verb = "WS"
			}
			f.Routes = append(f.Routes, Route{verb, joinPath(prefixes[obj], p), fw, st.Line(m[0])})
		}
		if path.Base(rel) == "urls.py" {
			for _, m := range reDjango.FindAllStringSubmatchIndex(code, -1) {
				p := strings.TrimSuffix(strings.TrimPrefix(code[m[2]:m[3]], "^"), "$")
				f.Routes = append(f.Routes, Route{"ANY", "/" + p, "django urls (include() prefixes not resolved)", st.Line(m[0])})
			}
		}
		for _, m := range rePyTable.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "uses", "ORM table name", st.Line(m[0])})
		}
		for _, m := range rePyClass.FindAllStringSubmatch(code, -1) {
			f.Classes = append(f.Classes, m[1])
		}
	case "PHP":
		if m := rePHPNs.FindStringSubmatch(code); m != nil {
			f.Namespace = m[1]
		}
		for _, m := range rePHPUse.FindAllStringSubmatchIndex(code, -1) {
			f.Imports = append(f.Imports, Import{Spec: code[m[2]:m[3]], Line: st.Line(m[0])})
		}
		for _, m := range rePHPClass.FindAllStringSubmatch(code, -1) {
			f.Classes = append(f.Classes, m[1])
		}
		prefix := ""
		if strings.HasSuffix(rel, "routes/api.php") {
			prefix = "/api" // Laravel RouteServiceProvider convention
		}
		for _, m := range reLaravel.FindAllStringSubmatchIndex(code, -1) {
			v := strings.ToUpper(code[m[2]:m[3]])
			p := code[m[4]:m[5]]
			switch v {
			case "RESOURCE", "APIRESOURCE":
				// nested resources "albums.songs" -> /albums/{album}/songs
				parts := strings.Split(p, ".")
				for i := 0; i < len(parts)-1; i++ {
					parts[i] = parts[i] + "/{" + strings.TrimSuffix(parts[i], "s") + "}"
				}
				p = strings.Join(parts, "/")
				v = "ANY"
			case "ANY", "MATCH", "VIEW", "REDIRECT":
				v = "ANY"
			}
			f.Routes = append(f.Routes, Route{v, joinPath(prefix, p), "laravel", st.Line(m[0])})
		}
		for _, m := range reSymfony.FindAllStringSubmatchIndex(code, -1) {
			v := "ANY"
			if mm := reSymMeth.FindStringSubmatch(code[m[4]:m[5]]); mm != nil {
				v = strings.ToUpper(strings.Trim(strings.Split(mm[1], ",")[0], ` '"`))
			}
			f.Routes = append(f.Routes, Route{v, code[m[2]:m[3]], "symfony", st.Line(m[0])})
		}
		for _, m := range rePHPTable.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "uses", "Eloquent $table", st.Line(m[0])})
		}
		for _, m := range rePHPSchema.FindAllStringSubmatchIndex(code, -1) {
			op := "defines"
			if code[m[2]:m[3]] == "table" {
				op = "write"
			}
			f.Tables = append(f.Tables, Table{code[m[4]:m[5]], op, "migration", st.Line(m[0])})
		}
	case "Go":
		for _, b := range reGoImportBlk.FindAllStringSubmatchIndex(code, -1) {
			for _, q := range reGoQuoted.FindAllStringSubmatchIndex(code[b[2]:b[3]], -1) {
				f.Imports = append(f.Imports, Import{Spec: code[b[2]+q[2] : b[2]+q[3]], Line: st.Line(b[2] + q[0])})
			}
		}
		for _, m := range reGoImportOne.FindAllStringSubmatchIndex(code, -1) {
			f.Imports = append(f.Imports, Import{Spec: code[m[2]:m[3]], Line: st.Line(m[0])})
		}
		prefixes := map[string]string{}
		for _, m := range reGoGroup.FindAllStringSubmatch(code, -1) {
			prefixes[m[1]] = joinPath(prefixes[m[2]], m[3])
		}
		for _, m := range reGoRoute.FindAllStringSubmatchIndex(code, -1) {
			obj, v := code[m[2]:m[3]], strings.ToUpper(code[m[4]:m[5]])
			if v == "HANDLE" || v == "HANDLEFUNC" || v == "METHODS" {
				v = "ANY"
			}
			f.Routes = append(f.Routes, Route{v, joinPath(prefixes[obj], code[m[6]:m[7]]), "go router", st.Line(m[0])})
		}
		for _, m := range reGoTable.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "uses", "GORM TableName", st.Line(m[0])})
		}
	case "Ruby":
		for _, m := range reRbRequire.FindAllStringSubmatchIndex(code, -1) {
			f.Imports = append(f.Imports, Import{Spec: "./" + code[m[2]:m[3]], Line: st.Line(m[0])})
		}
		if path.Base(rel) == "routes.rb" {
			for _, m := range reRbRoute.FindAllStringSubmatchIndex(code, -1) {
				f.Routes = append(f.Routes, Route{verbOf(code[m[2]:m[3]]), joinPath(code[m[4]:m[5]]), "rails routes", st.Line(m[0])})
			}
			for _, m := range reRbResources.FindAllStringSubmatchIndex(code, -1) {
				f.Routes = append(f.Routes, Route{"ANY", "/" + code[m[2]:m[3]], "rails resources", st.Line(m[0])})
			}
		}
		for _, m := range reRbTable.FindAllStringSubmatchIndex(code, -1) {
			f.Tables = append(f.Tables, Table{code[m[2]:m[3]], "defines", "rails migration", st.Line(m[0])})
		}
	}
	return f
}

// isJSNonRouter filters common objects whose .get()/.post() are not HTTP routes.
func isJSNonRouter(obj string) bool {
	switch obj {
	case "map", "Map", "cache", "headers", "params", "searchParams", "localStorage", "sessionStorage", "config",
		"axios", "http", "httpClient", "$http", "fetch", "request", "superagent", "api", "client", "this", "req", "res", "ctx",
		"formData", "store", "storage", "redis", "process", "env", "Reflect", "window", "document", "url", "URLSearchParams":
		return true
	}
	return false
}

func verbOf(v string) string {
	v = strings.ToUpper(v)
	if v == "ALL" || v == "MATCH" {
		return "ANY"
	}
	return v
}

func sub(s string, m []int, g int) string {
	if m[2*g] < 0 {
		return ""
	}
	return s[m[2*g]:m[2*g+1]]
}

func joinPath(parts ...string) string {
	var segs []string
	for _, p := range parts {
		p = strings.Trim(p, "/ ")
		if p != "" {
			segs = append(segs, p)
		}
	}
	return "/" + strings.Join(segs, "/")
}

// IsOutboundClient reports outbound HTTP call sites (fetch/axios/requests/Guzzle/http.Get...).
var reOutbound = regexp.MustCompile(`\b(?:fetch|axios(?:\.\w+)?|requests\.\w+|httpx\.\w+|urlopen|urllib3?\.\w+|aiohttp\.\w+|http\.(?:Get|Post|Head|NewRequest\w*)|Http::\w+|\$http\.\w+|this\.http\.\w+|ky(?:\.\w+)?|got(?:\.\w+)?|curl_init|file_get_contents|HttpClient|RestClient|RestRequest|WebClient|HttpWebRequest|WebRequest\.Create|GetAsync|PostAsync|PutAsync|DeleteAsync|SendAsync|GetFromJsonAsync|PostAsJsonAsync|Net::HTTP\.\w+|HTTParty\.\w+|Faraday\.\w+|resty\.\w+)\s*[(<]|new\s+(?:GuzzleHttp\\)?Client\s*\(`)

func HasOutboundClient(code string) bool { return reOutbound.MatchString(code) }
