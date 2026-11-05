package scan

import (
	"path"
	"strings"

	"github.com/oskillarrr/traverse/internal/legacy"
	"github.com/oskillarrr/traverse/internal/model"
)

// otherSemantics builds components, dependencies and contracts for the
// non-JVM extractors: script/web languages (module level), mainframe
// (programs, jobs, screens), ASP.NET minimal APIs and server pages.
func otherSemantics(f *model.Facts, results []*fileResult, o Options, moduleOf func(string) string, getDep func(string) *model.Deployable) {
	cls := NewClassifier(o.Config)
	include := func(r *fileResult) bool {
		return r != nil && (r.file.SourceSet == "main" || o.IncludeTests)
	}
	existing := map[string]bool{}
	for _, c := range f.Components {
		existing[c.FQN] = true
	}
	addComp := func(c model.Component) bool {
		if existing[c.FQN] {
			return false
		}
		existing[c.FQN] = true
		f.Components = append(f.Components, c)
		return true
	}
	expose := func(mod, verb, p, comp, detail, conf, file string, line int) {
		d := getDep(mod)
		key := verb + " " + model.JoinPath(d.ContextRoot, p)
		addContract(f, "http", "exposes", key, model.NormalizePath(p), comp, detail, conf, file, line)
	}
	goModules := map[string]string{} // module path -> directory
	for _, m := range f.Modules {
		if m.Build == "go" {
			goModules[m.Name] = m.Path
		}
	}

	// ---------- script / web languages ----------
	fileFQN := map[string]string{} // relative path without extension -> FQN
	phpFQN := map[string]string{}  // PHP Namespace\Class -> FQN
	pyFQN := map[string][]string{} // dotted module suffix -> FQNs
	goPkgFQN := map[string]string{}
	compOfFile := map[string]string{}
	for _, r := range results {
		if !include(r) || r.lite == nil || r.file.Category != "code" {
			continue
		}
		rel := r.file.Path
		dir, base := path.Dir(rel), path.Base(rel)
		stem := strings.TrimSuffix(base, path.Ext(base))
		if dir == "." {
			dir = ""
		}
		pkg := strings.ReplaceAll(dir, "/", ".")
		name, kind := stem, "module"
		if stem == "__init__" { // a Python package is represented by its directory
			pkg, name, kind = strings.ReplaceAll(path.Dir(dir), "/", "."), path.Base(dir), "package"
			if pkg == "." {
				pkg = ""
			}
		}
		if r.file.Language == "Go" {
			name, kind = path.Base(dir), "package"
			if dir == "" {
				name = "main"
			}
		}
		if r.file.Language == "PHP" && r.lite.Namespace != "" && len(r.lite.Classes) > 0 {
			pkg, name, kind = strings.ReplaceAll(r.lite.Namespace, `\`, "."), r.lite.Classes[0], "class"
		}
		fqn := name
		if pkg != "" {
			fqn = pkg + "." + name
		}
		if r.file.Language == "Go" {
			fqn = pkg
			if fqn == "" {
				fqn = "main"
			}
		}
		// classify with file-name tokens too (user.controller.ts, user_service.py)
		tokens := pkg + "." + strings.NewReplacer("-", ".", "_", ".").Replace(stem)
		layer, reason := cls.Classify(strings.Trim(tokens, "."), name, nil, nil)
		addComp(model.Component{FQN: fqn, Name: name, Kind: kind, Language: r.file.Language, Package: pkg, SourceSet: r.file.SourceSet,
			Module: moduleOf(rel), Layer: layer, LayerReason: reason, File: rel, Line: 1})
		compOfFile[rel] = fqn
		fileFQN[strings.TrimSuffix(rel, path.Ext(rel))] = fqn
		if stem == "index" || stem == "__init__" {
			fileFQN[dir] = fqn
		}
		switch r.file.Language {
		case "PHP":
			for _, c := range r.lite.Classes {
				if r.lite.Namespace != "" {
					phpFQN[r.lite.Namespace+`\`+c] = fqn
				}
			}
		case "Python":
			mod := strings.ReplaceAll(strings.TrimSuffix(rel, ".py"), "/", ".")
			mod = strings.TrimSuffix(mod, ".__init__")
			parts := strings.Split(mod, ".")
			for i := range parts {
				k := strings.Join(parts[i:], ".")
				pyFQN[k] = append(pyFQN[k], fqn)
			}
		case "Go":
			goPkgFQN[dir] = fqn
		}
	}
	jsExts := []string{"", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".vue", "/index.ts", "/index.tsx", "/index.js", "/index.vue"}
	resolveFile := func(p string) string {
		p = path.Clean(p)
		for _, e := range jsExts {
			q := p + e
			if fq, ok := fileFQN[strings.TrimSuffix(q, path.Ext(q))]; ok && (e == "" || strings.HasSuffix(q, e)) {
				return fq
			}
		}
		return fileFQN[p]
	}
	for _, r := range results {
		if !include(r) || r.lite == nil {
			continue
		}
		from := compOfFile[r.file.Path]
		if from == "" {
			continue
		}
		mod := moduleOf(r.file.Path)
		dir := path.Dir(r.file.Path)
		seen := map[string]bool{}
		resolvedOptional := map[int]bool{} // line -> a submodule import resolved
		for _, im := range r.lite.Imports {
			if !im.Optional && resolvedOptional[im.Line] {
				continue
			}
			to, kind := "", "import"
			switch r.file.Language {
			case "JavaScript", "TypeScript", "Vue", "Ruby":
				switch {
				case strings.HasPrefix(im.Spec, "."):
					to = resolveFile(path.Join(dir, im.Spec))
				case strings.HasPrefix(im.Spec, "@/") || strings.HasPrefix(im.Spec, "~/"):
					to = resolveFile(path.Join("src", im.Spec[2:]))
					if to == "" {
						to = resolveFile(im.Spec[2:])
					}
				}
			case "Python":
				if im.Rel > 0 {
					base := dir
					for i := 1; i < im.Rel; i++ {
						base = path.Dir(base)
					}
					to = resolveFile(path.Join(base, strings.ReplaceAll(im.Spec, ".", "/")))
				} else if c := pyFQN[im.Spec]; len(c) == 1 {
					to = c[0]
				}
			case "PHP":
				to = phpFQN[im.Spec]
			case "Go":
				for mp, mdir := range goModules {
					if im.Spec == mp || strings.HasPrefix(im.Spec, mp+"/") {
						d := strings.TrimPrefix(strings.TrimPrefix(im.Spec, mp), "/")
						to = goPkgFQN[path.Join(mdir, d)]
					}
				}
			}
			if to != "" && im.Optional {
				resolvedOptional[im.Line] = true
			}
			if to != "" && to != from && !seen[to] {
				seen[to] = true
				f.Dependencies = append(f.Dependencies, model.CodeDep{From: from, To: to, Kind: kind, Confidence: "high", File: r.file.Path, Line: im.Line})
			}
		}
		for _, rt := range r.lite.Routes {
			expose(mod, rt.Verb, rt.Path, from, rt.Framework, "medium", r.file.Path, rt.Line)
		}
		for _, t := range r.lite.Tables {
			addTable(f, t.Name, t.Op, from, t.Detail, "high", r.file.Path, t.Line)
		}
	}

	// ---------- ASP.NET minimal APIs and server pages ----------
	for _, r := range results {
		if !include(r) {
			continue
		}
		mod := moduleOf(r.file.Path)
		for _, rt := range r.minimal {
			expose(mod, rt.verb, rt.path, "", rt.framework, "medium", r.file.Path, rt.line)
		}
		if p, fw, ok := serverPage(r.file.Path, r.file.Language, modulePathOf(f, mod)); ok {
			expose(mod, "ANY", p, "", fw, "medium", r.file.Path, 1)
		}
	}

	// ---------- mainframe ----------
	type unitRef struct {
		u   *legacy.Unit
		r   *fileResult
		fqn string
	}
	var units []unitRef
	byName := map[string]map[string]string{} // kind -> name -> FQN
	reg := func(kind, name, fqn string) {
		if byName[kind] == nil {
			byName[kind] = map[string]string{}
		}
		if _, ok := byName[kind][name]; !ok {
			byName[kind][name] = fqn
		}
	}
	for _, r := range results {
		if !include(r) || r.mf == nil {
			continue
		}
		u := r.mf
		pkg := "mainframe." + strings.ReplaceAll(strings.Trim(path.Dir(r.file.Path), "./"), "/", ".")
		pkg = strings.TrimSuffix(pkg, ".")
		layer, reason := LBusiness, "cobol:program"
		switch {
		case u.Kind == "copybook":
			layer, reason = LModel, "cobol:copybook"
		case u.Kind == "job" || u.Kind == "proc":
			layer, reason = LConfig, "jcl:"+u.Kind
		case u.Kind == "mapset":
			layer, reason = LPresentation, "bms:mapset"
		case u.Screen:
			layer, reason = LPresentation, "cics:screen program"
		case u.CICS:
			reason = "cics:online program"
		case u.Batch:
			reason = "batch program"
		}
		fqn := pkg + "." + u.Name
		if !addComp(model.Component{FQN: fqn, Name: u.Name, Kind: u.Kind, Language: r.file.Language, Package: pkg, SourceSet: r.file.SourceSet,
			Module: moduleOf(r.file.Path), Layer: layer, LayerReason: reason, File: r.file.Path, Line: u.Line}) {
			continue
		}
		units = append(units, unitRef{u, r, fqn})
		switch u.Kind {
		case "program":
			reg("program", u.Name, fqn)
		case "copybook":
			reg("copybook", u.Name, fqn)
		case "mapset":
			reg("mapset", u.Name, fqn)
		case "proc":
			reg("proc", u.Name, fqn)
		}
	}
	csdFiles := map[string]string{} // CICS file name -> DSN
	for _, r := range results {
		if !include(r) || r.csd == nil {
			continue
		}
		for _, p := range r.csd.Provides {
			switch p.Kind {
			case "transid":
				comp := byName["program"][p.Detail]
				if comp == "" {
					comp = p.Detail + " (program not in repository)"
				}
				addContract(f, "cics-transaction", "exposes", p.Name, p.Name, comp, "CSD transaction → "+p.Detail, "high", r.file.Path, p.Line)
			case "file":
				csdFiles[p.Name] = p.Detail
				if p.Detail != "" {
					addContract(f, "dataset", "uses", p.Detail, p.Detail, "", "CSD file "+p.Name, "high", r.file.Path, p.Line)
				}
			}
		}
	}
	targetKinds := map[string][]string{
		"call": {"program"}, "link": {"program"}, "xctl": {"program"}, "exec": {"program"}, "copy": {"copybook"},
		"map": {"mapset"}, "proc": {"proc"},
	}
	for _, ur := range units {
		seen := map[string]bool{}
		for _, ref := range ur.u.Refs {
			file, line := ur.r.file.Path, ref.Line
			switch ref.Kind {
			case "file":
				addContract(f, "cics-file", ref.Op, ref.Name, ref.Name, ur.fqn, ref.Detail, "high", file, line)
				if dsn := csdFiles[ref.Name]; dsn != "" {
					addContract(f, "dataset", ref.Op, dsn, dsn, ur.fqn, "via CICS file "+ref.Name, "high", file, line)
				}
				continue
			case "dataset":
				if strings.Contains(ref.Name, "&") {
					continue // symbolic names cannot be compared across jobs
				}
				addContract(f, "dataset", ref.Op, ref.Name, ref.Name, ur.fqn, "JCL DD", "high", file, line)
				continue
			case "transid":
				addContract(f, "cics-transaction", "calls", ref.Name, ref.Name, ur.fqn, ref.Detail, "medium", file, line)
				continue
			case "queue-ts", "queue-td":
				addContract(f, "cics-queue", ref.Op, ref.Name, ref.Name, ur.fqn, strings.ToUpper(strings.TrimPrefix(ref.Kind, "queue-"))+" queue", "medium", file, line)
				continue
			}
			var to string
			for _, k := range targetKinds[ref.Kind] {
				if fq, ok := byName[k][ref.Name]; ok {
					to = fq
				}
			}
			if to == "" || to == ur.fqn || seen[to+ref.Kind] {
				if to == "" && (ref.Kind == "call" || ref.Kind == "link" || ref.Kind == "xctl") && !legacy.IsUtility(ref.Name) {
					addContract(f, "program", "calls", ref.Name, ref.Name, ur.fqn, ref.Kind+" to program outside the repository", "medium", file, line)
				}
				continue
			}
			seen[to+ref.Kind] = true
			conf := "high"
			if ref.Detail != "" && strings.Contains(ref.Detail, "variable") {
				conf = "low"
			}
			f.Dependencies = append(f.Dependencies, model.CodeDep{From: ur.fqn, To: to, Kind: ref.Kind, Confidence: conf, File: file, Line: line})
		}
		if len(ur.u.Dynamic) > 0 {
			f.Diagnostics = append(f.Diagnostics, model.Diagnostic{File: ur.r.file.Path, Kind: "dynamic-call",
				Message: "program transfers to names held in variables: " + strings.Join(ur.u.Dynamic, ", ")})
		}
		// programs exposed to other systems through CICS LINK
		if ur.u.Kind == "program" {
			addContract(f, "program", "exposes", ur.u.Name, ur.u.Name, ur.fqn, "COBOL program", "medium", ur.r.file.Path, ur.u.Line)
		}
	}
}

func modulePathOf(f *model.Facts, id string) string {
	for _, m := range f.Modules {
		if m.ID == id {
			return m.Path
		}
	}
	return ""
}

// serverPage returns the URL path of a directly addressable server page.
func serverPage(rel, language, moduleDir string) (string, string, bool) {
	ext := strings.ToLower(path.Ext(rel))
	switch {
	case language == "JSP" && (ext == ".jsp" || ext == ".jspx"):
		lower := strings.ToLower(rel)
		if strings.Contains(lower, "/web-inf/") {
			return "", "", false // not directly addressable
		}
		for _, root := range []string{"src/main/webapp/", "webcontent/", "webroot/", "web/", "public_html/"} {
			if i := strings.Index(lower, root); i >= 0 {
				return "/" + rel[i+len(root):], "JSP page", true
			}
		}
	case language == "ASP.NET" && (ext == ".aspx" || ext == ".asmx" || ext == ".ashx" || ext == ".svc"):
		p := strings.TrimPrefix(strings.TrimPrefix(rel, moduleDir), "/")
		fw := "ASP.NET Web Forms page"
		switch ext {
		case ".asmx":
			fw = "ASMX web service"
		case ".svc":
			fw = "WCF service"
		case ".ashx":
			fw = "ASP.NET handler"
		}
		return "/" + p, fw, true
	case language == "ASP.NET" && ext == ".cshtml" && strings.Contains(rel, "/Pages/") && !strings.HasPrefix(path.Base(rel), "_"):
		// Razor Pages: /Pages/Basket/Index.cshtml -> /Basket
		i := strings.Index(rel, "/Pages/")
		p := strings.TrimSuffix(rel[i+len("/Pages/"):], ".cshtml")
		p = strings.TrimSuffix(strings.TrimSuffix(p, "Index"), "/")
		return "/" + p, "Razor Page", true
	case language == "Classic ASP":
		p := strings.TrimPrefix(strings.TrimPrefix(rel, moduleDir), "/")
		return "/" + p, "Classic ASP page", true
	}
	return "", "", false
}
