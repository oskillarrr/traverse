package lang

import (
	"sort"
	"strings"
)

// Delim describes a string literal delimiter.
type Delim struct {
	Open, Close string
	Escape      bool // backslash escapes the next byte
	Multiline   bool // may span lines; otherwise an unterminated literal ends at EOL
}

// Syntax describes comments and string literals of a language family.
type Syntax struct {
	Line  []string
	Block [][2]string
	Str   []Delim
	// LineHook reports whole-line comments decided by column/prefix (COBOL, JCL, REM...).
	LineHook func(line string) bool
}

var (
	cStr = []Delim{
		{`"""`, `"""`, true, true},
		{"`", "`", true, true},
		{`"`, `"`, true, false},
		{`'`, `'`, true, false},
	}
	CStyle     = &Syntax{Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Str: cStr}
	CBlockOnly = &Syntax{Block: [][2]string{{"/*", "*/"}}, Str: []Delim{{`"`, `"`, true, false}, {`'`, `'`, true, false}}}
	GoSyntax   = &Syntax{Line: []string{"//"}, Block: [][2]string{{"/*", "*/"}}, Str: []Delim{{"`", "`", false, true}, {`"`, `"`, true, false}, {`'`, `'`, true, false}}}
	PHP        = &Syntax{Line: []string{"//", "#"}, Block: [][2]string{{"/*", "*/"}}, Str: []Delim{{`"`, `"`, true, true}, {`'`, `'`, true, true}}}
	Python     = &Syntax{Line: []string{"#"}, Str: []Delim{{`"""`, `"""`, true, true}, {`'''`, `'''`, true, true}, {`"`, `"`, true, false}, {`'`, `'`, true, false}}}
	Hash       = &Syntax{Line: []string{"#"}, Str: []Delim{{`"`, `"`, true, false}, {`'`, `'`, false, false}}}
	PowerShell = &Syntax{Line: []string{"#"}, Block: [][2]string{{"<#", "#>"}}, Str: []Delim{{`"`, `"`, false, false}, {`'`, `'`, false, false}}}
	SQLSyntax  = &Syntax{Line: []string{"--"}, Block: [][2]string{{"/*", "*/"}}, Str: []Delim{{`'`, `'`, false, true}}}
	XMLSyntax  = &Syntax{Block: [][2]string{{"<!--", "-->"}}}
	JSP        = &Syntax{Block: [][2]string{{"<%--", "--%>"}, {"<!--", "-->"}}}
	ASPX       = &Syntax{Block: [][2]string{{"<%--", "--%>"}, {"<!--", "-->"}}}
	VB         = &Syntax{Line: []string{"'"}, Str: []Delim{{`"`, `"`, false, false}}, LineHook: prefixFold("rem ")}
	Pascal     = &Syntax{Line: []string{"//"}, Block: [][2]string{{"{", "}"}, {"(*", "*)"}}, Str: []Delim{{`'`, `'`, false, false}}}
	Lisp       = &Syntax{Line: []string{";"}, Str: []Delim{{`"`, `"`, true, true}}}
	Batch      = &Syntax{LineHook: func(l string) bool {
		t := strings.ToLower(strings.TrimSpace(l))
		return strings.HasPrefix(t, "rem ") || t == "rem" || strings.HasPrefix(t, "::") || strings.HasPrefix(t, "@rem")
	}}
	Properties = &Syntax{LineHook: func(l string) bool {
		t := strings.TrimSpace(l)
		return strings.HasPrefix(t, "#") || strings.HasPrefix(t, "!")
	}}
	INI = &Syntax{LineHook: func(l string) bool {
		t := strings.TrimSpace(l)
		return strings.HasPrefix(t, ";") || strings.HasPrefix(t, "#")
	}}
	JCL   = &Syntax{LineHook: func(l string) bool { return strings.HasPrefix(l, "//*") }}
	COBOL = &Syntax{Line: []string{"*>"}, Str: []Delim{{`"`, `"`, false, false}, {`'`, `'`, false, false}}, LineHook: func(l string) bool {
		// Fixed format: indicator area in column 7.
		return len(l) >= 7 && (l[6] == '*' || l[6] == '/')
	}}
	Fortran = &Syntax{Line: []string{"!"}, Str: []Delim{{`'`, `'`, false, false}, {`"`, `"`, false, false}}, LineHook: func(l string) bool {
		return len(l) > 0 && (l[0] == 'C' || l[0] == 'c' || l[0] == '*')
	}}
	ABAP = &Syntax{Line: []string{`"`}, Str: []Delim{{`'`, `'`, false, false}, {"`", "`", false, false}}, LineHook: func(l string) bool { return strings.HasPrefix(l, "*") }}
	Asm  = &Syntax{LineHook: func(l string) bool { return strings.HasPrefix(l, "*") || strings.HasPrefix(l, ".*") }}
	None = &Syntax{}
)

func prefixFold(p string) func(string) bool {
	return func(l string) bool {
		t := strings.ToLower(strings.TrimSpace(l))
		return strings.HasPrefix(t, p) || t == strings.TrimSpace(p)
	}
}

// Stripped is the result of lexing a file. Both texts have exactly the same
// byte length as the source, so byte offsets and line numbers are preserved.
type Stripped struct {
	Code    string // comments blanked, string literals kept
	Bare    string // comments and string literal contents blanked
	CodeN   int
	CommN   int
	BlankN  int
	lineIdx []int
}

// Strip removes comments (and, in Bare, string contents) preserving offsets.
func Strip(src string, syn *Syntax) *Stripped {
	if syn == nil {
		syn = None
	}
	code := []byte(src)
	bare := []byte(src)
	n := len(src)
	s := &Stripped{}
	hasCode, hasComm := false, false
	endLine := func() {
		switch {
		case hasCode:
			s.CodeN++
		case hasComm:
			s.CommN++
		default:
			s.BlankN++
		}
		hasCode, hasComm = false, false
	}
	blank := func(i int, both bool) {
		if src[i] == '\n' {
			return
		}
		code[i] = ' '
		if both {
			bare[i] = ' '
		}
	}
	i := 0
	lineStart := true
	for i < n {
		c := src[i]
		if lineStart && syn.LineHook != nil {
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				end = n - i
			}
			line := strings.TrimRight(src[i:i+end], "\r")
			if strings.TrimSpace(line) != "" && syn.LineHook(line) {
				for k := i; k < i+end; k++ {
					blank(k, true)
				}
				hasComm = true
				i += end
				continue
			}
		}
		lineStart = false
		if c == '\n' {
			endLine()
			lineStart = true
			i++
			continue
		}
		// block comment
		if open, close, ok := matchBlock(src, i, syn.Block); ok {
			hasComm = true
			j := i + len(open)
			k := strings.Index(src[j:], close)
			stop := n
			if k >= 0 {
				stop = j + k + len(close)
			}
			for p := i; p < stop; p++ {
				if src[p] == '\n' {
					endLine()
					hasComm = true
					continue
				}
				blank(p, true)
			}
			i = stop
			continue
		}
		// line comment
		if matchAny(src, i, syn.Line) {
			hasComm = true
			for i < n && src[i] != '\n' {
				blank(i, true)
				i++
			}
			continue
		}
		// string literal
		if d, ok := matchStr(src, i, syn.Str); ok {
			hasCode = true
			j := i + len(d.Open)
			for j < n {
				if d.Escape && src[j] == '\\' && j+1 < n && src[j+1] != '\n' {
					bare[j], bare[j+1] = ' ', ' '
					j += 2
					continue
				}
				if strings.HasPrefix(src[j:], d.Close) {
					break
				}
				if src[j] == '\n' {
					if !d.Multiline {
						break // unterminated: recover at end of line
					}
					endLine()
					hasCode = true
					j++
					continue
				}
				bare[j] = ' '
				j++
			}
			if j < n && strings.HasPrefix(src[j:], d.Close) {
				j += len(d.Close)
			}
			i = j
			continue
		}
		if c != ' ' && c != '\t' && c != '\r' {
			hasCode = true
		}
		i++
	}
	if n > 0 && src[n-1] != '\n' {
		endLine()
	}
	s.Code, s.Bare = string(code), string(bare)
	s.lineIdx = lineIndex(src)
	return s
}

func matchBlock(src string, i int, blocks [][2]string) (string, string, bool) {
	for _, b := range blocks {
		if strings.HasPrefix(src[i:], b[0]) {
			return b[0], b[1], true
		}
	}
	return "", "", false
}

func matchAny(src string, i int, ps []string) bool {
	for _, p := range ps {
		if strings.HasPrefix(src[i:], p) {
			return true
		}
	}
	return false
}

func matchStr(src string, i int, ds []Delim) (Delim, bool) {
	for _, d := range ds {
		if strings.HasPrefix(src[i:], d.Open) {
			return d, true
		}
	}
	return Delim{}, false
}

func lineIndex(src string) []int {
	idx := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			idx = append(idx, i+1)
		}
	}
	return idx
}

// Line returns the 1-based line number of a byte offset.
func (s *Stripped) Line(offset int) int {
	return sort.Search(len(s.lineIdx), func(k int) bool { return s.lineIdx[k] > offset })
}
