package scan

import (
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
	"github.com/oskillarrr/traverse/internal/lite"
	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/sqlparse"
)

var (
	reCreateTbl = regexp.MustCompile(`(?i)\bcreate\s+(?:global\s+temporary\s+)?table\s+(?:if\s+not\s+exists\s+)?([A-Za-z_"\[][\w$#".\]]*)`)
	reExecSQL   = regexp.MustCompile(`(?is)EXEC\s+SQL(.*?)END-EXEC`)
	reDQ        = regexp.MustCompile(`"((?:[^"\\\n]|\\.)*)"`)
	reSQ        = regexp.MustCompile(`'((?:[^'\\\n]|\\.)*)'`)
)

// namespaceHosts appear in XML namespaces and schemas, not real integrations.
var namespaceHosts = map[string]bool{
	"www.w3.org": true, "java.sun.com": true, "xmlns.jcp.org": true, "jakarta.ee": true, "maven.apache.org": true,
	"www.springframework.org": true, "schemas.xmlsoap.org": true, "schemas.microsoft.com": true, "json-schema.org": true,
	"purl.org": true, "docs.oasis-open.org": true, "www.hibernate.org": true, "localhost": true, "127.0.0.1": true,
	"example.com": true, "www.example.com": true, "tempuri.org": true, "eidas.europa.eu": true, "uri.etsi.org": true,
	"www.oasis-open.org": true, "xmlns.oracle.com": true, "struts.apache.org": true, "jakarta.apache.org": true,
	"www.ibatis.com": true, "ibatis.apache.org": true, "mybatis.org": true, "www.mybatis.org": true, "logging.apache.org": true,
	"cxf.apache.org": true, "camel.apache.org": true, "www.oracle.com": true, "openoffice.org": true, "0.0.0.0": true,
}

// otherContracts extracts contracts from non-Java sources: SQL DDL/DML,
// embedded SQL in COBOL/PL/I and SQL literals in other languages.
func otherContracts(rel string, l lang.Language, text string, st *lang.Stripped) []model.Contract {
	var out []model.Contract
	add := func(table, op, detail, conf string, off int) {
		full, m := sqlparse.Normalize(table)
		t := full
		out = append(out, model.Contract{Kind: "db-table", Direction: op, Key: t, Match: m, Detail: detail, Confidence: conf, File: rel, Line: st.Line(off)})
	}
	switch l.Name {
	case "SQL", "PL/SQL", "Transact-SQL":
		created := map[string]bool{}
		for _, m := range reCreateTbl.FindAllStringSubmatchIndex(st.Code, -1) {
			name := st.Code[m[2]:m[3]]
			created[strings.ToUpper(name)] = true
			add(name, "defines", "CREATE TABLE", "high", m[0])
		}
		for _, stmt := range splitStatements(st.Code) {
			for _, tr := range sqlparse.Tables(stmt.text) {
				if !created[strings.ToUpper(tr.Table)] {
					add(tr.Table, tr.Op, "SQL script", "medium", stmt.off)
				}
			}
		}
	case "COBOL", "PL/I", "RPG", "C":
		for _, m := range reExecSQL.FindAllStringSubmatchIndex(st.Code, -1) {
			for _, tr := range sqlparse.Tables(st.Code[m[2]:m[3]]) {
				add(tr.Table, tr.Op, "EXEC SQL", "high", m[0])
			}
		}
	case "C#", "VB.NET/VB6", "PHP", "Python", "JavaScript", "TypeScript", "Kotlin", "Groovy", "Scala", "PowerBuilder", "Delphi/Pascal", "Go", "Ruby":
		client := lite.HasOutboundClient(st.Code)
		for _, re := range []*regexp.Regexp{reDQ, reSQ} {
			for _, m := range re.FindAllStringSubmatchIndex(st.Code, -1) {
				lit := st.Code[m[2]:m[3]]
				if reURLLit.MatchString(lit) {
					if !client {
						continue // a URL in a file without HTTP client calls is content, not an integration
					}
					host, p := splitHostPath(lit)
					if namespaceHosts[strings.ToLower(host)] {
						continue
					}
					np := model.NormalizePath(p)
					if model.LiteralSegments(np) == 0 {
						continue
					}
					out = append(out, model.Contract{Kind: "http", Direction: "calls", Key: "ANY " + np, Match: np, Detail: "URL literal",
						Confidence: "medium", Host: host, Transport: model.URLTransport(lit), File: rel, Line: st.Line(m[0])})
					continue
				}
				for _, tr := range sqlparse.Tables(lit) {
					add(tr.Table, tr.Op, "SQL in string literal", "medium", m[0])
				}
			}
		}
	}
	return out
}

type stmt struct {
	text string
	off  int
}

func splitStatements(code string) []stmt {
	var out []stmt
	start := 0
	for i := 0; i < len(code); i++ {
		if code[i] == ';' || (code[i] == '/' && (i == 0 || code[i-1] == '\n') && (i+1 == len(code) || code[i+1] == '\n' || code[i+1] == '\r')) {
			out = append(out, stmt{code[start:i], start})
			start = i + 1
		}
	}
	if start < len(code) {
		out = append(out, stmt{code[start:], start})
	}
	return out
}
