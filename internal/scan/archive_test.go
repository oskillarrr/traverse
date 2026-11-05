package scan

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/rules"
)

func zipBytes(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for n, c := range files {
		fw, err := w.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(c)
	}
	w.Close()
	return buf.Bytes()
}

func TestBinaryOnlyModuleIsReadFromTheInside(t *testing.T) {
	root := t.TempDir()
	lib := zipBytes(t, map[string][]byte{
		"META-INF/maven/commons-collections/commons-collections/pom.properties": []byte("groupId=commons-collections\nartifactId=commons-collections\nversion=3.2.1\n"),
	})
	war := zipBytes(t, map[string][]byte{
		"META-INF/maven/com.example.sign/sign-gateway/pom.properties": []byte("groupId=com.example.sign\nartifactId=sign-gateway\nversion=2.4-INT\n"),
		"WEB-INF/web.xml": []byte(`<web-app><login-config><auth-method>CLIENT-CERT</auth-method></login-config>
<security-constraint><user-data-constraint><transport-guarantee>CONFIDENTIAL</transport-guarantee></user-data-constraint></security-constraint></web-app>`),
		"WEB-INF/classes/config.properties":               []byte("signer.url=https://signer.example.es/signerws/services/DSSSignerVerify\n"),
		"WEB-INF/classes/com/example/sign/Servicio.class": []byte{0xCA, 0xFE},
		"WEB-INF/lib/commons-collections-3.2.1.jar":       lib,
	})
	os.MkdirAll(filepath.Join(root, "sign-gateway"), 0o755)
	os.WriteFile(filepath.Join(root, "sign-gateway", "pom.xml"), []byte(`<project><groupId>com.example.sign</groupId><artifactId>sign-gateway</artifactId><version>2.3</version><packaging>war</packaging></project>`), 0o644)
	os.WriteFile(filepath.Join(root, "sign-gateway", "sign-gateway-2.4-INT.war"), war, 0o644)

	cat, _ := rules.Load(true, nil)
	f, err := Repo(root, "signgw", "", Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &Config{}, Now: "t", Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, fl := range f.Files {
		if strings.Contains(fl.Path, "!/") {
			t.Errorf("artifact entry inventoried as a repository file: %s", fl.Path)
		}
	}
	mtls, signer, lib31, mismatch := false, false, false, false
	for _, fd := range f.Findings {
		if fd.RuleID == "DESC-WEB-MTLS" && strings.HasSuffix(fd.File, ".war!/WEB-INF/web.xml") {
			mtls = true
		}
	}
	for _, c := range f.Contracts {
		if c.Kind == "http" && c.Counterpart == "signerws" && strings.Contains(c.File, ".war!/WEB-INF/classes/config.properties") {
			signer = true
		}
	}
	for _, m := range f.Modules {
		for _, d := range m.Deps {
			if d.ID == "maven:commons-collections:commons-collections" && d.Version == "3.2.1" {
				lib31 = true
			}
		}
	}
	for _, d := range f.Diagnostics {
		if d.Kind == "artifact-version-mismatch" && strings.Contains(d.Message, "2.4-INT") && strings.Contains(d.Message, "2.3") {
			mismatch = true
		}
	}
	if !mtls || !signer || !lib31 || !mismatch {
		t.Fatalf("mtls=%v signer=%v lib=%v mismatch=%v\nfindings=%+v\ncontracts=%+v\ndiag=%+v", mtls, signer, lib31, mismatch, f.Findings, f.Contracts, f.Diagnostics)
	}
}

func TestArtifactVersion(t *testing.T) {
	for n, v := range map[string]string{"sign-gateway-2.4-INT.war": "2.4-INT", "app.war": "", "x/signhub-test-services-2.4-INT.war": "2.4-INT"} {
		if got := ArtifactVersion(n); got != v {
			t.Errorf("%s → %q", n, got)
		}
	}
}

func TestArtifactNameVersionDiffersFromMetadata(t *testing.T) {
	root := t.TempDir()
	war := zipBytes(t, map[string][]byte{
		"META-INF/maven/com.example.sign/sign-gateway/pom.properties": []byte("groupId=com.example.sign\nartifactId=sign-gateway\nversion=2.3\n"),
		"META-INF/maven/com.example.sign/sign-gateway/pom.xml":        []byte("<project><groupId>com.example.sign</groupId><artifactId>sign-gateway</artifactId><version>2.3</version></project>"),
		"WEB-INF/web.xml": []byte("<web-app/>"),
	})
	os.MkdirAll(filepath.Join(root, "sign-gateway"), 0o755)
	os.WriteFile(filepath.Join(root, "sign-gateway", "pom.xml"), []byte(`<project><groupId>com.example.sign</groupId><artifactId>sign-gateway</artifactId><version>2.3</version><packaging>war</packaging></project>`), 0o644)
	os.WriteFile(filepath.Join(root, "sign-gateway", "sign-gateway-2.4-INT.war"), war, 0o644)
	cat, _ := rules.Load(true, nil)
	f, err := Repo(root, "signgw", "", Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &Config{}, Now: "t", Workers: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Modules) != 1 {
		t.Errorf("embedded pom.xml must not create a second module: %+v", f.Modules)
	}
	found := false
	for _, d := range f.Diagnostics {
		found = found || d.Kind == "artifact-name-version"
	}
	if !found {
		t.Errorf("missing artifact-name-version: %+v", f.Diagnostics)
	}
}
