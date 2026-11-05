// Package build reads build manifests of many ecosystems without running any
// build tool. Each module gets a coordinate (e.g. "maven:g:a", "npm:name")
// that is used to link repositories in the portfolio view.
package build

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"github.com/oskillarrr/traverse/internal/descriptors"
	"github.com/oskillarrr/traverse/internal/model"
)

// IsManifest reports whether a relative path is a build manifest handled here.
func IsManifest(rel string) bool {
	b := strings.ToLower(path.Base(rel))
	switch b {
	case "pom.xml", "build.xml", "ivy.xml", "build.gradle", "build.gradle.kts", "package.json",
		"packages.config", "pyproject.toml", "composer.json", "go.mod", "setup.py":
		return true
	}
	if strings.HasPrefix(b, "requirements") && strings.HasSuffix(b, ".txt") {
		return true
	}
	switch path.Ext(b) {
	case ".csproj", ".vbproj", ".fsproj", ".vcxproj":
		return true
	}
	return false
}

// Parse returns a module for any manifest except pom.xml (see Pom/ResolveMaven).
func Parse(rel, text string) *model.Module {
	b := strings.ToLower(path.Base(rel))
	dir := path.Dir(rel)
	if dir == "." {
		dir = ""
	}
	m := &model.Module{Path: dir, Descriptor: rel}
	switch {
	case b == "build.xml":
		return ant(m, text)
	case b == "ivy.xml":
		return ivy(m, text)
	case strings.HasPrefix(b, "build.gradle"):
		return gradle(m, text)
	case b == "package.json":
		return npm(m, text)
	case b == "composer.json":
		return composer(m, text)
	case b == "go.mod":
		return gomod(m, text)
	case b == "packages.config":
		return packagesConfig(m, text)
	case b == "pyproject.toml", b == "setup.py", strings.HasPrefix(b, "requirements"):
		return python(m, b, text)
	default:
		return msbuild(m, text)
	}
}

func base(dir string) string {
	if dir == "" {
		return "root"
	}
	return path.Base(dir)
}

func lineOf(text string, off int) int { return strings.Count(text[:off], "\n") + 1 }

func ant(m *model.Module, text string) *model.Module {
	root, _ := descriptors.ParseXML(text)
	doc := root.Root()
	if doc == nil || doc.Name != "project" {
		return nil
	}
	m.Build = "ant"
	m.Name = doc.Attr["name"]
	if m.Name == "" {
		m.Name = base(m.Path)
	}
	m.ID = "ant:" + m.Name
	for _, k := range []string{"ear", "war", "jar"} {
		if len(doc.All(k)) > 0 {
			m.Packaging = k
			break
		}
	}
	for _, j := range doc.All("javac") {
		if s := j.Attr["source"]; s != "" && !strings.Contains(s, "${") {
			m.Runtime = "java " + s
		} else if t := j.Attr["target"]; t != "" && !strings.Contains(t, "${") {
			m.Runtime = "java " + t
		}
	}
	return m
}

func ivy(m *model.Module, text string) *model.Module {
	root, _ := descriptors.ParseXML(text)
	doc := root.Root()
	if doc == nil || doc.Name != "ivy-module" {
		return nil
	}
	m.Build = "ivy"
	info := doc.Child("info")
	m.Name = info.Attr["module"]
	m.ID = "maven:" + info.Attr["organisation"] + ":" + m.Name
	for _, d := range doc.All("dependency") {
		m.Deps = append(m.Deps, model.Dependency{ID: "maven:" + d.Attr["org"] + ":" + d.Attr["name"], Version: d.Attr["rev"],
			Scope: d.Attr["conf"], Kind: "declared", Resolved: !strings.Contains(d.Attr["rev"], "$"), Line: d.Line, File: m.Descriptor})
	}
	return m
}

var (
	reGradleDep    = regexp.MustCompile(`(?m)^\s*(implementation|api|compile|compileOnly|runtime|runtimeOnly|testImplementation|testCompile|providedCompile|providedRuntime|annotationProcessor|earlib|deploy)\s*\(?\s*['"]([^:'"\s]+):([^:'"\s]+)(?::([^'"\s]+))?['"]`)
	reGradleProj   = regexp.MustCompile(`(?m)^\s*(implementation|api|compile|runtimeOnly|testImplementation|earlib|deploy)\s*\(?\s*project\s*\(\s*(?:path\s*:\s*)?['"]:?([^'"]+)['"]`)
	reGradleGroup  = regexp.MustCompile(`(?m)^\s*group\s*=\s*['"]([^'"]+)['"]`)
	reGradleVer    = regexp.MustCompile(`(?m)^\s*version\s*=\s*['"]([^'"]+)['"]`)
	reGradleCompat = regexp.MustCompile(`(?m)sourceCompatibility\s*=\s*['"]?(?:JavaVersion\.VERSION_)?([\w.]+)`)
)

func gradle(m *model.Module, text string) *model.Module {
	m.Build = "gradle"
	m.Name = base(m.Path)
	g := ""
	if x := reGradleGroup.FindStringSubmatch(text); x != nil {
		g = x[1]
	}
	if x := reGradleVer.FindStringSubmatch(text); x != nil {
		m.Version = x[1]
	}
	if x := reGradleCompat.FindStringSubmatch(text); x != nil {
		m.Runtime = "java " + strings.ReplaceAll(x[1], "_", ".")
	}
	m.ID = "maven:" + g + ":" + m.Name
	switch {
	case strings.Contains(text, "'war'") || strings.Contains(text, `"war"`) || strings.Contains(text, "id 'war'"):
		m.Packaging = "war"
	case strings.Contains(text, "'ear'") || strings.Contains(text, `"ear"`):
		m.Packaging = "ear"
	}
	for _, x := range reGradleDep.FindAllStringSubmatchIndex(text, -1) {
		v := ""
		if x[8] >= 0 {
			v = text[x[8]:x[9]]
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "maven:" + text[x[4]:x[5]] + ":" + text[x[6]:x[7]], Version: v,
			Scope: text[x[2]:x[3]], Kind: "declared", Resolved: v != "" && !strings.Contains(v, "$"), Line: lineOf(text, x[0]), File: m.Descriptor})
	}
	for _, x := range reGradleProj.FindAllStringSubmatchIndex(text, -1) {
		p := strings.ReplaceAll(text[x[4]:x[5]], ":", "/")
		m.Deps = append(m.Deps, model.Dependency{ID: "gradle-project:" + p, Scope: text[x[2]:x[3]], Kind: "project-ref", Resolved: true, Line: lineOf(text, x[0]), File: m.Descriptor})
	}
	return m
}

func npm(m *model.Module, text string) *model.Module {
	var p struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Engines         map[string]string `json:"engines"`
	}
	if json.Unmarshal([]byte(text), &p) != nil {
		return nil
	}
	m.Build, m.Name, m.Version = "npm", p.Name, p.Version
	if m.Name == "" {
		m.Name = base(m.Path)
	}
	m.ID = "npm:" + m.Name
	if n := p.Engines["node"]; n != "" {
		m.Runtime = "node " + n
	}
	for _, set := range []struct {
		deps  map[string]string
		scope string
	}{{p.Dependencies, "runtime"}, {p.DevDependencies, "dev"}} {
		for k, v := range set.deps {
			m.Deps = append(m.Deps, model.Dependency{ID: "npm:" + k, Version: v, Scope: set.scope, Kind: "declared", Resolved: true, File: m.Descriptor})
		}
	}
	return m
}

func composer(m *model.Module, text string) *model.Module {
	var p struct {
		Name       string            `json:"name"`
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal([]byte(text), &p) != nil {
		return nil
	}
	m.Build, m.Name = "composer", p.Name
	if m.Name == "" {
		m.Name = base(m.Path)
	}
	m.ID = "composer:" + m.Name
	for k, v := range p.Require {
		if k == "php" {
			m.Runtime = "php " + v
			continue
		}
		if composerPlatform(k) {
			continue
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "composer:" + k, Version: v, Scope: "runtime", Kind: "declared", Resolved: true, File: m.Descriptor})
	}
	for k, v := range p.RequireDev {
		if k == "php" || composerPlatform(k) {
			continue
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "composer:" + k, Version: v, Scope: "dev", Kind: "declared", Resolved: true, File: m.Descriptor})
	}
	return m
}

var (
	reGoModule  = regexp.MustCompile(`(?m)^module\s+(\S+)`)
	reGoVersion = regexp.MustCompile(`(?m)^go\s+(\S+)`)
	reGoReq     = regexp.MustCompile(`(?m)^\s*(?:require\s+)?([a-zA-Z0-9.\-_/~]+\.[a-z]{2,}[^\s]*)\s+(v[^\s]+)`)
)

func gomod(m *model.Module, text string) *model.Module {
	x := reGoModule.FindStringSubmatch(text)
	if x == nil {
		return nil
	}
	m.Build, m.Name, m.ID = "go", x[1], "go:"+x[1]
	if v := reGoVersion.FindStringSubmatch(text); v != nil {
		m.Runtime = "go " + v[1]
	}
	for _, r := range reGoReq.FindAllStringSubmatchIndex(text, -1) {
		name := text[r[2]:r[3]]
		if name == m.Name {
			continue
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "go:" + name, Version: text[r[4]:r[5]], Kind: "declared", Resolved: true, Line: lineOf(text, r[0]), File: m.Descriptor})
	}
	return m
}

func packagesConfig(m *model.Module, text string) *model.Module {
	root, _ := descriptors.ParseXML(text)
	m.Build, m.Name = "nuget", base(m.Path)
	m.ID = "nuget-packages:" + m.Path
	for _, p := range root.All("package") {
		m.Deps = append(m.Deps, model.Dependency{ID: "nuget:" + p.Attr["id"], Version: p.Attr["version"], Kind: "declared", Resolved: true, Line: p.Line, File: m.Descriptor})
	}
	return m
}

var (
	reReq        = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9][A-Za-z0-9._\-]*)(?:\[[^\]]*\])?\s*(?:(==|>=|<=|~=|>|<|!=)\s*([^\s;#,]+))?`)
	rePyName     = regexp.MustCompile(`(?m)^name\s*=\s*["']([^"']+)["']`)
	rePyDeps     = regexp.MustCompile(`(?s)dependencies\s*=\s*\[(.*?)\]`)
	rePyDep      = regexp.MustCompile(`["']([A-Za-z0-9][A-Za-z0-9._\-]*)(?:\[[^\]]*\])?\s*([<>=!~]=?\s*[^"';,]+)?`)
	rePySetupNm  = regexp.MustCompile(`name\s*=\s*["']([^"']+)["']`)
	rePyRequires = regexp.MustCompile(`(?m)python_requires\s*=\s*["']([^"']+)["']|requires-python\s*=\s*["']([^"']+)["']`)
)

func python(m *model.Module, b, text string) *model.Module {
	m.Build, m.Name = "python", base(m.Path)
	switch {
	case strings.HasPrefix(b, "requirements"):
		for _, l := range strings.Split(text, "\n") {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "-") {
				continue
			}
			if x := reReq.FindStringSubmatch(l); x != nil {
				m.Deps = append(m.Deps, model.Dependency{ID: "pypi:" + strings.ToLower(x[1]), Version: x[3], Kind: "declared", Resolved: true, File: m.Descriptor})
			}
		}
		m.ID = "pypi-requirements:" + m.Path
		return m
	case b == "setup.py":
		if x := rePySetupNm.FindStringSubmatch(text); x != nil {
			m.Name = x[1]
		}
	default:
		if x := rePyName.FindStringSubmatch(text); x != nil {
			m.Name = x[1]
		}
		if x := rePyDeps.FindStringSubmatch(text); x != nil {
			for _, d := range rePyDep.FindAllStringSubmatch(x[1], -1) {
				m.Deps = append(m.Deps, model.Dependency{ID: "pypi:" + strings.ToLower(d[1]), Version: strings.TrimSpace(d[2]), Kind: "declared", Resolved: true, File: m.Descriptor})
			}
		}
	}
	if x := rePyRequires.FindStringSubmatch(text); x != nil {
		m.Runtime = "python " + x[1] + x[2]
	}
	m.ID = "pypi:" + strings.ToLower(m.Name)
	return m
}

func msbuild(m *model.Module, text string) *model.Module {
	root, _ := descriptors.ParseXML(text)
	doc := root.Root()
	if doc == nil || doc.Name != "Project" {
		return nil
	}
	m.Build = "msbuild"
	name := ""
	for _, n := range doc.All("AssemblyName") {
		name = strings.TrimSpace(n.Text)
		break
	}
	if name == "" {
		name = strings.TrimSuffix(path.Base(m.Descriptor), path.Ext(m.Descriptor))
	}
	m.Name = name
	m.ID = "nuget:" + name
	for _, tag := range []string{"TargetFramework", "TargetFrameworks", "TargetFrameworkVersion"} {
		for _, n := range doc.All(tag) {
			m.Runtime = ".NET " + strings.TrimSpace(n.Text)
		}
	}
	for _, n := range doc.All("OutputType") {
		m.Packaging = strings.ToLower(strings.TrimSpace(n.Text))
	}
	for _, p := range doc.All("PackageReference") {
		v := p.Attr["Version"]
		if v == "" {
			v = p.T("Version")
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "nuget:" + p.Attr["Include"], Version: v, Kind: "declared", Resolved: true, Line: p.Line, File: m.Descriptor})
	}
	for _, p := range doc.All("Reference") {
		inc := p.Attr["Include"]
		nm := strings.TrimSpace(strings.Split(inc, ",")[0])
		if nm == "" || strings.HasPrefix(nm, "System") || strings.HasPrefix(nm, "Microsoft.") || nm == "mscorlib" {
			continue
		}
		ver := ""
		for _, part := range strings.Split(inc, ",") {
			if kv := strings.SplitN(strings.TrimSpace(part), "=", 2); len(kv) == 2 && kv[0] == "Version" {
				ver = kv[1]
			}
		}
		m.Deps = append(m.Deps, model.Dependency{ID: "nuget:" + nm, Version: ver, Kind: "assembly-ref", Resolved: true, Line: p.Line, File: m.Descriptor})
	}
	for _, p := range doc.All("ProjectReference") {
		inc := strings.ReplaceAll(p.Attr["Include"], `\`, "/")
		target := path.Clean(path.Join(m.Path, inc))
		m.Deps = append(m.Deps, model.Dependency{ID: "path:" + target, Kind: "project-ref", Resolved: true, Line: p.Line, File: m.Descriptor})
	}
	return m
}

var reJarName = regexp.MustCompile(`^(.+?)[-_]v?(\d+(?:[.\-_]\w+)*)\.jar$`)

// VendoredJar turns a committed library file (typical in legacy repos without
// dependency management, e.g. WEB-INF/lib) into a dependency.
func VendoredJar(rel string) model.Dependency {
	b := path.Base(rel)
	name, ver := strings.TrimSuffix(b, path.Ext(b)), ""
	if x := reJarName.FindStringSubmatch(b); x != nil {
		name, ver = x[1], x[2]
	}
	return model.Dependency{ID: "jar:" + strings.ToLower(name), Version: ver, Kind: "vendored-jar", Resolved: ver != "", File: rel}
}

// composerPlatform reports Composer platform requirements (PHP extensions and
// system libraries), which are not packages.
func composerPlatform(k string) bool {
	k = strings.ToLower(k)
	return strings.HasPrefix(k, "ext-") || strings.HasPrefix(k, "lib-") || k == "composer-plugin-api" || k == "composer-runtime-api" || k == "composer"
}
