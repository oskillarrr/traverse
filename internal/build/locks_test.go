package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func writeFiles(t *testing.T, files map[string]string) string {
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func dep(id, v string) model.Dependency {
	return model.Dependency{ID: id, Version: v, Kind: "declared", Resolved: true}
}

func locked(m model.Module) map[string]string {
	out := map[string]string{}
	for _, d := range m.Deps {
		out[d.ID] = d.Locked
	}
	for _, d := range m.Transitive {
		out["T "+d.ID] = d.Version
	}
	return out
}

func TestLockfiles(t *testing.T) {
	root := writeFiles(t, map[string]string{
		"web/package-lock.json": `{"lockfileVersion":3,"packages":{"":{"name":"web"},
		  "node_modules/lodash":{"version":"4.17.20"},"node_modules/@scope/ui":{"version":"1.2.3"},
		  "node_modules/@scope/ui/node_modules/lodash":{"version":"3.10.1"},"node_modules/ms":{"version":"2.1.3"},
		  "node_modules/local":{"link":true}}}`,
		"old/package-lock.json": `{"lockfileVersion":1,"dependencies":{"express":{"version":"4.16.0","dependencies":{"qs":{"version":"6.5.1"}}},"x":{"version":"file:../x"}}}`,
		"y/yarn.lock":           "# yarn lockfile v1\n\n\"@babel/core@^7.0.0\", \"@babel/core@^7.1.0\":\n  version \"7.1.2\"\n  dependencies:\n    debug \"^4.1.0\"\n\ndebug@^4.1.0:\n  version \"4.3.4\"\n",
		"b/yarn.lock":           "__metadata:\n  version: 6\n\n\"lodash@npm:^4.17.0\":\n  version: 4.17.21\n  resolution: \"lodash@npm:4.17.21\"\n",
		"p/pnpm-lock.yaml":      "lockfileVersion: '9.0'\n\nimporters:\n\n  .:\n    dependencies:\n      axios:\n        specifier: ^1.6.0\n        version: 1.6.7(debug@4.3.4)\n\npackages:\n\n  axios@1.6.7:\n    resolution: {integrity: sha512-x}\n\n  '@types/node@20.1.0':\n    resolution: {integrity: sha512-y}\n",
		"bun/bun.lock":          "{\n  \"lockfileVersion\": 1,\n  \"workspaces\": {\"\": {\"dependencies\": {\"marked\": \"^17.0.1\",},},},\n  \"packages\": {\n    \"marked\": [\"marked@17.0.2\", \"\", {}, \"sha512-z\"],\n    \"a/b\": [\"b@1.0.0\", \"\", {}, \"sha\"],\n  },\n}\n",
		"php/composer.lock":     `{"packages":[{"name":"Monolog/Monolog","version":"2.9.1"},{"name":"psr/log","version":"v3.0.0"}],"packages-dev":[{"name":"phpunit/phpunit","version":"dev-main"}]}`,
		"py/poetry.lock":        "[[package]]\nname = \"Requests\"\nversion = \"2.31.0\"\n\n[package.dependencies]\nurllib3 = \">=1.21.1\"\n\n[[package]]\nname = \"urllib3\"\nversion = \"2.0.7\"\n",
		"Directory.Packages.props": `<Project><PropertyGroup><AspNetVersion>8.0.2</AspNetVersion></PropertyGroup><ItemGroup>
		  <PackageVersion Include="Newtonsoft.Json" Version="13.0.1" /><PackageVersion Include="Microsoft.AspNetCore.Mvc" Version="$(AspNetVersion)" /></ItemGroup></Project>`,
		"src/Api/packages.lock.json": `{"version":1,"dependencies":{"net8.0":{"Serilog":{"type":"Direct","resolved":"3.1.1"},"Serilog.Sinks":{"type":"Transitive","resolved":"1.0.0"}}}}`,
		"g/gradle.lockfile":          "# comment\norg.yaml:snakeyaml:1.33=compileClasspath\ncom.fasterxml.jackson.core:jackson-databind:2.13.0=runtimeClasspath\nempty=\n",
	})
	mods := []model.Module{
		{ID: "npm:web", Build: "npm", Path: "web", Deps: []model.Dependency{dep("npm:lodash", "^4.17.0"), dep("npm:@scope/ui", "~1.2.0")}},
		{ID: "npm:old", Build: "npm", Path: "old", Deps: []model.Dependency{dep("npm:express", "^4.0.0")}},
		{ID: "npm:y", Build: "npm", Path: "y/pkg/sub", Deps: []model.Dependency{dep("npm:@babel/core", "^7.1.0")}},
		{ID: "npm:b", Build: "npm", Path: "b", Deps: []model.Dependency{dep("npm:lodash", "^4.17.0")}},
		{ID: "npm:p", Build: "npm", Path: "p", Deps: []model.Dependency{dep("npm:axios", "^1.6.0")}},
		{ID: "npm:bun", Build: "npm", Path: "bun", Deps: []model.Dependency{dep("npm:marked", "^17.0.1")}},
		{ID: "composer:app", Build: "composer", Path: "php", Deps: []model.Dependency{dep("composer:monolog/monolog", "^2.0")}},
		{ID: "pypi:tool", Build: "python", Path: "py", Deps: []model.Dependency{dep("pypi:requests", ">=2.0")}},
		{ID: "nuget:Api", Build: "msbuild", Path: "src/Api", Deps: []model.Dependency{dep("nuget:Newtonsoft.Json", ""), dep("nuget:Microsoft.AspNetCore.Mvc", ""), dep("nuget:Serilog", "")}},
		{ID: "maven:g:a", Build: "gradle", Path: "g", Deps: []model.Dependency{dep("maven:org.yaml:snakeyaml", "")}},
	}
	if diags := ApplyLocks(root, mods); len(diags) != 0 {
		t.Errorf("diagnostics: %v", diags)
	}
	want := []map[string]string{
		{"npm:lodash": "4.17.20", "npm:@scope/ui": "1.2.3", "T npm:ms": "2.1.3"},
		{"npm:express": "4.16.0", "T npm:qs": "6.5.1"},
		{"npm:@babel/core": "7.1.2", "T npm:debug": "4.3.4"},
		{"npm:lodash": "4.17.21"},
		{"npm:axios": "1.6.7", "T npm:@types/node": "20.1.0"},
		{"npm:marked": "17.0.2", "T npm:b": "1.0.0"},
		{"composer:monolog/monolog": "2.9.1", "T composer:psr/log": "3.0.0"},
		{"pypi:requests": "2.31.0", "T pypi:urllib3": "2.0.7"},
		{"nuget:Newtonsoft.Json": "13.0.1", "nuget:Microsoft.AspNetCore.Mvc": "8.0.2", "nuget:Serilog": "3.1.1", "T nuget:serilog.sinks": "1.0.0"},
		{"maven:org.yaml:snakeyaml": "1.33", "T maven:com.fasterxml.jackson.core:jackson-databind": "2.13.0"},
	}
	for i, m := range mods {
		got := locked(m)
		for k, v := range want[i] {
			if got[k] != v {
				t.Errorf("%s: %s = %q, want %q (all: %v)", m.ID, k, got[k], v, got)
			}
		}
	}
	// the nested lodash 3.10.1 is transitive; the link entry and dev-main are skipped
	if got := locked(mods[0]); got["T npm:lodash"] != "3.10.1" {
		t.Errorf("nested npm copy: %v", got)
	}
	for _, d := range mods[6].Transitive {
		if d.ID == "composer:phpunit/phpunit" {
			t.Error("dev branches must be skipped")
		}
	}
}

func TestLockOwnership(t *testing.T) {
	root := writeFiles(t, map[string]string{"package-lock.json": `{"lockfileVersion":3,"packages":{"node_modules/a":{"version":"1.0.0"},"node_modules/b":{"version":"2.0.0"}}}`})
	mods := []model.Module{
		{ID: "npm:x", Build: "npm", Path: "packages/x", Deps: []model.Dependency{dep("npm:a", "^1")}},
		{ID: "npm:y", Build: "npm", Path: "packages/y", Deps: []model.Dependency{dep("npm:b", "^2")}},
	}
	ApplyLocks(root, mods)
	if mods[0].Deps[0].Locked != "1.0.0" || mods[1].Deps[0].Locked != "2.0.0" {
		t.Errorf("workspace root lockfile: %+v %+v", mods[0].Deps, mods[1].Deps)
	}
	if len(mods[1].Transitive) != 0 {
		t.Error("a lockfile's transitive list is attached to one module only")
	}
}
