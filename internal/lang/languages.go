// Package lang knows which language a file is written in, how its comments
// and strings look, and how to strip comments while preserving line numbers.
package lang

import (
	"path/filepath"
	"strings"
)

// Category groups languages for reporting.
const (
	CatCode   = "code"
	CatMarkup = "markup"
	CatConfig = "config"
	CatBuild  = "build"
	CatData   = "data"
	CatDocs   = "docs"
)

type Language struct {
	Name     string
	Category string
	Syntax   *Syntax
}

var byExt = map[string]Language{}
var byName = map[string]Language{} // exact file names (Dockerfile, Jenkinsfile...)

func reg(name, cat string, syn *Syntax, exts ...string) {
	l := Language{Name: name, Category: cat, Syntax: syn}
	for _, e := range exts {
		if strings.HasPrefix(e, ".") {
			byExt[e] = l
		} else {
			byName[strings.ToLower(e)] = l
		}
	}
}

func init() {
	// JVM
	reg("Java", CatCode, CStyle, ".java")
	reg("Kotlin", CatCode, CStyle, ".kt", ".kts")
	reg("Groovy", CatCode, CStyle, ".groovy", ".gvy", ".gsp")
	reg("Scala", CatCode, CStyle, ".scala")
	reg("Clojure", CatCode, Lisp, ".clj", ".cljs", ".cljc")
	// Web / server pages
	reg("JSP", CatMarkup, JSP, ".jsp", ".jspx", ".jspf", ".tag", ".tagx")
	reg("JSF/Facelets", CatMarkup, XMLSyntax, ".xhtml")
	reg("ASP.NET", CatMarkup, ASPX, ".aspx", ".ascx", ".asmx", ".ashx", ".master", ".cshtml", ".vbhtml", ".razor")
	reg("Classic ASP", CatMarkup, ASPX, ".asp")
	reg("PHP", CatCode, PHP, ".php", ".php3", ".php4", ".php5", ".phtml", ".inc")
	reg("HTML", CatMarkup, XMLSyntax, ".html", ".htm")
	reg("CSS", CatMarkup, CBlockOnly, ".css")
	reg("SCSS/LESS", CatMarkup, CStyle, ".scss", ".sass", ".less")
	reg("Velocity/Freemarker", CatMarkup, XMLSyntax, ".vm", ".ftl")
	// JS family
	reg("JavaScript", CatCode, CStyle, ".js", ".mjs", ".cjs", ".jsx")
	reg("TypeScript", CatCode, CStyle, ".ts", ".tsx")
	reg("Vue", CatCode, CStyle, ".vue")
	reg("ActionScript/Flex", CatCode, CStyle, ".as", ".mxml")
	// .NET / Microsoft
	reg("C#", CatCode, CStyle, ".cs")
	reg("VB.NET/VB6", CatCode, VB, ".vb", ".bas", ".cls", ".frm", ".ctl", ".vbs")
	reg("F#", CatCode, CStyle, ".fs", ".fsx")
	reg("PowerShell", CatCode, PowerShell, ".ps1", ".psm1", ".psd1")
	reg("Batch", CatCode, Batch, ".bat", ".cmd")
	// Native / systems
	reg("C", CatCode, CStyle, ".c", ".h")
	reg("C++", CatCode, CStyle, ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx")
	reg("Objective-C", CatCode, CStyle, ".m", ".mm")
	reg("Go", CatCode, GoSyntax, ".go")
	reg("Rust", CatCode, CStyle, ".rs")
	reg("Swift", CatCode, CStyle, ".swift")
	reg("Dart", CatCode, CStyle, ".dart")
	// Scripting
	reg("Python", CatCode, Python, ".py", ".pyw")
	reg("Ruby", CatCode, Hash, ".rb", ".erb", ".rake")
	reg("Perl", CatCode, Hash, ".pl", ".pm")
	reg("Shell", CatCode, Hash, ".sh", ".bash", ".ksh", ".csh", ".zsh")
	reg("Lua", CatCode, SQLSyntax, ".lua")
	reg("R", CatCode, Hash, ".r")
	// Legacy / mainframe
	reg("COBOL", CatCode, COBOL, ".cbl", ".cob", ".cpy", ".cobol", ".cbx")
	reg("JCL", CatCode, JCL, ".jcl", ".proc", ".jobs")
	reg("CICS BMS", CatMarkup, Asm, ".bms")
	reg("Assembler (HLASM)", CatCode, Asm, ".asm", ".mac", ".hlasm", ".mlc")
	reg("REXX", CatCode, CBlockOnly, ".rexx", ".rex")
	reg("Control cards", CatConfig, None, ".ctlcard")
	reg("CICS CSD", CatConfig, None, ".csd")
	reg("Prisma", CatConfig, CStyle, ".prisma")
	reg("PL/I", CatCode, CBlockOnly, ".pli", ".pl1")
	reg("RPG", CatCode, Hash, ".rpg", ".rpgle", ".sqlrpgle")
	reg("Natural", CatCode, Hash, ".nsp", ".nsn")
	reg("Delphi/Pascal", CatCode, Pascal, ".pas", ".dpr", ".dpk", ".dfm", ".lpr")
	reg("PowerBuilder", CatCode, CStyle, ".srw", ".sru", ".srd", ".srf", ".srm", ".srs")
	reg("Fortran", CatCode, Fortran, ".f", ".for", ".f77", ".f90", ".f95")
	reg("ABAP", CatCode, ABAP, ".abap")
	// Data / database
	reg("SQL", CatCode, SQLSyntax, ".sql", ".ddl", ".dml")
	reg("PL/SQL", CatCode, SQLSyntax, ".pls", ".plb", ".pks", ".pkb", ".prc", ".fnc", ".trg", ".vw", ".tps", ".tpb")
	reg("Transact-SQL", CatCode, SQLSyntax, ".tsql")
	// Contracts / schemas
	reg("WSDL", CatConfig, XMLSyntax, ".wsdl")
	reg("XSD", CatConfig, XMLSyntax, ".xsd")
	reg("OpenAPI/JSON", CatConfig, None, ".json")
	reg("Protobuf", CatConfig, CStyle, ".proto")
	reg("GraphQL", CatConfig, Hash, ".graphql", ".gql")
	reg("Avro", CatConfig, None, ".avsc")
	// Config
	reg("XML", CatConfig, XMLSyntax, ".xml", ".xmi", ".tld", ".jrxml", ".xsl", ".xslt", ".bpmn", ".config")
	reg("Properties", CatConfig, Properties, ".properties")
	reg("YAML", CatConfig, Hash, ".yml", ".yaml")
	reg("TOML/INI", CatConfig, INI, ".toml", ".ini", ".cfg", ".conf")
	reg("Terraform", CatConfig, Hash, ".tf", ".tfvars", ".hcl")
	reg("Dockerfile", CatConfig, Hash, "dockerfile", "containerfile")
	reg("Jenkinsfile", CatBuild, CStyle, "jenkinsfile")
	reg("Makefile", CatBuild, Hash, "makefile", "gnumakefile")
	// Build descriptors (matched by name, take precedence over extension)
	reg("Maven POM", CatBuild, XMLSyntax, "pom.xml")
	reg("Ant", CatBuild, XMLSyntax, "build.xml")
	reg("Ivy", CatBuild, XMLSyntax, "ivy.xml")
	reg("Gradle", CatBuild, CStyle, ".gradle", "build.gradle.kts", "settings.gradle.kts")
	reg("npm", CatBuild, None, "package.json")
	reg("MSBuild", CatBuild, XMLSyntax, ".csproj", ".vbproj", ".fsproj", ".vcxproj", ".sln", "packages.config")
	reg("Python build", CatBuild, Hash, "requirements.txt", "pyproject.toml", "setup.cfg", "pipfile")
	reg("Composer", CatBuild, None, "composer.json")
	reg("Go modules", CatBuild, GoSyntax, "go.mod")
	reg("GitLab CI", CatBuild, Hash, ".gitlab-ci.yml")
	// Docs
	reg("Markdown", CatDocs, None, ".md", ".markdown")
	reg("Text", CatDocs, None, ".txt", ".adoc", ".rst")
}

// Detect returns the language of a path, or ok=false if it is unknown
// (binaries, images, archives... are inventoried by the caller as "Other").
func Detect(path string) (Language, bool) {
	base := strings.ToLower(filepath.Base(path))
	if l, ok := byName[base]; ok {
		return l, true
	}
	if strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt") {
		return byName["requirements.txt"], true
	}
	if strings.HasPrefix(base, "dockerfile") {
		return byName["dockerfile"], true
	}
	if l, ok := byExt[strings.ToLower(filepath.Ext(base))]; ok {
		return l, true
	}
	return Language{}, false
}

// Vendored archives worth inventorying even though they are not text.
func IsArchive(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jar", ".war", ".ear", ".rar", ".sar", ".har", ".dll", ".nupkg", ".whl", ".egg":
		return true
	}
	return false
}

// ByName returns a registered language by its display name.
func ByName(name string) (Language, bool) {
	for _, l := range byExt {
		if l.Name == name {
			return l, true
		}
	}
	for _, l := range byName {
		if l.Name == name {
			return l, true
		}
	}
	return Language{}, false
}

// Refine resolves extensions shared by several technologies (e.g. ".ctl" is
// a VB6 user control or a mainframe control card) by looking at the content.
func Refine(l Language, path, text string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	head := text
	if len(head) > 4096 {
		head = head[:4096]
	}
	switch ext {
	case ".ctl", ".cls", ".frm", ".bas":
		if strings.Contains(head, "VERSION ") || strings.Contains(head, "Attribute VB_") || strings.Contains(head, "Begin VB.") {
			return l
		}
		return byExt[".ctlcard"]
	case ".prc":
		for _, line := range strings.SplitN(head, "\n", 40) {
			if strings.HasPrefix(line, "//") {
				return byExt[".jcl"]
			}
		}
	}
	return l
}
