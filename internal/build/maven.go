package build

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/descriptors"
	"github.com/oskillarrr/traverse/internal/model"
)

type pomDep struct {
	g, a, v, scope string
	managed        bool
	line           int
}

// Pom is a parsed but not yet resolved pom.xml.
type Pom struct {
	Rel, Dir                 string
	G, A, V, Packaging, Name string
	PG, PA, PV, PRel         string
	Props                    map[string]string
	FinalName                string
	Modules                  []string
	Deps                     []pomDep
	parent                   *Pom
}

// ParsePom reads a pom.xml without resolving inheritance.
func ParsePom(rel, text string) *Pom {
	root, _ := descriptors.ParseXML(text)
	doc := root.Root()
	if doc == nil || doc.Name != "project" {
		return nil
	}
	dir := path.Dir(rel)
	if dir == "." {
		dir = ""
	}
	p := &Pom{Rel: rel, Dir: dir, G: doc.T("groupId"), A: doc.T("artifactId"), V: doc.T("version"),
		Packaging: doc.T("packaging"), Name: doc.T("name"), Props: map[string]string{}, FinalName: doc.T("build", "finalName")}
	if par := doc.Child("parent"); par != nil {
		p.PG, p.PA, p.PV, p.PRel = par.T("groupId"), par.T("artifactId"), par.T("version"), par.T("relativePath")
	}
	if pr := doc.Child("properties"); pr != nil {
		for _, c := range pr.Children {
			p.Props[c.Name] = strings.TrimSpace(c.Text)
		}
	}
	for _, m := range doc.Path("modules").ChildrenNamed("module") {
		p.Modules = append(p.Modules, strings.TrimSpace(m.Text))
	}
	for _, d := range doc.Path("dependencies").ChildrenNamed("dependency") {
		p.Deps = append(p.Deps, pomDep{d.T("groupId"), d.T("artifactId"), d.T("version"), d.T("scope"), false, d.Line})
	}
	for _, d := range doc.Path("dependencyManagement", "dependencies").ChildrenNamed("dependency") {
		p.Deps = append(p.Deps, pomDep{d.T("groupId"), d.T("artifactId"), d.T("version"), d.T("scope"), true, d.Line})
	}
	// compiler level: properties or maven-compiler-plugin configuration
	for _, pl := range doc.All("plugin") {
		if pl.T("artifactId") == "maven-compiler-plugin" {
			if s := pl.T("configuration", "source"); s != "" {
				p.Props["traverse.compiler"] = s
			} else if r := pl.T("configuration", "release"); r != "" {
				p.Props["traverse.compiler"] = r
			}
		}
	}
	return p
}

var reProp = regexp.MustCompile(`\$\{([^}]+)\}`)

func (p *Pom) prop(k string) (string, bool) {
	for x := p; x != nil; x = x.parent {
		switch k {
		case "project.version", "version", "pom.version":
			if x.V != "" {
				return x.V, true
			}
			if x.PV != "" {
				return x.PV, true
			}
		case "project.groupId", "groupId", "pom.groupId":
			if x.G != "" {
				return x.G, true
			}
			if x.PG != "" {
				return x.PG, true
			}
		case "project.artifactId":
			return x.A, true
		case "project.parent.version", "parent.version":
			return x.PV, x.PV != ""
		case "project.parent.groupId":
			return x.PG, x.PG != ""
		}
		if v, ok := x.Props[k]; ok {
			return v, true
		}
	}
	return "", false
}

func (p *Pom) resolve(s string) (string, bool) {
	for i := 0; i < 6 && strings.Contains(s, "${"); i++ {
		s = reProp.ReplaceAllStringFunc(s, func(m string) string {
			if v, ok := p.prop(m[2 : len(m)-1]); ok {
				return v
			}
			return m
		})
	}
	return s, !strings.Contains(s, "${")
}

func (p *Pom) managedVersion(g, a string) string {
	for x := p; x != nil; x = x.parent {
		for _, d := range x.Deps {
			if d.managed && d.g == g && d.a == a && d.v != "" {
				v, _ := x.resolve(d.v)
				return v
			}
		}
	}
	return ""
}

// ResolveMaven links parents within the repository and produces modules with
// inherited coordinates, resolved properties and managed versions.
func ResolveMaven(poms []*Pom) []model.Module {
	byGA := map[string]*Pom{}
	byDir := map[string]*Pom{}
	for _, p := range poms {
		byDir[p.Dir] = p
	}
	for _, p := range poms {
		g := p.G
		if g == "" {
			g = p.PG
		}
		byGA[g+":"+p.A] = p
	}
	for _, p := range poms {
		if p.PA == "" {
			continue
		}
		if par, ok := byGA[p.PG+":"+p.PA]; ok && par != p {
			p.parent = par
			continue
		}
		rel := p.PRel
		if rel == "" {
			rel = "../pom.xml"
		}
		if par, ok := byDir[path.Clean(path.Dir(path.Join(p.Dir, rel)))]; ok && par != p {
			p.parent = par
		}
	}
	var out []model.Module
	for _, p := range poms {
		g, _ := p.resolve(firstNonEmpty(p.G, p.PG))
		v, _ := p.resolve(firstNonEmpty(p.V, p.PV))
		pk := firstNonEmpty(p.Packaging, "jar")
		m := model.Module{ID: "maven:" + g + ":" + p.A, Build: "maven", Name: p.A, Version: v,
			Packaging: pk, Path: p.Dir, Descriptor: p.Rel}
		if pk == "war" || pk == "ear" {
			name := p.A + "-" + v
			if p.FinalName != "" {
				if fn, ok := p.resolve(p.FinalName); ok {
					name = fn
				}
			}
			m.Artifact = name + "." + pk
		}
		if p.PA != "" {
			m.Parent = "maven:" + p.PG + ":" + p.PA
		}
		for _, mod := range p.Modules {
			if c, ok := byDir[path.Clean(path.Join(p.Dir, mod))]; ok {
				cg := firstNonEmpty(c.G, c.PG)
				m.Children = append(m.Children, "maven:"+cg+":"+c.A)
			}
		}
		for _, k := range []string{"maven.compiler.source", "maven.compiler.release", "java.version", "traverse.compiler", "jdk.version"} {
			if s, ok := p.prop(k); ok && s != "" {
				if rs, ok := p.resolve(s); ok {
					m.Runtime = "java " + rs
					break
				}
			}
		}
		for _, d := range p.Deps {
			dg, _ := p.resolve(d.g)
			da, _ := p.resolve(d.a)
			ver := d.v
			if ver == "" && !d.managed {
				ver = p.managedVersion(d.g, d.a)
			}
			rv, ok := p.resolve(ver)
			m.Deps = append(m.Deps, model.Dependency{ID: "maven:" + dg + ":" + da, Version: rv, Scope: d.scope,
				Managed: d.managed, Kind: "declared", Resolved: ok && rv != "", Line: d.line, File: p.Rel})
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
