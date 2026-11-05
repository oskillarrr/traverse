package build

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/descriptors"
	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/yamlite"
)

// MaxLockBytes bounds the size of a lockfile read from disk.
var MaxLockBytes int64 = 64 << 20

// lock is the parsed content of one lockfile.
type lock struct {
	rel    string
	prefix string            // coordinate prefix: npm: composer: nuget: pypi: maven:
	direct map[string]string // name (or name@spec) -> exact version
	all    map[string]bool   // name@version of every pinned package
	byName map[string][]string
	owner  string // module that receives the transitive list
}

func newLock(rel, prefix string) *lock {
	return &lock{rel: rel, prefix: prefix, direct: map[string]string{}, all: map[string]bool{}, byName: map[string][]string{}}
}

var rePyLock = regexp.MustCompile(`[-_.]+`)

// norm applies the registry's name comparison rules.
func (l *lock) norm(name string) string {
	switch l.prefix {
	case "pypi:":
		return rePyLock.ReplaceAllString(strings.ToLower(name), "-")
	case "composer:", "nuget:":
		return strings.ToLower(name)
	}
	return name
}

func (l *lock) add(name, version string) {
	if name == "" || version == "" {
		return
	}
	name = l.norm(name)
	k := name + "@" + version
	if !l.all[k] {
		l.all[k] = true
		l.byName[name] = append(l.byName[name], version)
	}
}

// resolve returns the pinned version of a direct dependency.
func (l *lock) resolve(name, spec string) string {
	name = l.norm(name)
	if v := l.direct[name+"@"+spec]; v != "" {
		return v
	}
	if v := l.direct[name]; v != "" {
		return v
	}
	if vs := l.byName[name]; len(vs) == 1 {
		return vs[0]
	}
	return ""
}

var lockFamilies = map[string][]string{
	"npm":      {"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock"},
	"composer": {"composer.lock"},
	"python":   {"poetry.lock", "uv.lock", "Pipfile.lock"},
	"msbuild":  {"packages.lock.json"},
	"nuget":    {"packages.lock.json"},
	"gradle":   {"gradle.lockfile"},
}

// ApplyLocks reads, without executing anything, the lockfiles that sit next to
// a module descriptor (or above it, for monorepo workspaces) and the .NET
// central package management file, fills Dependency.Locked for direct
// dependencies and Module.Transitive with the other pinned packages.
func ApplyLocks(root string, mods []model.Module) []model.Diagnostic {
	var diags []model.Diagnostic
	cache := map[string]*lock{}
	load := func(rel, family string) *lock {
		if l, ok := cache[rel]; ok {
			return l
		}
		cache[rel] = nil
		abs := filepath.Join(root, filepath.FromSlash(rel))
		st, err := os.Stat(abs)
		if err != nil || st.IsDir() {
			return nil
		}
		if st.Size() > MaxLockBytes {
			diags = append(diags, model.Diagnostic{File: rel, Kind: "too-big", Message: "lockfile exceeds the size limit; versions not pinned"})
			return nil
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			return nil
		}
		l, err := parseLock(rel, family, string(b))
		if err != nil {
			diags = append(diags, model.Diagnostic{File: rel, Kind: "lockfile-partial", Message: err.Error()})
		}
		cache[rel] = l
		return l
	}
	find := func(dir, family string, up bool) *lock {
		for {
			for _, name := range lockFamilies[family] {
				rel := path.Join(dir, name)
				if dir == "" || dir == "." {
					rel = name
				}
				if l := load(rel, family); l != nil {
					return l
				}
			}
			if !up || dir == "" || dir == "." || dir == "/" {
				return nil
			}
			dir = path.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}
	cpm := func(dir string) *lock {
		for {
			rel := path.Join(dir, "Directory.Packages.props")
			if dir == "" {
				rel = "Directory.Packages.props"
			}
			if l := load(rel, "cpm"); l != nil {
				return l
			}
			if dir == "" {
				return nil
			}
			dir = path.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}
	for i := range mods {
		m := &mods[i]
		family := m.Build
		var locks []*lock
		if l := find(m.Path, family, family == "npm" || family == "python"); l != nil {
			locks = append(locks, l)
		}
		if family == "msbuild" {
			if l := cpm(m.Path); l != nil {
				locks = append(locks, l)
			}
		}
		if len(locks) == 0 {
			continue
		}
		direct := map[string]bool{}
		for j := range m.Deps {
			d := &m.Deps[j]
			if d.Managed || d.Kind == "project-ref" {
				continue
			}
			for _, l := range locks {
				if !strings.HasPrefix(d.ID, l.prefix) {
					continue
				}
				name := strings.TrimPrefix(d.ID, l.prefix)
				if v := l.resolve(name, d.Version); v != "" {
					d.Locked, d.LockFile = v, l.rel
					direct[l.norm(name)+"@"+v] = true
					break
				}
			}
		}
		for _, l := range locks {
			if l.prefix == "nuget:" && strings.HasSuffix(l.rel, ".props") {
				continue // central versions are not a resolution graph
			}
			if l.owner != "" {
				continue // one module per lockfile receives the transitive list
			}
			l.owner = m.ID
			keys := make([]string, 0, len(l.all))
			for k := range l.all {
				if !direct[k] {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for _, k := range keys {
				i := strings.LastIndexByte(k, '@')
				name, v := k[:i], k[i+1:]
				m.Transitive = append(m.Transitive, model.Dependency{ID: l.prefix + name, Version: v, Kind: "transitive", Resolved: true, File: l.rel})
			}
		}
	}
	return diags
}

var (
	reTrailingComma = regexp.MustCompile(`,(\s*[}\]])`)
	reTomlPkg       = regexp.MustCompile(`(?m)^\[\[package\]\]\s*$`)
	reTomlName      = regexp.MustCompile(`(?m)^name\s*=\s*"([^"]+)"`)
	reTomlVersion   = regexp.MustCompile(`(?m)^version\s*=\s*"([^"]+)"`)
	reMSBuildProp   = regexp.MustCompile(`\$\(([A-Za-z0-9_.\-]+)\)`)
)

func parseLock(rel, family, text string) (*lock, error) {
	base := path.Base(rel)
	switch base {
	case "package-lock.json", "npm-shrinkwrap.json":
		return npmLock(rel, text)
	case "yarn.lock":
		return yarnLock(rel, text), nil
	case "pnpm-lock.yaml":
		return pnpmLock(rel, text)
	case "bun.lock":
		return bunLock(rel, text)
	case "composer.lock":
		return composerLock(rel, text)
	case "poetry.lock", "uv.lock":
		l := newLock(rel, "pypi:")
		for _, blk := range reTomlPkg.Split(text, -1)[1:] {
			if i := strings.Index(blk, "\n["); i > 0 {
				blk = blk[:i]
			}
			n, v := reTomlName.FindStringSubmatch(blk), reTomlVersion.FindStringSubmatch(blk)
			if n != nil && v != nil {
				l.add(n[1], v[1])
			}
		}
		return l, nil
	case "Pipfile.lock":
		var doc map[string]map[string]struct {
			Version string `json:"version"`
		}
		l := newLock(rel, "pypi:")
		if err := json.Unmarshal([]byte(text), &doc); err != nil {
			return l, err
		}
		for _, sec := range []string{"default", "develop"} {
			for n, p := range doc[sec] {
				l.add(n, strings.TrimPrefix(p.Version, "=="))
			}
		}
		return l, nil
	case "packages.lock.json":
		var doc struct {
			Dependencies map[string]map[string]struct {
				Type     string `json:"type"`
				Resolved string `json:"resolved"`
			} `json:"dependencies"`
		}
		l := newLock(rel, "nuget:")
		if err := json.Unmarshal([]byte(text), &doc); err != nil {
			return l, err
		}
		for _, tfm := range doc.Dependencies {
			for n, p := range tfm {
				if p.Type == "Project" {
					continue
				}
				l.add(n, p.Resolved)
				if p.Type == "Direct" || p.Type == "CentralTransitive" {
					l.direct[l.norm(n)] = p.Resolved
				}
			}
		}
		return l, nil
	case "gradle.lockfile":
		l := newLock(rel, "maven:")
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "empty=") {
				continue
			}
			coord, _, _ := strings.Cut(line, "=")
			parts := strings.Split(coord, ":")
			if len(parts) == 3 {
				l.add(parts[0]+":"+parts[1], parts[2])
			}
		}
		return l, nil
	case "Directory.Packages.props":
		l := newLock(rel, "nuget:")
		root, err := descriptors.ParseXML(text)
		if err != nil || root == nil {
			return l, err
		}
		props := map[string]string{}
		for _, pg := range root.All("PropertyGroup") {
			for _, c := range pg.Children {
				props[c.Name] = strings.TrimSpace(c.Text)
			}
		}
		for _, p := range root.All("PackageVersion") {
			v := reMSBuildProp.ReplaceAllStringFunc(p.Attr["Version"], func(s string) string {
				return props[s[2:len(s)-1]]
			})
			if p.Attr["Include"] != "" && v != "" {
				l.direct[l.norm(p.Attr["Include"])] = v
				l.add(p.Attr["Include"], v)
			}
		}
		return l, nil
	}
	return nil, nil
}

func npmLock(rel, text string) (*lock, error) {
	var doc struct {
		Packages map[string]struct {
			Version string `json:"version"`
			Link    bool   `json:"link"`
		} `json:"packages"`
		Dependencies map[string]json.RawMessage `json:"dependencies"`
	}
	l := newLock(rel, "npm:")
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return l, err
	}
	if len(doc.Packages) > 0 { // lockfileVersion 2 and 3
		for k, p := range doc.Packages {
			i := strings.LastIndex(k, "node_modules/")
			if i < 0 || p.Link {
				continue
			}
			name := k[i+len("node_modules/"):]
			l.add(name, p.Version)
			if k == "node_modules/"+name {
				l.direct[name] = p.Version
			}
		}
		return l, nil
	}
	type v1 struct {
		Version      string                     `json:"version"`
		Dependencies map[string]json.RawMessage `json:"dependencies"`
	}
	var walk func(m map[string]json.RawMessage, top bool)
	walk = func(m map[string]json.RawMessage, top bool) {
		for name, raw := range m {
			var d v1
			if json.Unmarshal(raw, &d) != nil {
				continue
			}
			if strings.Contains(d.Version, ":") { // file:, git+, github:
				continue
			}
			l.add(name, d.Version)
			if top {
				l.direct[name] = d.Version
			}
			walk(d.Dependencies, false)
		}
	}
	walk(doc.Dependencies, true)
	return l, nil
}

// specName extracts the package name of a yarn/pnpm specifier ("@s/n@^1.0.0").
func specName(spec string) (name, rng string) {
	spec = strings.Trim(strings.TrimSpace(spec), `"'`)
	i := strings.LastIndexByte(spec, '@')
	if i <= 0 {
		return spec, ""
	}
	name, rng = spec[:i], spec[i+1:]
	rng = strings.TrimPrefix(rng, "npm:")
	return name, rng
}

func yarnLock(rel, text string) *lock {
	l := newLock(rel, "npm:")
	var specs []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line[0] != ' ' && strings.HasSuffix(line, ":") {
			specs = specs[:0]
			for _, s := range strings.Split(strings.TrimSuffix(line, ":"), ",") {
				specs = append(specs, strings.TrimSpace(s))
			}
			continue
		}
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "version ") || strings.HasPrefix(t, "version:") {
			v := strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(t, "version:"), "version")), `"'`)
			for _, s := range specs {
				name, rng := specName(s)
				if name == "__metadata" || strings.Contains(rng, "workspace:") || strings.Contains(rng, "patch:") {
					continue
				}
				l.add(name, v)
				l.direct[name+"@"+rng] = v
			}
			specs = specs[:0]
		}
	}
	return l
}

var rePeerSuffix = regexp.MustCompile(`\(.*$`)

func pnpmLock(rel, text string) (*lock, error) {
	l := newLock(rel, "npm:")
	docs, err := yamlite.ParseAll(text)
	if len(docs) == 0 {
		return l, err
	}
	doc := docs[0]
	for _, e := range doc.Get("packages").Entries() {
		k := strings.TrimPrefix(rePeerSuffix.ReplaceAllString(e.Key, ""), "/")
		var name, v string
		if i := strings.LastIndexByte(k, '@'); i > 0 { // v6+: name@1.2.3
			name, v = k[:i], k[i+1:]
		} else if i := strings.LastIndexByte(k, '/'); i > 0 { // v5: name/1.2.3
			name, v = k[:i], k[i+1:]
		}
		if name != "" && !strings.Contains(v, ":") {
			l.add(name, v)
		}
	}
	directs := func(n *yamlite.Node) {
		for _, sec := range []string{"dependencies", "devDependencies", "optionalDependencies"} {
			for _, e := range n.Get(sec).Entries() {
				v := e.Value.Str()
				if v == "" {
					v = e.Value.S("version")
				}
				v = rePeerSuffix.ReplaceAllString(v, "")
				if v != "" && !strings.Contains(v, ":") {
					l.direct[e.Key] = v
				}
			}
		}
	}
	if imp := doc.Get("importers"); imp != nil {
		directs(imp.Get("."))
	} else {
		directs(doc)
	}
	return l, err
}

func bunLock(rel, text string) (*lock, error) {
	l := newLock(rel, "npm:")
	var doc struct {
		Packages map[string][]json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal([]byte(reTrailingComma.ReplaceAllString(text, "$1")), &doc); err != nil {
		return l, err
	}
	for key, arr := range doc.Packages {
		if len(arr) == 0 {
			continue
		}
		var id string
		if json.Unmarshal(arr[0], &id) != nil {
			continue
		}
		name, v := specName(id)
		if name == "" || v == "" || strings.Contains(v, ":") {
			continue
		}
		l.add(name, v)
		if key == name {
			l.direct[name] = v
		}
	}
	return l, nil
}

func composerLock(rel, text string) (*lock, error) {
	var doc struct {
		Packages    []struct{ Name, Version string } `json:"packages"`
		PackagesDev []struct{ Name, Version string } `json:"packages-dev"`
	}
	l := newLock(rel, "composer:")
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		return l, err
	}
	for _, p := range append(doc.Packages, doc.PackagesDev...) {
		v := strings.TrimPrefix(p.Version, "v")
		if strings.HasPrefix(v, "dev-") {
			continue
		}
		l.add(p.Name, v)
		l.direct[l.norm(p.Name)] = v
	}
	return l, nil
}
