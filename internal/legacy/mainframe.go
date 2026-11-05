// Package legacy extracts architecture from mainframe sources: COBOL programs
// and copybooks, CICS commands and resource definitions (CSD), BMS screens
// and JCL jobs. It works on comment-free text from the lang lexer.
package legacy

import (
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/lang"
)

type Ref struct {
	Name   string
	Kind   string // call | link | xctl | copy | map | file | transid | exec | dataset | table
	Op     string // read | write | defines | uses
	Detail string
	Line   int
}

type Unit struct {
	Name     string // PROGRAM-ID, copybook name, job name or mapset
	Kind     string // program | copybook | job | proc | mapset | csd
	Line     int
	CICS     bool
	Screen   bool // sends or receives BMS maps
	Batch    bool // file SELECT/ASSIGN, no CICS
	Refs     []Ref
	Provides []Ref    // things the unit defines (maps, transactions, files)
	Dynamic  []string // variables holding program names resolved only at run time
}

var reMoveTo = `(?i)\bMOVE\s+['"]\s*([\w-]+)\s*['"]\s+TO\s+%s\b`

// resolveAll returns the literal program/resource names a COBOL operand can
// take: the literal itself, its VALUE clause, or literals MOVEd into it.
func resolveAll(code, quoted, name string) ([]string, bool) {
	if quoted != "" {
		return []string{strings.ToUpper(name)}, true
	}
	q := regexp.QuoteMeta(name)
	var out []string
	seen := map[string]bool{}
	for _, pat := range []string{reValueOf, reMoveTo} {
		re, err := regexp.Compile(strings.Replace(pat, "%s", q, 1))
		if err != nil {
			continue
		}
		for _, v := range re.FindAllStringSubmatch(code, -1) {
			if u := strings.ToUpper(v[1]); !seen[u] {
				seen[u] = true
				out = append(out, u)
			}
		}
	}
	return out, len(out) > 0
}

var (
	reProgID   = regexp.MustCompile(`(?i)\bPROGRAM-ID\.?\s+['"]?([\w-]+)`)
	reCall     = regexp.MustCompile(`(?i)\bCALL\s+['"]([\w-]+)['"]`)
	reCallVar  = regexp.MustCompile(`(?i)\bCALL\s+([A-Z][\w-]*)`)
	reCicsXfer = regexp.MustCompile(`(?is)EXEC\s+CICS\s+(LINK|XCTL)\b[^.]*?PROGRAM\s*\(\s*(['"]?)([\w-]+)['"]?\s*\)`)
	reCopy     = regexp.MustCompile(`(?im)^\s*\d*\s*COPY\s+['"]?([\w-]+)`)
	reCicsMap  = regexp.MustCompile(`(?is)EXEC\s+CICS\s+(SEND|RECEIVE)\s+MAP\s*\(\s*(['"]?)([\w-]+)['"]?\s*\)(?:[^.]*?MAPSET\s*\(\s*(['"]?)([\w-]+)['"]?\s*\))?`)
	reCicsFile = regexp.MustCompile(`(?is)EXEC\s+CICS\s+(READ|WRITE|REWRITE|DELETE|STARTBR|READNEXT|READPREV|ENDBR|UNLOCK)\b[^.]*?\b(?:DATASET|FILE)\s*\(\s*(['"]?)([\w-]+)['"]?\s*\)`)
	reCicsTran = regexp.MustCompile(`(?is)EXEC\s+CICS\s+(RETURN|START)\b[^.]*?TRANSID\s*\(\s*(['"]?)([\w-]+)['"]?\s*\)`)
	reCicsQ    = regexp.MustCompile(`(?is)EXEC\s+CICS\s+(WRITEQ|READQ)\s+(TS|TD)\b[^.]*?QUEUE\s*\(\s*(['"]?)([\w-]+)['"]?\s*\)`)
	reSelect   = regexp.MustCompile(`(?i)\bSELECT\s+(?:OPTIONAL\s+)?([\w-]+)\s+ASSIGN\s+(?:TO\s+)?['"]?([\w-]+)`)
	reValueOf  = `(?i)\b%s\s+PIC[^.]*?VALUE\s+['"]\s*([\w-]+)\s*['"]`

	reJCLJob    = regexp.MustCompile(`(?m)^//([\w#@$]+)\s+JOB\b`)
	reJCLProc   = regexp.MustCompile(`(?m)^//([\w#@$]+)\s+PROC\b`)
	reJCLExec   = regexp.MustCompile(`(?m)^//([\w#@$]*)\s+EXEC\s+(?:PGM=([\w#@$]+)|PROC=([\w#@$]+)|([\w#@$]+))`)
	reJCLDSN    = regexp.MustCompile(`(?i)\bDSN(?:AME)?=([\w.#@$&()+-]+)`)
	reJCLDisp   = regexp.MustCompile(`(?i)\bDISP=\(?\s*(NEW|OLD|SHR|MOD)`)
	reJCLRun    = regexp.MustCompile(`(?i)\bRUN\s+PROG(?:RAM)?\s*\(\s*([\w-]+)\s*\)`)
	reBMSMapset = regexp.MustCompile(`(?m)^([\w#@$]+)\s+DFHMSD\b`)
	reBMSMap    = regexp.MustCompile(`(?m)^([\w#@$]+)\s+DFHMDI\b`)

	reCSDTran = regexp.MustCompile(`(?is)DEFINE\s+TRANSACTION\s*\(\s*([\w#@$]+)\s*\)(.{0,600}?)(?:DEFINE|\z)`)
	reCSDFile = regexp.MustCompile(`(?is)DEFINE\s+FILE\s*\(\s*([\w#@$]+)\s*\)(.{0,800}?)(?:DEFINE|\z)`)
	reCSDProg = regexp.MustCompile(`(?i)\bPROGRAM\s*\(\s*([\w#@$]+)\s*\)`)
	reCSDDsn  = regexp.MustCompile(`(?i)\bDSNAME\s*\(\s*([\w.#@$]+)\s*\)`)
)

// Utility programs that are not part of the application.
var utilities = map[string]bool{
	"IEFBR14": true, "IDCAMS": true, "IEBGENER": true, "SORT": true, "DFSORT": true, "ICETOOL": true, "ICEMAN": true,
	"IKJEFT01": true, "IKJEFT1A": true, "IKJEFT1B": true, "DSNUTILB": true, "IEBCOPY": true, "ADRDSSU": true, "IEHPROGM": true,
	"DSNTIAUL": true, "DSNTEP2": true, "DSNTEP4": true, "IEBUPDTE": true, "IEBPTPCH": true, "SYNCSORT": true, "FILEAID": true,
	"CEEDUMP": true, "CEE3ABD": true, "CBLTDLI": true, "DFSRRC00": true, "SDSF": true, "FTP": true, "BPXBATCH": true,
}

// IsUtility reports whether a program name is a system utility or a
// runtime service (Language Environment CEE*, IBM MQ API MQ*, DB2 DSN*).
func IsUtility(p string) bool {
	p = strings.ToUpper(p)
	if utilities[p] {
		return true
	}
	for _, pre := range []string{"CEE", "MQ", "DSN", "DFH", "IGZ", "ILBO", "CBL"} {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// Assembler returns a program unit for an HLASM source (named by its file).
func Assembler(fileBase string) *Unit {
	return &Unit{Kind: "program", Name: strings.ToUpper(fileBase), Line: 1}
}

// fixedFormat blanks the sequence area (columns 1-6) and the identification
// area (columns 73-80) of fixed-format sources, preserving offsets.
func fixedFormat(code string, seq bool) string {
	b := []byte(code)
	start := 0
	for i := 0; i <= len(b); i++ {
		if i == len(b) || b[i] == '\n' {
			end := i
			if end > start && b[end-1] == '\r' {
				end--
			}
			if seq {
				for k := start; k < start+6 && k < end; k++ {
					if b[k] >= '0' && b[k] <= '9' {
						b[k] = ' '
					}
				}
			}
			for k := start + 72; k < end; k++ {
				b[k] = ' '
			}
			start = i + 1
		}
	}
	return string(b)
}

// COBOL parses a program or copybook.
func COBOL(st *lang.Stripped, fileBase string, copybook bool) *Unit {
	code := fixedFormat(st.Code, true)
	u := &Unit{Kind: "program"}
	if m := reProgID.FindStringSubmatchIndex(code); m != nil {
		u.Name, u.Line = strings.ToUpper(code[m[2]:m[3]]), st.Line(m[0])
	} else {
		u.Name, u.Line = strings.ToUpper(fileBase), 1
		if copybook || !strings.Contains(strings.ToUpper(code), "PROCEDURE DIVISION") {
			u.Kind = "copybook"
		}
	}
	resolve := func(quoted, name string) (string, bool) {
		if vs, ok := resolveAll(code, quoted, name); ok {
			return vs[0], true
		}
		return strings.ToUpper(name), false
	}
	add := func(r Ref, off int) {
		r.Line = st.Line(off)
		u.Refs = append(u.Refs, r)
	}
	for _, m := range reCall.FindAllStringSubmatchIndex(code, -1) {
		add(Ref{Name: strings.ToUpper(code[m[2]:m[3]]), Kind: "call"}, m[0])
	}
	for _, m := range reCallVar.FindAllStringSubmatchIndex(code, -1) {
		name := code[m[2]:m[3]]
		if strings.EqualFold(name, "USING") {
			continue
		}
		if vs, ok := resolveAll(code, "", name); ok {
			for _, v := range vs {
				add(Ref{Name: v, Kind: "call", Detail: "via variable " + strings.ToUpper(name)}, m[0])
			}
		} else {
			u.Dynamic = append(u.Dynamic, strings.ToUpper(name))
		}
	}
	for _, m := range reCicsXfer.FindAllStringSubmatchIndex(code, -1) {
		u.CICS = true
		kind := strings.ToLower(code[m[2]:m[3]])
		vs, ok := resolveAll(code, code[m[4]:m[5]], code[m[6]:m[7]])
		if !ok {
			u.Dynamic = append(u.Dynamic, strings.ToUpper(code[m[6]:m[7]]))
			continue // program chosen at run time from data (e.g. a menu table)
		}
		for _, v := range vs {
			detail := ""
			if code[m[4]:m[5]] == "" {
				detail = "via variable " + strings.ToUpper(code[m[6]:m[7]])
			}
			add(Ref{Name: v, Kind: kind, Detail: detail}, m[0])
		}
	}
	for _, m := range reCopy.FindAllStringSubmatchIndex(code, -1) {
		add(Ref{Name: strings.ToUpper(code[m[2]:m[3]]), Kind: "copy"}, m[0])
	}
	for _, m := range reCicsMap.FindAllStringSubmatchIndex(code, -1) {
		u.CICS, u.Screen = true, true
		mp, _ := resolve(code[m[4]:m[5]], code[m[6]:m[7]])
		set := ""
		if m[10] >= 0 {
			set, _ = resolve(code[m[8]:m[9]], code[m[10]:m[11]])
		}
		op := "write"
		if strings.EqualFold(code[m[2]:m[3]], "RECEIVE") {
			op = "read"
		}
		add(Ref{Name: firstNonEmpty(set, mp), Kind: "map", Op: op, Detail: "map " + mp}, m[0])
	}
	for _, m := range reCicsFile.FindAllStringSubmatchIndex(code, -1) {
		u.CICS = true
		name, _ := resolve(code[m[4]:m[5]], code[m[6]:m[7]])
		op := "read"
		switch strings.ToUpper(code[m[2]:m[3]]) {
		case "WRITE", "REWRITE", "DELETE":
			op = "write"
		}
		add(Ref{Name: name, Kind: "file", Op: op, Detail: "CICS " + strings.ToUpper(code[m[2]:m[3]])}, m[0])
	}
	for _, m := range reCicsTran.FindAllStringSubmatchIndex(code, -1) {
		u.CICS = true
		name, _ := resolve(code[m[4]:m[5]], code[m[6]:m[7]])
		add(Ref{Name: name, Kind: "transid", Detail: "CICS " + strings.ToUpper(code[m[2]:m[3]])}, m[0])
	}
	for _, m := range reCicsQ.FindAllStringSubmatchIndex(code, -1) {
		u.CICS = true
		name, _ := resolve(code[m[6]:m[7]], code[m[8]:m[9]])
		op := "read"
		if strings.HasPrefix(strings.ToUpper(code[m[2]:m[3]]), "WRITE") {
			op = "write"
		}
		add(Ref{Name: name, Kind: "queue-" + strings.ToLower(code[m[4]:m[5]]), Op: op}, m[0])
	}
	for _, m := range reSelect.FindAllStringSubmatchIndex(code, -1) {
		u.Batch = true
		add(Ref{Name: strings.ToUpper(code[m[4]:m[5]]), Kind: "ddname", Detail: "file " + strings.ToUpper(code[m[2]:m[3]])}, m[0])
	}
	if strings.Contains(strings.ToUpper(code), "EXEC CICS") {
		u.CICS = true
	}
	if u.CICS {
		u.Batch = false
	}
	return u
}

// JCL parses a job (or catalogued procedure).
func JCL(st *lang.Stripped, fileBase string) *Unit {
	code := fixedFormat(st.Code, false)
	u := &Unit{Kind: "job", Name: strings.ToUpper(fileBase), Line: 1}
	if m := reJCLJob.FindStringSubmatchIndex(code); m != nil {
		u.Name, u.Line = strings.ToUpper(code[m[2]:m[3]]), st.Line(m[0])
	} else if m := reJCLProc.FindStringSubmatchIndex(code); m != nil {
		u.Name, u.Line, u.Kind = strings.ToUpper(code[m[2]:m[3]]), st.Line(m[0]), "proc"
	}
	for _, m := range reJCLExec.FindAllStringSubmatchIndex(code, -1) {
		step := ""
		if m[2] >= 0 {
			step = code[m[2]:m[3]]
		}
		switch {
		case m[4] >= 0:
			u.Refs = append(u.Refs, Ref{Name: strings.ToUpper(code[m[4]:m[5]]), Kind: "exec", Detail: "step " + step, Line: st.Line(m[0])})
		case m[6] >= 0:
			u.Refs = append(u.Refs, Ref{Name: strings.ToUpper(code[m[6]:m[7]]), Kind: "proc", Detail: "step " + step, Line: st.Line(m[0])})
		case m[8] >= 0:
			u.Refs = append(u.Refs, Ref{Name: strings.ToUpper(code[m[8]:m[9]]), Kind: "proc", Detail: "step " + step, Line: st.Line(m[0])})
		}
	}
	for _, m := range reJCLRun.FindAllStringSubmatchIndex(code, -1) {
		u.Refs = append(u.Refs, Ref{Name: strings.ToUpper(code[m[2]:m[3]]), Kind: "exec", Detail: "DB2 RUN PROGRAM", Line: st.Line(m[0])})
	}
	lines := strings.Split(code, "\n")
	for i, l := range lines {
		if !strings.HasPrefix(l, "//") {
			continue
		}
		dm := reJCLDSN.FindStringSubmatch(l)
		if dm == nil {
			continue
		}
		dsn := strings.ToUpper(strings.TrimRight(dm[1], ","))
		if strings.HasPrefix(dsn, "&&") || dsn == "NULLFILE" || strings.HasPrefix(dsn, "*.") {
			continue // temporary datasets and backward references
		}
		if p := strings.IndexByte(dsn, '('); p > 0 { // GDG / member
			dsn = dsn[:p]
		}
		op := "uses"
		// DISP may be on this or the next continuation lines
		for k := i; k < len(lines) && k < i+4; k++ {
			if d := reJCLDisp.FindStringSubmatch(lines[k]); d != nil {
				switch strings.ToUpper(d[1]) {
				case "NEW":
					op = "defines"
				case "MOD", "OLD":
					op = "write"
				case "SHR":
					op = "read"
				}
				break
			}
			if k > i && !strings.HasPrefix(lines[k], "//  ") && !strings.HasPrefix(lines[k], "//         ") {
				break
			}
		}
		u.Refs = append(u.Refs, Ref{Name: dsn, Kind: "dataset", Op: op, Line: i + 1})
	}
	return u
}

// BMS parses a BMS mapset.
func BMS(st *lang.Stripped, fileBase string) *Unit {
	u := &Unit{Kind: "mapset", Name: strings.ToUpper(fileBase), Line: 1, Screen: true}
	if m := reBMSMapset.FindStringSubmatchIndex(st.Code); m != nil {
		u.Name, u.Line = strings.ToUpper(st.Code[m[2]:m[3]]), st.Line(m[0])
	}
	for _, m := range reBMSMap.FindAllStringSubmatchIndex(st.Code, -1) {
		u.Provides = append(u.Provides, Ref{Name: strings.ToUpper(st.Code[m[2]:m[3]]), Kind: "map", Line: st.Line(m[0])})
	}
	return u
}

// HasCSD reports whether a text contains CICS resource definitions.
func HasCSD(text string) bool {
	return strings.Contains(strings.ToUpper(text), "DEFINE TRANSACTION(") || strings.Contains(strings.ToUpper(text), "DEFINE TRANSACTION (")
}

// CSD parses CICS resource definitions: transactions and files.
func CSD(st *lang.Stripped) *Unit {
	u := &Unit{Kind: "csd"}
	for _, m := range reCSDTran.FindAllStringSubmatchIndex(st.Code, -1) {
		body := st.Code[m[4]:m[5]]
		prog := ""
		if p := reCSDProg.FindStringSubmatch(body); p != nil {
			prog = strings.ToUpper(p[1])
		}
		u.Provides = append(u.Provides, Ref{Name: strings.ToUpper(st.Code[m[2]:m[3]]), Kind: "transid", Detail: prog, Line: st.Line(m[0])})
	}
	for _, m := range reCSDFile.FindAllStringSubmatchIndex(st.Code, -1) {
		body := st.Code[m[4]:m[5]]
		dsn := ""
		if p := reCSDDsn.FindStringSubmatch(body); p != nil {
			dsn = strings.ToUpper(p[1])
		}
		u.Provides = append(u.Provides, Ref{Name: strings.ToUpper(st.Code[m[2]:m[3]]), Kind: "file", Detail: dsn, Line: st.Line(m[0])})
	}
	return u
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
