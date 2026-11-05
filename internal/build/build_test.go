package build

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestMavenInheritanceAndProperties(t *testing.T) {
	parent := ParsePom("pom.xml", `<project><groupId>g</groupId><artifactId>parent</artifactId><version>1.2</version>
<properties><lib.version>3.4</lib.version><java.version>1.6</java.version></properties>
<dependencyManagement><dependencies><dependency><groupId>x</groupId><artifactId>y</artifactId><version>${lib.version}</version></dependency></dependencies></dependencyManagement></project>`)
	child := ParsePom("app/pom.xml", `<project><parent><groupId>g</groupId><artifactId>parent</artifactId><version>1.2</version></parent>
<artifactId>app</artifactId><packaging>war</packaging><dependencies><dependency><groupId>x</groupId><artifactId>y</artifactId></dependency>
<dependency><groupId>${project.groupId}</groupId><artifactId>other</artifactId><version>${project.version}</version></dependency></dependencies></project>`)
	mods := ResolveMaven([]*Pom{parent, child})
	var app = mods[1]
	if app.ID != "maven:g:app" || app.Version != "1.2" || app.Runtime != "java 1.6" {
		t.Fatalf("app: %+v", app)
	}
	if d := app.Deps[0]; d.Version != "3.4" || !d.Resolved {
		t.Errorf("managed version not applied: %+v", d)
	}
	if d := app.Deps[1]; d.ID != "maven:g:other" || d.Version != "1.2" {
		t.Errorf("project properties: %+v", d)
	}
	if !mods[0].Deps[0].Managed {
		t.Errorf("dependencyManagement must be flagged")
	}
}

func TestOtherEcosystems(t *testing.T) {
	g := Parse("svc/build.gradle", "group = 'org.x'\napply plugin: 'war'\ndependencies {\n  implementation 'org.a:b:1.0'\n  compile project(':core')\n}\n")
	if g.ID != "maven:org.x:svc" || g.Packaging != "war" || len(g.Deps) != 2 {
		t.Errorf("gradle: %+v", g)
	}
	n := Parse("web/package.json", `{"name":"front","dependencies":{"angular":"1.5.0"}}`)
	if n.ID != "npm:front" || len(n.Deps) != 1 {
		t.Errorf("npm: %+v", n)
	}
	c := Parse("P/P.csproj", `<Project><PropertyGroup><AssemblyName>P.Web</AssemblyName><TargetFrameworkVersion>v4.5</TargetFrameworkVersion></PropertyGroup>
<ItemGroup><PackageReference Include="Newtonsoft.Json" Version="9.0.1"/><ProjectReference Include="..\Core\Core.csproj"/></ItemGroup></Project>`)
	if c.ID != "nuget:P.Web" || c.Runtime != ".NET v4.5" || len(c.Deps) != 2 || c.Deps[1].ID != "path:Core/Core.csproj" {
		t.Errorf("csproj: %+v", c)
	}
	if j := VendoredJar("WEB-INF/lib/commons-lang-2.6.jar"); j.ID != "jar:commons-lang" || j.Version != "2.6" {
		t.Errorf("jar: %+v", j)
	}
}

func TestJarCoordinates(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string, props map[string]string) string {
		p := filepath.Join(dir, name)
		f, _ := os.Create(p)
		zw := zip.NewWriter(f)
		for n, body := range props {
			w, _ := zw.Create(n)
			w.Write([]byte(body))
		}
		zw.Close()
		f.Close()
		return p
	}
	p := mk("lib-x.jar", map[string]string{"META-INF/maven/org.x/lib-x/pom.properties": "#Generated\nversion=1.2.3\ngroupId=org.x\nartifactId=lib-x\n"})
	if d := VendoredJarAt(p, "WEB-INF/lib/lib-x.jar", 100); d.ID != "maven:org.x:lib-x" || d.Version != "1.2.3" || !d.Resolved || d.Kind != "vendored-jar" {
		t.Errorf("pom.properties: %+v", d)
	}
	shaded := mk("app-all-2.0.jar", map[string]string{
		"META-INF/maven/org.a/dep/pom.properties":     "groupId=org.a\nartifactId=dep\nversion=1\n",
		"META-INF/maven/org.b/app-all/pom.properties": "groupId=org.b\nartifactId=app-all\nversion=2.0\n"})
	if d := VendoredJarAt(shaded, "lib/app-all-2.0.jar", 100); d.ID != "maven:org.b:app-all" {
		t.Errorf("shaded: %+v", d)
	}
	none := mk("old-1.0.jar", map[string]string{"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\n"})
	if d := VendoredJarAt(none, "lib/old-1.0.jar", 100); d.ID != "jar:old" || d.Version != "1.0" {
		t.Errorf("fallback: %+v", d)
	}
	if d := VendoredJarAt(filepath.Join(dir, "missing.jar"), "lib/missing-2.jar", 100); d.ID != "jar:missing" {
		t.Errorf("unreadable: %+v", d)
	}
}
