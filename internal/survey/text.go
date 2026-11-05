package survey

import (
	"fmt"
	"path"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

var labels = map[string]map[string]string{
	"es": {"main": "LÍNEA PRINCIPAL", "copy": "copia de %s", "identical": "idéntica a %s", "code": "código",
		"data": "datos (SQL)", "runtime": "runtime incluido (JDK/servidor)", "docs": "documentación", "models": "modelos", "binaries": "solo binarios",
		"other": "sin código", "empty": "vacía", "files": "ficheros", "more": "+%d más", "analyzed": "Se analiza", "skipped": "Se omite",
		"none": "No hay copias: se analiza todo.", "reason.trunk": "es trunk", "reason.highest-version": "versión más alta",
		"reason.highest-version+newest": "versión más alta y más reciente", "reason.newest": "la más reciente",
		"split": "Hay %d proyectos independientes; si son sistemas distintos, analízalos por separado."},
	"en": {"main": "MAIN LINE", "copy": "copy of %s", "identical": "identical to %s", "code": "code",
		"data": "data (SQL)", "runtime": "bundled runtime (JDK/server)", "docs": "documentation", "models": "models", "binaries": "binaries only",
		"other": "no code", "empty": "empty", "files": "files", "more": "+%d more", "analyzed": "Analyzed", "skipped": "Left out",
		"none": "No copies: everything is analyzed.", "reason.trunk": "it is trunk", "reason.highest-version": "highest version",
		"reason.highest-version+newest": "highest version, most recent", "reason.newest": "most recent",
		"split": "There are %d independent projects; if they are different systems, analyze them separately."},
}

// Label returns a localized label (es | en).
func Label(lang, key string) string {
	if m, ok := labels[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return labels["en"][key]
}

// Text renders the annotated directory tree.
func (r *Result) Text(lang string) string {
	var b strings.Builder
	children := map[string][]Node{}
	var root Node
	for _, n := range r.Tree {
		if n.Path == "" {
			root = n
			continue
		}
		p := path.Dir(n.Path)
		if p == "." {
			p = ""
		}
		children[p] = append(children[p], n)
	}
	name := path.Base(strings.ReplaceAll(r.Root, "\\", "/"))
	b.WriteString(name + "/" + annotate(root, lang) + "\n")
	var walk func(p, indent string)
	walk = func(p, indent string) {
		kids := children[p]
		for i, k := range kids {
			last := i == len(kids)-1
			branch, next := "├─ ", "│  "
			if last {
				branch, next = "└─ ", "   "
			}
			b.WriteString(indent + branch + path.Base(k.Path) + "/" + annotate(k, lang) + "\n")
			walk(k.Path, indent+next)
		}
	}
	walk("", "")
	return b.String()
}

func annotate(n Node, lang string) string {
	var parts []string
	switch n.Role {
	case "main":
		parts = append(parts, Label(lang, "main"))
	case "copy":
		if strings.HasPrefix(n.Note, "identical to ") {
			parts = append(parts, fmt.Sprintf(Label(lang, "identical"), strings.TrimPrefix(n.Note, "identical to ")))
		} else {
			parts = append(parts, fmt.Sprintf(Label(lang, "copy"), strings.TrimPrefix(n.Note, "copy of ")))
		}
	case "code", "data", "docs", "models", "binaries", "other", "runtime", "empty":
		parts = append(parts, Label(lang, n.Role))
	}
	if n.Coord != "" {
		c := n.Coord
		if n.Version != "" {
			c += " " + n.Version
		}
		parts = append(parts, c)
	}
	if n.Files > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", n.Files, Label(lang, "files")))
	}
	if n.Hidden > 0 && n.Role != "copy" && n.Role != "docs" && n.Role != "models" && n.Role != "data" {
		parts = append(parts, fmt.Sprintf(Label(lang, "more"), n.Hidden))
	}
	if len(parts) == 0 {
		return ""
	}
	return "   " + strings.Join(parts, " · ")
}

// Plan lists, in plain words, what will be analyzed and what is left out.
func (r *Result) Plan(lang string) string {
	var b strings.Builder
	s := r.Summary
	for _, x := range s.Roots {
		line := Label(lang, "analyzed") + ": " + disp(x.Path)
		if x.Coord != "" {
			line += " (" + strings.TrimSpace(x.Coord+" "+x.Version) + ")"
		}
		if x.Main {
			line += " — " + Label(lang, "main") + ", " + Label(lang, "reason."+x.Reason)
		}
		b.WriteString(line + "\n")
	}
	for _, c := range s.Copies {
		v := ""
		if c.Version != "" {
			v = " " + c.Version
		}
		rel := fmt.Sprintf(Label(lang, "copy"), disp(c.Of))
		if c.Identical {
			rel = fmt.Sprintf(Label(lang, "identical"), disp(c.Of))
		}
		verb := Label(lang, "skipped")
		if s.AllCopies {
			verb = Label(lang, "analyzed")
		}
		b.WriteString(fmt.Sprintf("%s: %s%s — %s\n", verb, disp(c.Path), v, rel))
	}
	if len(s.Copies) == 0 {
		b.WriteString(Label(lang, "none") + "\n")
	}
	// several main lines, each with its own copies: separately versioned projects
	mains := 0
	for _, x := range s.Roots {
		if x.Main {
			mains++
		}
	}
	if mains > 1 && hasDistinctTopLevels(s.Roots) {
		b.WriteString(fmt.Sprintf(Label(lang, "split"), mains) + "\n")
	}
	return b.String()
}

func disp(p string) string {
	if p == "" {
		return "."
	}
	return p
}

// hasDistinctTopLevels: the roots live under different top-level folders
// (independent projects side by side, not modules of one build).
func hasDistinctTopLevels(roots []model.SurveyRoot) bool {
	tops := map[string]bool{}
	for _, r := range roots {
		tops[strings.SplitN(r.Path, "/", 2)[0]] = true
	}
	return len(tops) > 1
}
