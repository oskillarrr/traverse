// Package descriptors parses deployment descriptors and build manifests.
// XML is parsed with encoding/xml, which never resolves external entities or
// DTDs, so parsing is offline and XXE-safe by construction.
package descriptors

import (
	"encoding/xml"
	"io"
	"strings"
)

type Node struct {
	Name     string // local name
	Attr     map[string]string
	Children []*Node
	Text     string
	Line     int
	Parent   *Node
}

// ParseXML builds a namespace-agnostic tree. It is lenient (non-strict) so
// that hand-edited legacy descriptors still yield partial results.
func ParseXML(text string) (*Node, error) {
	d := xml.NewDecoder(strings.NewReader(text))
	d.Strict = false
	d.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil } // already UTF-8
	root := &Node{Name: "#document"}
	cur := root
	var sb strings.Builder
	for {
		line, _ := d.InputPos()
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return root, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &Node{Name: t.Name.Local, Attr: map[string]string{}, Line: line, Parent: cur}
			for _, a := range t.Attr {
				n.Attr[a.Name.Local] = a.Value
			}
			cur.Children = append(cur.Children, n)
			cur = n
			sb.Reset()
		case xml.CharData:
			sb.Write(t)
			cur.Text += string(t)
		case xml.EndElement:
			cur.Text = strings.TrimSpace(cur.Text)
			if cur.Parent != nil {
				cur = cur.Parent
			}
		}
	}
	return root, nil
}

// Root returns the first element of the document.
func (n *Node) Root() *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		return c
	}
	return nil
}

// Child returns the first direct child with the local name.
func (n *Node) Child(name string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// Children named.
func (n *Node) ChildrenNamed(name string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, c := range n.Children {
		if c.Name == name {
			out = append(out, c)
		}
	}
	return out
}

// Path walks direct children by names.
func (n *Node) Path(names ...string) *Node {
	cur := n
	for _, nm := range names {
		cur = cur.Child(nm)
		if cur == nil {
			return nil
		}
	}
	return cur
}

// T returns trimmed text of a child path, or "".
func (n *Node) T(names ...string) string {
	if c := n.Path(names...); c != nil {
		return strings.TrimSpace(c.Text)
	}
	return ""
}

// InnerText returns this node's text and all descendant text, in document
// order (approximately: text that sits between child elements is merged onto
// the parent). It is enough for scanning statement bodies that embed dynamic
// tags, where each clause keeps its keyword and operand together.
func (n *Node) InnerText() string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var rec func(*Node)
	rec = func(x *Node) {
		if x.Text != "" {
			sb.WriteString(x.Text)
			sb.WriteByte(' ')
		}
		for _, c := range x.Children {
			rec(c)
		}
	}
	rec(n)
	return sb.String()
}

// All returns all descendants with the local name (depth-first).
func (n *Node) All(name string) []*Node {
	var out []*Node
	var rec func(*Node)
	rec = func(x *Node) {
		for _, c := range x.Children {
			if c.Name == name {
				out = append(out, c)
			}
			rec(c)
		}
	}
	if n != nil {
		rec(n)
	}
	return out
}
