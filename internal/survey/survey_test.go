package survey

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func pom(g, a, v, packaging string, modules ...string) string {
	var m strings.Builder
	for _, x := range modules {
		m.WriteString("<module>" + x + "</module>")
	}
	return `<project><groupId>` + g + `</groupId><artifactId>` + a + `</artifactId><version>` + v +
		`</version><packaging>` + packaging + `</packaging><modules>` + m.String() + `</modules></project>`
}

// project writes a two-module Maven project with n Java classes.
func project(t *testing.T, root, dir, version string, n int) {
	write(t, root, dir+"/pom.xml", pom("es.acme", "orders", version, "pom", "orders-srv", "orders-war"))
	write(t, root, dir+"/orders-srv/pom.xml", pom("es.acme", "orders-srv", version, "jar"))
	write(t, root, dir+"/orders-war/pom.xml", pom("es.acme", "orders-war", version, "war"))
	for i := 0; i < n; i++ {
		write(t, root, dir+"/orders-srv/src/main/java/orders/C"+string(rune('A'+i))+".java", "class C"+string(rune('A'+i))+" {}")
	}
	write(t, root, dir+"/orders-war/src/main/webapp/WEB-INF/web.xml", "<web-app/>")
}

func TestReleaseCopiesKeepTrunk(t *testing.T) {
	root := t.TempDir()
	write(t, root, "bbdd/01_ddl.sql", "create table t(a int);")
	write(t, root, "doc/manual.docx", "x")
	write(t, root, "ea/modelo.eap", "x")
	project(t, root, "fuente/trunk", "2.10.2", 10)
	project(t, root, "fuente/develop", "2.11.0", 11)
	project(t, root, "fuente/entregas/2.10.2", "2.10.2", 10)
	project(t, root, "fuente/entregas/1.0", "1.0", 7)
	project(t, root, "fuente/entregas/JBOSS7_Entrega", "1.0", 6)

	r, err := Run(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := r.Summary
	if len(s.Roots) != 1 || s.Roots[0].Path != "fuente/trunk" || !s.Roots[0].Main || s.Roots[0].Reason != "trunk" {
		t.Fatalf("main line: %+v", s.Roots)
	}
	if len(s.Copies) != 4 {
		t.Fatalf("copies: %+v", s.Copies)
	}
	for _, c := range s.Copies {
		if c.Path == "fuente/entregas/2.10.2" && !c.Identical {
			t.Errorf("entregas/2.10.2 should be identical to trunk: %+v", c)
		}
		if c.Path == "fuente/develop" && (c.Identical || c.Version != "2.11.0") {
			t.Errorf("develop: %+v", c)
		}
	}
	kinds := map[string]string{}
	for _, a := range s.NonCode {
		kinds[a.Path] = a.Kind
	}
	if kinds["doc"] != "docs" || kinds["ea"] != "models" || kinds["bbdd"] != "data" {
		t.Errorf("non-code areas: %+v", s.NonCode)
	}
	if len(r.Skip) != 4 {
		t.Errorf("skip: %v", r.Skip)
	}
	if !strings.Contains(s.Decision, "fuente/trunk (2.10.2), 4 copies") {
		t.Errorf("decision: %s", s.Decision)
	}
}

func TestHighestVersionWithoutTrunk(t *testing.T) {
	root := t.TempDir()
	project(t, root, "releases/1.0", "1.0", 6)
	project(t, root, "releases/1.10", "1.10", 8)
	project(t, root, "releases/1.9", "1.9", 7)
	r, _ := Run(root, Options{})
	if len(r.Summary.Roots) != 1 || r.Summary.Roots[0].Path != "releases/1.10" || r.Summary.Roots[0].Reason != "highest-version" {
		t.Fatalf("roots: %+v", r.Summary.Roots)
	}
}

func TestDistinctModulesAreNotCopies(t *testing.T) {
	root := t.TempDir()
	for _, a := range []string{"svc-a", "svc-b"} {
		write(t, root, a+"/pom.xml", pom("es.x", a, "1.0", "jar"))
		for _, c := range []string{"A", "B", "C", "D", "E", "F"} {
			write(t, root, a+"/src/main/resources/"+c+".properties", "k=v")
		}
	}
	r, _ := Run(root, Options{})
	if len(r.Summary.Copies) != 0 || len(r.Summary.Roots) != 2 {
		t.Fatalf("roots %+v copies %+v", r.Summary.Roots, r.Summary.Copies)
	}
}

func TestCopiesWithoutDescriptors(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"cobol", "cobol_copia"} {
		for _, p := range []string{"A", "B", "C", "D", "E", "F"} {
			write(t, root, d+"/src/PGM"+p+".cbl", "       IDENTIFICATION DIVISION.")
		}
	}
	old := time.Now().Add(-48 * time.Hour)
	filepath.Walk(filepath.Join(root, "cobol_copia"), func(p string, _ os.FileInfo, _ error) error {
		return os.Chtimes(p, old, old)
	})
	r, _ := Run(root, Options{})
	if len(r.Summary.Copies) != 1 || r.Summary.Copies[0].Path != "cobol_copia" || !r.Summary.Copies[0].Identical {
		t.Fatalf("copies: %+v roots %+v", r.Summary.Copies, r.Summary.Roots)
	}
}

func TestBinaryOnlyModules(t *testing.T) {
	root := t.TempDir()
	write(t, root, "sign-gateway/pom.xml", pom("com.example.sign", "sign-gateway", "2.3", "war"))
	write(t, root, "sign-gateway/sign-gateway-2.4-INT.war", "PK")
	write(t, root, "sign-admin/pom.xml", pom("com.example.sign", "sign-admin", "2.3", "war"))
	write(t, root, "sign-admin/sign-admin-2.4-INT.war", "PK")
	write(t, root, "apps/jboss/fire_config/config.properties", "a=b")
	r, _ := Run(root, Options{})
	if len(r.Summary.BinaryOnly) != 2 || r.Summary.BinaryOnly[0].Path != "sign-admin" {
		t.Fatalf("binary only: %+v", r.Summary.BinaryOnly)
	}
	if len(r.Summary.Copies) != 0 {
		t.Fatalf("copies: %+v", r.Summary.Copies)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"2.10.2", "2.9.0", 1}, {"1.0", "1.0.1", -1}, {"2.4", "2.4", 0}, {"1.0.0-7", "1.0.0-6", 1}}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("%s vs %s = %d", c.a, c.b, got)
		}
	}
}

func TestTestResourcesAreNotCopiesOfMain(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pom.xml", pom("es.x", "web", "1.0", "war"))
	for _, d := range []string{"src/main/resources/spring", "src/test/resources/spring_test"} {
		for _, c := range []string{"a", "b", "c", "d", "e", "f"} {
			write(t, root, d+"/ctx-"+c+".xml", "<beans/>")
		}
	}
	for _, env := range []string{"int", "pre", "pro"} {
		for _, c := range []string{"a", "b", "c", "d", "e", "f"} {
			write(t, root, "web/config/"+env+"/"+c+".yaml", "k: v")
		}
	}
	r, _ := Run(root, Options{})
	if len(r.Summary.Copies) != 0 {
		t.Fatalf("copies: %+v", r.Summary.Copies)
	}
}

func TestSVNBranchesAndBundledJDK(t *testing.T) {
	root := t.TempDir()
	project(t, root, "fuente/trunk", "2.10.2", 8)
	project(t, root, "fuente/branches/FEATURE-109", "2.10.2", 8)
	write(t, root, "fuente/branches/orders_12581/ordersSRV/pom.xml", pom("es.acme", "ordersSRV", "0.1", "jar"))
	write(t, root, "Liberty_Settings/opt/jdk/jdk21.0.5_11/lib/modules", "x")
	write(t, root, "Liberty_Settings/server.xml", "<server/>")
	r, _ := Run(root, Options{})
	got := map[string]bool{}
	for _, c := range r.Summary.Copies {
		got[c.Path] = true
	}
	if !got["fuente/branches/FEATURE-109"] || !got["fuente/branches/orders_12581"] {
		t.Errorf("branches not left out: %+v", r.Summary.Copies)
	}
	rt := false
	for _, a := range r.Summary.NonCode {
		if a.Kind == "runtime" && a.Path == "Liberty_Settings/opt/jdk" {
			rt = true
		}
	}
	if !rt {
		t.Errorf("bundled JDK not reported: %+v", r.Summary.NonCode)
	}
	for _, s := range []string{"fuente/branches", "Liberty_Settings/opt/jdk"} {
		found := false
		for _, k := range r.Skip {
			found = found || k == s
		}
		if !found {
			t.Errorf("%s not skipped: %v", s, r.Skip)
		}
	}
}

func TestRootProjectIsListed(t *testing.T) {
	root := t.TempDir()
	write(t, root, "pom.xml", pom("es.acme.sgpd", "portal_web", "3.2.0", "war"))
	write(t, root, "src/main/java/A.java", "class A {}")
	r, _ := Run(root, Options{})
	if len(r.Summary.Roots) != 1 || r.Summary.Roots[0].Path != "" {
		t.Fatalf("roots: %+v", r.Summary.Roots)
	}
}
