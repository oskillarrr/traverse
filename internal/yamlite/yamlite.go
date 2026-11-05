// Package yamlite is a small, dependency-free reader for the YAML subset used
// by Docker Compose, Kubernetes, OpenShift and Helm manifests: block mappings
// and sequences, plain/quoted scalars, block scalars (| >), single-line flow
// collections, comments and multiple documents. Anchors and tags are
// ignored; aliases are returned as plain scalars. It never executes anything
// and never fails hard: malformed input yields a partial tree and an error.
package yamlite

import (
	"fmt"
	"strings"
)

type Kind int

const (
	Scalar Kind = iota
	Map
	Seq
)

type Pair struct {
	Key   string
	Value *Node
}

type Node struct {
	Kind  Kind
	Value string // scalar value
	Pairs []Pair
	Items []*Node
	Line  int
}

// Entries returns the key/value pairs of a mapping (nil-safe).
func (n *Node) Entries() []Pair {
	if n == nil || n.Kind != Map {
		return nil
	}
	return n.Pairs
}

// Str returns the scalar value (nil-safe).
func (n *Node) Str() string {
	if n == nil || n.Kind != Scalar {
		return ""
	}
	return n.Value
}

// Get returns the value of a mapping key (nil if absent or not a map).
func (n *Node) Get(key string) *Node {
	if n == nil || n.Kind != Map {
		return nil
	}
	for _, p := range n.Pairs {
		if p.Key == key {
			return p.Value
		}
	}
	return nil
}

// Path walks nested mapping keys.
func (n *Node) Path(keys ...string) *Node {
	cur := n
	for _, k := range keys {
		cur = cur.Get(k)
	}
	return cur
}

// S returns the scalar at a path, or "".
func (n *Node) S(keys ...string) string {
	if c := n.Path(keys...); c != nil && c.Kind == Scalar {
		return c.Value
	}
	return ""
}

// List returns the items of a sequence (nil otherwise).
func (n *Node) List() []*Node {
	if n == nil || n.Kind != Seq {
		return nil
	}
	return n.Items
}

// Keys returns mapping keys in document order.
func (n *Node) Keys() []string {
	if n == nil || n.Kind != Map {
		return nil
	}
	ks := make([]string, len(n.Pairs))
	for i, p := range n.Pairs {
		ks[i] = p.Key
	}
	return ks
}

type line struct {
	indent int
	text   string
	no     int
}

type parser struct {
	lines []line
	pos   int
	err   error
}

// ParseAll parses every document of a YAML stream.
func ParseAll(src string) ([]*Node, error) {
	var docs []*Node
	var cur []line
	var firstErr error
	flush := func() {
		if len(cur) == 0 {
			return
		}
		p := &parser{lines: cur}
		n := p.block(cur[0].indent)
		if n != nil {
			docs = append(docs, n)
		}
		if p.err != nil && firstErr == nil {
			firstErr = p.err
		}
		cur = nil
	}
	raw := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i := 0; i < len(raw); i++ {
		l := strings.TrimRight(raw[i], " \t")
		if strings.HasPrefix(l, "---") && (len(l) == 3 || l[3] == ' ') || l == "..." {
			flush()
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), "%") && len(cur) == 0 { // directives
			continue
		}
		t := stripComment(l)
		if strings.TrimSpace(t) == "" {
			// keep blank lines inside block scalars only as empty text
			cur = append(cur, line{indent: -1, text: "", no: i + 1})
			continue
		}
		ind := len(t) - len(strings.TrimLeft(t, " "))
		cur = append(cur, line{indent: ind, text: strings.TrimSpace(t), no: i + 1})
	}
	flush()
	return docs, firstErr
}

func stripComment(s string) string {
	inS, inD := false, false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inD {
				inS = !inS
			}
		case '"':
			if !inS && (i == 0 || s[i-1] != '\\') {
				inD = !inD
			}
		case '#':
			if !inS && !inD && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
				return s[:i]
			}
		}
	}
	return s
}

func (p *parser) skipBlank() {
	for p.pos < len(p.lines) && p.lines[p.pos].indent < 0 {
		p.pos++
	}
}

func (p *parser) peek() *line {
	p.skipBlank()
	if p.pos < len(p.lines) {
		return &p.lines[p.pos]
	}
	return nil
}

// block parses the node starting at the current line with the given indent.
func (p *parser) block(indent int) *Node {
	l := p.peek()
	if l == nil || l.indent < indent {
		return nil
	}
	if isSeqItem(l.text) {
		return p.seq(l.indent)
	}
	if _, _, ok := splitKey(l.text); ok {
		return p.mapping(l.indent)
	}
	p.pos++
	return scalar(l.text, l.no)
}

func isSeqItem(t string) bool { return t == "-" || strings.HasPrefix(t, "- ") }

func (p *parser) seq(indent int) *Node {
	n := &Node{Kind: Seq, Line: p.lines[p.pos].no}
	for {
		l := p.peek()
		if l == nil || l.indent != indent || !isSeqItem(l.text) {
			return n
		}
		rest := strings.TrimSpace(strings.TrimPrefix(l.text, "-"))
		if rest == "" {
			p.pos++
			n.Items = append(n.Items, p.block(indent+1))
			continue
		}
		// "- key: value" starts a mapping whose indent is the content column
		col := indent + (len(l.text) - len(strings.TrimLeft(strings.TrimPrefix(l.text, "-"), " "))) + 0
		if _, _, ok := splitKey(rest); ok && !strings.HasPrefix(rest, "{") && !strings.HasPrefix(rest, "[") && !isQuotedScalar(rest) {
			p.lines[p.pos] = line{indent: col, text: rest, no: l.no}
			n.Items = append(n.Items, p.mapping(col))
			continue
		}
		if isSeqItem(rest) { // nested "- - x"
			p.lines[p.pos] = line{indent: col, text: rest, no: l.no}
			n.Items = append(n.Items, p.seq(col))
			continue
		}
		p.pos++
		n.Items = append(n.Items, p.value(rest, l.no, indent))
	}
}

func (p *parser) mapping(indent int) *Node {
	n := &Node{Kind: Map, Line: p.lines[p.pos].no}
	for {
		l := p.peek()
		if l == nil || l.indent != indent {
			if l != nil && l.indent > indent && p.err == nil {
				p.err = fmt.Errorf("line %d: unexpected indentation", l.no)
				p.pos++
				continue
			}
			return n
		}
		if isSeqItem(l.text) {
			return n // a sequence at the parent's level ends this mapping
		}
		k, v, ok := splitKey(l.text)
		if !ok {
			if p.err == nil {
				p.err = fmt.Errorf("line %d: expected 'key: value'", l.no)
			}
			p.pos++
			continue
		}
		p.pos++
		var val *Node
		switch {
		case v == "" || isAnchorOnly(v):
			next := p.peek()
			switch {
			case next != nil && next.indent > indent:
				val = p.block(next.indent)
			case next != nil && next.indent == indent && isSeqItem(next.text):
				val = p.seq(indent) // "key:\n- a" (sequence not indented)
			default:
				val = &Node{Kind: Scalar, Line: l.no}
			}
		default:
			val = p.value(v, l.no, indent)
		}
		n.Pairs = append(n.Pairs, Pair{Key: k, Value: val})
	}
}

// value parses an inline value: block scalar indicator, flow collection or scalar.
func (p *parser) value(v string, no, indent int) *Node {
	v = stripTagAnchor(v)
	switch {
	case strings.HasPrefix(v, "|") || strings.HasPrefix(v, ">"):
		var b strings.Builder
		for p.pos < len(p.lines) {
			l := p.lines[p.pos]
			if l.indent >= 0 && l.indent <= indent {
				break
			}
			b.WriteString(l.text)
			b.WriteByte('\n')
			p.pos++
		}
		return &Node{Kind: Scalar, Value: strings.TrimRight(b.String(), "\n"), Line: no}
	case strings.HasPrefix(v, "[") || strings.HasPrefix(v, "{"):
		// join continuation lines until brackets balance
		text := v
		for depth(text) > 0 && p.pos < len(p.lines) {
			text += " " + p.lines[p.pos].text
			p.pos++
		}
		n, _ := flow(text, no)
		return n
	}
	return scalar(v, no)
}

func isAnchorOnly(v string) bool {
	return (strings.HasPrefix(v, "&") || strings.HasPrefix(v, "!")) && !strings.Contains(v, " ")
}

func stripTagAnchor(v string) string {
	for strings.HasPrefix(v, "&") || strings.HasPrefix(v, "!") {
		i := strings.IndexByte(v, ' ')
		if i < 0 {
			return ""
		}
		v = strings.TrimSpace(v[i+1:])
	}
	return v
}

func depth(s string) int {
	d := 0
	inS, inD := false, false
	for _, c := range s {
		switch {
		case c == '\'' && !inD:
			inS = !inS
		case c == '"' && !inS:
			inD = !inD
		case inS || inD:
		case c == '[' || c == '{':
			d++
		case c == ']' || c == '}':
			d--
		}
	}
	return d
}

func isQuotedScalar(s string) bool {
	return (strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && !strings.Contains(s[1:len(s)-1], `":`)) ||
		(strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) && !strings.Contains(s[1:len(s)-1], `':`))
}

// splitKey splits "key: value" honouring quoted keys; value may be empty.
func splitKey(t string) (string, string, bool) {
	if strings.HasPrefix(t, "{") || strings.HasPrefix(t, "[") {
		return "", "", false
	}
	if strings.HasPrefix(t, `"`) || strings.HasPrefix(t, `'`) {
		q := t[0]
		end := strings.IndexByte(t[1:], q)
		if end < 0 {
			return "", "", false
		}
		rest := strings.TrimLeft(t[end+2:], " ")
		if !strings.HasPrefix(rest, ":") {
			return "", "", false
		}
		return t[1 : end+1], strings.TrimSpace(rest[1:]), true
	}
	for i := 0; i < len(t); i++ {
		if t[i] == ':' && (i+1 == len(t) || t[i+1] == ' ' || t[i+1] == '\t') {
			k := strings.TrimSpace(t[:i])
			if k == "" || strings.ContainsAny(k, "\"'") {
				return "", "", false
			}
			return k, strings.TrimSpace(t[i+1:]), true
		}
	}
	return "", "", false
}

func scalar(v string, no int) *Node {
	v = stripTagAnchor(strings.TrimSpace(v))
	if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
		inner := v[1 : len(v)-1]
		if v[0] == '\'' {
			inner = strings.ReplaceAll(inner, "''", "'")
		} else {
			inner = strings.NewReplacer(`\"`, `"`, `\\`, `\`, `\n`, "\n", `\t`, "\t").Replace(inner)
		}
		return &Node{Kind: Scalar, Value: inner, Line: no}
	}
	return &Node{Kind: Scalar, Value: v, Line: no}
}

// flow parses a single-line flow collection: [a, b] or {k: v, k2: [x]}.
func flow(s string, no int) (*Node, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return &Node{Kind: Scalar, Line: no}, ""
	}
	switch s[0] {
	case '[':
		n := &Node{Kind: Seq, Line: no}
		rest := strings.TrimSpace(s[1:])
		for rest != "" && rest[0] != ']' {
			var item *Node
			item, rest = flowItem(rest, no)
			n.Items = append(n.Items, item)
			rest = strings.TrimLeft(strings.TrimSpace(rest), ",")
			rest = strings.TrimSpace(rest)
		}
		if rest != "" {
			rest = rest[1:]
		}
		return n, rest
	case '{':
		n := &Node{Kind: Map, Line: no}
		rest := strings.TrimSpace(s[1:])
		for rest != "" && rest[0] != '}' {
			var key string
			key, rest = flowScalarText(rest, ":,}")
			rest = strings.TrimSpace(rest)
			var val *Node = &Node{Kind: Scalar, Line: no}
			if strings.HasPrefix(rest, ":") {
				val, rest = flowItem(strings.TrimSpace(rest[1:]), no)
			}
			n.Pairs = append(n.Pairs, Pair{Key: unquote(strings.TrimSpace(key)), Value: val})
			rest = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(rest), ","))
		}
		if rest != "" {
			rest = rest[1:]
		}
		return n, rest
	}
	return scalar(s, no), ""
}

func flowItem(s string, no int) (*Node, string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") || strings.HasPrefix(s, "{") {
		return flow(s, no)
	}
	v, rest := flowScalarText(s, ",]}")
	return scalar(v, no), rest
}

func flowScalarText(s, stops string) (string, string) {
	if strings.HasPrefix(s, `"`) || strings.HasPrefix(s, `'`) {
		q := s[0]
		for i := 1; i < len(s); i++ {
			if s[i] == q && (q == '\'' || s[i-1] != '\\') {
				return s[:i+1], s[i+1:]
			}
		}
		return s, ""
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(stops, s[i]) >= 0 && (s[i] != ':' || i+1 == len(s) || s[i+1] == ' ') {
			return strings.TrimSpace(s[:i]), s[i:]
		}
	}
	return strings.TrimSpace(s), ""
}

func unquote(s string) string {
	if n := scalar(s, 0); n != nil {
		return n.Value
	}
	return s
}

// HelmSafe neutralises Go template directives so Helm templates can be read
// as YAML: control lines are dropped and inline actions become TEMPLATE.
func HelmSafe(src string) string {
	var b strings.Builder
	for _, l := range strings.Split(src, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "{{") && strings.HasSuffix(t, "}}") && strings.Count(t, "{{") == 1 {
			b.WriteByte('\n') // control or include line: keep line numbers
			continue
		}
		for {
			i := strings.Index(l, "{{")
			if i < 0 {
				break
			}
			j := strings.Index(l[i:], "}}")
			if j < 0 {
				l = l[:i]
				break
			}
			l = l[:i] + "TEMPLATE" + l[i+j+2:]
		}
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String()
}
