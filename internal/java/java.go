// Package java is a structural (not semantic) extractor for Java sources.
// It works on comment-free text produced by the lang lexer, tracks brace
// nesting and attaches annotation blocks to the declaration that follows them.
// It tolerates any Java version from 1.2 to 21+ because it never needs to
// fully parse expressions.
package java

import (
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
)

type Annotation struct {
	Name string // simple name (last segment)
	Args string // raw argument text including string literals
	Line int
}

type Method struct {
	Name        string
	Annotations []Annotation
	Line        int
	Public      bool
}

func containsTok(ts []string, v string) bool {
	for _, t := range ts {
		if t == v {
			return true
		}
	}
	return false
}

// csAttributes parses a C# attribute section "[A, B(args)]" starting at toks[i]
// and returns them as annotations.
func csAttributes(toks []token, i int, s *lang.Stripped, matchParen func(int) int) ([]Annotation, int, bool) {
	at := func(k int) string {
		if k >= 0 && k < len(toks) {
			return toks[k].t
		}
		return ""
	}
	j := i + 1
	if isIdent(at(j)) && at(j+1) == ":" { // target: assembly:, return:, method:
		j += 2
	}
	var out []Annotation
	for {
		if !isIdent(at(j)) {
			return nil, i, false
		}
		name := at(j)
		for at(j+1) == "." && isIdent(at(j+2)) {
			j += 2
			name = at(j)
		}
		a := Annotation{Name: strings.TrimSuffix(name, "Attribute"), Line: s.Line(toks[j].off)}
		if at(j+1) == "(" {
			end := matchParen(j + 1)
			a.Args = s.Code[toks[j+1].off+1 : toks[end].off]
			j = end
		}
		out = append(out, a)
		switch at(j + 1) {
		case ",":
			j += 2
		case "]":
			return out, j + 1, true
		default:
			return nil, i, false
		}
	}
}

type Type struct {
	Name, FQN, Kind string
	Outer           string
	Line            int
	Annotations     []Annotation
	Extends         []string // as written (simple or qualified)
	Methods         []Method
	Fields          []Field
	TopLevel        bool
}

type Field struct {
	Type        string
	Annotations []Annotation
	Line        int
}

type Import struct {
	Name     string
	Static   bool
	Wildcard bool
	Line     int
}

type File struct {
	Package  string
	Imports  []Import
	Types    []*Type
	Refs     map[string]int // simple capitalized identifiers -> first line
	FQNRefs  map[string]int // qualified type references in code
	Literals []Literal      // string literals (content, line)
	Lookups  []Literal      // JNDI names passed to lookup("...")
}

type Literal struct {
	Value string
	Line  int
}

type token struct {
	t   string
	off int
}

var (
	rePackage = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)
	reImport  = regexp.MustCompile(`(?m)^\s*import\s+(static\s+)?([\w.$]+?)(\.\*)?\s*;`)
	reFQNRef  = regexp.MustCompile(`\b([a-z_][\w]*(?:\.[a-z_][\w]*)+)\.([A-Z][\w$]*)`)
	reStrLit  = regexp.MustCompile(`"((?:[^"\\\n]|\\.)*)"`)
	reLookup  = regexp.MustCompile(`\.lookup\s*\(\s*"([^"]+)"`)
)

var keywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "catch": true, "return": true, "new": true,
	"synchronized": true, "throw": true, "else": true, "do": true, "try": true, "assert": true, "super": true, "this": true,
}

var modifiers = map[string]bool{
	"public": true, "protected": true, "private": true, "static": true, "final": true, "abstract": true,
	"transient": true, "volatile": true, "native": true, "strictfp": true, "default": true, "sealed": true, "non": true,
	// C# / Kotlin
	"readonly": true, "internal": true, "const": true, "override": true, "virtual": true, "async": true, "partial": true,
	"required": true, "new": true, "extern": true, "unsafe": true, "event": true, "volatile_": true, "lateinit": true,
	"open": true, "inline": true, "suspend": true, "data": true,
}

// Dialect selects the surface syntax handled by the structural parser.
// C# and Kotlin share enough with Java (braces, types, annotations or
// attributes) for the same tokenizer to recover their structure.
type Dialect int

const (
	Java Dialect = iota
	CSharp
	Kotlin
)

var (
	reKtPackage = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)`)
	reKtImport  = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+?)(\.\*)?(?:\s+as\s+\w+)?\s*;?\s*$`)
	reCsNs      = regexp.MustCompile(`(?m)^\s*namespace\s+([\w.]+)`)
	reCsUsing   = regexp.MustCompile(`(?m)^\s*(?:global\s+)?using\s+(static\s+)?([\w.]+)\s*;`)
)

// Parse extracts the structure of one Java compilation unit.
func Parse(s *lang.Stripped) *File { return ParseDialect(s, Java) }

// ParseDialect extracts the structure of a Java, C# or Kotlin source file.
func ParseDialect(s *lang.Stripped, d Dialect) *File {
	f := &File{Refs: map[string]int{}, FQNRefs: map[string]int{}}
	switch d {
	case CSharp:
		if m := reCsNs.FindStringSubmatch(s.Bare); m != nil {
			f.Package = m[1]
		}
		for _, m := range reCsUsing.FindAllStringSubmatchIndex(s.Bare, -1) {
			// "using Namespace;" imports a whole namespace, like a Java wildcard
			f.Imports = append(f.Imports, Import{Name: s.Bare[m[4]:m[5]], Static: m[2] >= 0, Wildcard: m[2] < 0, Line: s.Line(m[0])})
		}
	case Kotlin:
		if m := reKtPackage.FindStringSubmatch(s.Bare); m != nil {
			f.Package = m[1]
		}
		for _, m := range reKtImport.FindAllStringSubmatchIndex(s.Bare, -1) {
			f.Imports = append(f.Imports, Import{Name: s.Bare[m[2]:m[3]], Wildcard: m[4] >= 0, Line: s.Line(m[0])})
		}
	default:
		if m := rePackage.FindStringSubmatch(s.Bare); m != nil {
			f.Package = m[1]
		}
		for _, m := range reImport.FindAllStringSubmatchIndex(s.Bare, -1) {
			f.Imports = append(f.Imports, Import{
				Name: s.Bare[m[4]:m[5]], Static: m[2] >= 0, Wildcard: m[6] >= 0, Line: s.Line(m[0]),
			})
		}
	}
	for _, m := range reFQNRef.FindAllStringSubmatchIndex(s.Bare, -1) {
		line := s.Line(m[0])
		if lineStartsWith(s.Bare, m[0], "import", "package", "using", "namespace") {
			continue
		}
		fqn := s.Bare[m[2]:m[3]] + "." + s.Bare[m[4]:m[5]]
		if _, ok := f.FQNRefs[fqn]; !ok {
			f.FQNRefs[fqn] = line
		}
	}
	for _, m := range reStrLit.FindAllStringSubmatchIndex(s.Code, -1) {
		f.Literals = append(f.Literals, Literal{Value: s.Code[m[2]:m[3]], Line: s.Line(m[0])})
	}
	for _, m := range reLookup.FindAllStringSubmatchIndex(s.Code, -1) {
		f.Lookups = append(f.Lookups, Literal{Value: s.Code[m[2]:m[3]], Line: s.Line(m[0])})
	}
	toks := tokenize(s.Bare)
	parseStructure(f, toks, s, d)
	return f
}

func lineStartsWith(text string, off int, words ...string) bool {
	start := strings.LastIndexByte(text[:off], '\n') + 1
	l := strings.TrimSpace(text[start:off])
	for _, w := range words {
		if strings.HasPrefix(l, w+" ") || l == w {
			return true
		}
	}
	return false
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func tokenize(src string) []token {
	toks := make([]token, 0, len(src)/4)
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '"' || c == '\'':
			i++
		case isIdentStart(c):
			j := i + 1
			for j < len(src) && isIdentPart(src[j]) {
				j++
			}
			toks = append(toks, token{src[i:j], i})
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && (isIdentPart(src[j]) || src[j] == '.') {
				j++
			}
			i = j
		default:
			toks = append(toks, token{string(c), i})
			i++
		}
	}
	return toks
}

func isIdent(t string) bool { return t != "" && isIdentStart(t[0]) }

type typeCtx struct {
	t         *Type
	bodyDepth int
}

func parseStructure(f *File, toks []token, s *lang.Stripped, d Dialect) {
	var stack []typeCtx
	var pending []Annotation
	var pendingType *Type
	var memberToks []string
	depth := 0
	sawAssign := false
	at := func(k int) string {
		if k >= 0 && k < len(toks) {
			return toks[k].t
		}
		return ""
	}
	memberLevel := func() bool { return len(stack) > 0 && depth == stack[len(stack)-1].bodyDepth }
	newline := func(j int) bool { // a line break separates toks[j-1] and toks[j]
		return j > 0 && j < len(toks) && strings.Contains(s.Bare[toks[j-1].off:toks[j].off], "\n")
	}
	matchParen := func(k int) int { // k at "(" -> index of matching ")"
		d := 0
		for j := k; j < len(toks); j++ {
			switch toks[j].t {
			case "(":
				d++
			case ")":
				d--
				if d == 0 {
					return j
				}
			}
		}
		return len(toks) - 1
	}

	for _, tk := range toks {
		if isIdent(tk.t) && tk.t[0] >= 'A' && tk.t[0] <= 'Z' {
			if _, ok := f.Refs[tk.t]; !ok {
				f.Refs[tk.t] = s.Line(tk.off)
			}
		}
	}
	for i := 0; i < len(toks); i++ {
		t := toks[i].t
		if d == Kotlin {
			switch {
			case t == "enum" && at(i+1) == "class":
				continue // "enum class X" -> handled by "class"
			case t == "object" && isIdent(at(i+1)) && at(i-1) != "companion":
				t = "class"
			case (t == "fun" || t == "val" || t == "var") && memberLevel():
				memberToks, sawAssign = nil, false // Kotlin members are newline-terminated
			}
		}
		if d == CSharp && t == "[" && (i == 0 || strings.Contains("{};]", at(i-1)) || newline(i)) {
			if anns, end, ok := csAttributes(toks, i, s, matchParen); ok {
				pending = append(pending, anns...)
				i = end
				continue
			}
		}
		switch {
		case t == "@" && at(i+1) == "interface" && isIdent(at(i+2)):
			pendingType = declare(f, &stack, at(i+2), "annotation", s.Line(toks[i].off), pending)
			pending = nil
			i += 2
		case t == "@" && isIdent(at(i+1)):
			name := at(i + 1)
			j := i + 1
			for at(j+1) == "." && isIdent(at(j+2)) {
				j += 2
				name = at(j)
			}
			a := Annotation{Name: name, Line: s.Line(toks[i].off)}
			if at(j+1) == "(" {
				end := matchParen(j + 1)
				a.Args = s.Code[toks[j+1].off+1 : toks[end].off]
				j = end
			}
			pending = append(pending, a)
			i = j
		case (t == "class" || t == "interface" || t == "enum" || t == "record" || (d == CSharp && t == "struct")) && at(i-1) != "." && isIdent(at(i+1)) && at(i-1) != "::":
			if t == "record" && d == Java && at(i+2) != "(" && at(i+2) != "<" {
				continue // contextual keyword used as identifier
			}
			kind := t
			typ := declare(f, &stack, at(i+1), kind, s.Line(toks[i].off), pending)
			pending = nil
			// header: generics, record components, extends/implements/permits until "{"
			j := i + 2
			angle, paren := 0, 0
			collecting, inWhere, bodiless := false, false, false
			for ; j < len(toks); j++ {
				x := toks[j].t
				if x == "{" && paren == 0 {
					break
				}
				if paren == 0 && angle == 0 && (x == ";" ||
					(d == Kotlin && newline(j) && x != ":" && x != "," && at(j-1) != ":" && at(j-1) != ",")) {
					bodiless = true // Kotlin data classes, C# positional records
					break
				}
				if d != Java && paren == 0 && angle == 0 {
					if x == "where" {
						collecting, inWhere = false, true
						continue
					}
					if x == ":" && !inWhere {
						collecting = true
						continue
					}
				}
				switch x {
				case "<":
					angle++
				case ">":
					angle--
				case "(":
					paren++
				case ")":
					paren--
				case "extends", "implements":
					collecting = angle == 0 && paren == 0
					continue
				case "permits", "throws":
					collecting = false
				}
				if collecting && angle == 0 && paren == 0 && isIdent(x) && at(j+1) != "." {
					// qualified names: rebuild a.b.C
					q := x
					for k := j - 1; k >= 1 && toks[k].t == "." && isIdent(toks[k-1].t); k -= 2 {
						q = toks[k-1].t + "." + q
					}
					typ.Extends = append(typ.Extends, q)
				}
			}
			if bodiless {
				i = j - 1
				continue
			}
			pendingType = typ
			i = j - 1
		case t == "{":
			depth++
			if pendingType != nil {
				stack = append(stack, typeCtx{pendingType, depth})
				pendingType = nil
			}
			pending, memberToks, sawAssign = nil, nil, false
		case t == "}":
			depth--
			for len(stack) > 0 && stack[len(stack)-1].bodyDepth > depth {
				stack = stack[:len(stack)-1]
			}
			pending, memberToks, sawAssign = nil, nil, false
		case t == ";":
			if memberLevel() && len(memberToks) > 0 {
				if ft := fieldType(memberToks); ft != "" {
					top := stack[len(stack)-1].t
					top.Fields = append(top.Fields, Field{Type: ft, Annotations: pending, Line: s.Line(toks[i].off)})
				}
			}
			pending, memberToks, sawAssign = nil, nil, false
		case t == "=":
			if memberLevel() {
				sawAssign = true
			}
		case isIdent(t) && at(i+1) == "(" && memberLevel() && !sawAssign && !keywords[t] && at(i-1) != ".":
			top := stack[len(stack)-1].t
			top.Methods = append(top.Methods, Method{Name: t, Annotations: pending, Line: s.Line(toks[i].off), Public: containsTok(memberToks, "public")})
			pending, memberToks = nil, nil
			i = matchParen(i + 1) // skip parameters (and their annotations)
		default:
			if memberLevel() && !sawAssign {
				memberToks = append(memberToks, t)
			}
		}
	}
}

func declare(f *File, stack *[]typeCtx, name, kind string, line int, anns []Annotation) *Type {
	t := &Type{Name: name, Kind: kind, Line: line, Annotations: anns}
	if len(*stack) > 0 {
		outer := (*stack)[len(*stack)-1].t
		t.Outer = outer.FQN
		t.FQN = outer.FQN + "." + name
	} else {
		t.TopLevel = true
		t.FQN = name
		if f.Package != "" {
			t.FQN = f.Package + "." + name
		}
	}
	f.Types = append(f.Types, t)
	return t
}

// fieldType picks the declared type from tokens like: private final List < Foo > foo
func fieldType(ts []string) string {
	for k, x := range ts {
		if modifiers[x] || x == "-" {
			continue
		}
		if !isIdent(x) {
			return ""
		}
		// qualified type a.b.C -> take last segment
		for k+2 < len(ts) && ts[k+1] == "." && isIdent(ts[k+2]) {
			k += 2
			x = ts[k]
		}
		return x
	}
	return ""
}

// Has reports whether an annotation with the given simple name is present.
func Has(anns []Annotation, names ...string) (Annotation, bool) {
	for _, a := range anns {
		for _, n := range names {
			if a.Name == n {
				return a, true
			}
		}
	}
	return Annotation{}, false
}

var reLit = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
var reArgName = regexp.MustCompile(`(?s)^\s*([A-Za-z_]\w*)\s*=([^=].*)$`)

// splitArgs splits annotation arguments at top-level commas, respecting
// string literals, arrays {...} and nested annotations (...).
func splitArgs(args string) []string {
	var parts []string
	depth, start := 0, 0
	inStr := false
	for i := 0; i < len(args); i++ {
		c := args[i]
		switch {
		case inStr:
			if c == '\\' {
				i++
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
		case c == '{' || c == '(':
			depth++
		case c == '}' || c == ')':
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, args[start:i])
			start = i + 1
		}
	}
	return append(parts, args[start:])
}

// Values returns string literal values of annotation arguments.
// key "" means the default (unnamed) value.
func (a Annotation) Values(keys ...string) []string {
	var out []string
	parts := splitArgs(a.Args)
	for _, k := range keys {
		for _, p := range parts {
			name, val := "", p
			if m := reArgName.FindStringSubmatch(p); m != nil {
				name, val = m[1], m[2]
			}
			if name != k {
				continue
			}
			for _, l := range reLit.FindAllStringSubmatch(val, -1) {
				out = append(out, l[1])
			}
		}
	}
	return out
}

// First returns the first value among keys, or "".
func (a Annotation) First(keys ...string) string {
	if v := a.Values(keys...); len(v) > 0 {
		return v[0]
	}
	return ""
}
