package scan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oskillarrr/traverse/internal/rules"
)

func TestConfigRulesAreRelativeToTheFile(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "traverse.json")
	abs := filepath.Join(dir, "elsewhere", "x.json")
	b, _ := json.Marshal(map[string][]string{"rules": {"packs/acme.json", abs}})
	os.WriteFile(cfg, b, 0o644)
	c, err := LoadConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if c.Rules[0] != filepath.Join(dir, "packs", "acme.json") || c.Rules[1] != abs {
		t.Errorf("rules: %v", c.Rules)
	}
}

func TestRegionalPackIsOptIn(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "src"), 0o755)
	os.WriteFile(filepath.Join(root, "src", "Signer.java"), []byte("package x;\nimport es.gob.afirma.core.signers.AOSigner;\n"+
		"import eu.europa.esig.dss.xades.signature.XAdESService;\npublic class Signer {}\n"), 0o644)
	kinds := func(extra []string) map[string]string {
		cat, err := rules.Load(true, extra)
		if err != nil {
			t.Fatal(err)
		}
		f, err := Repo(root, "app", "", Options{MaxFileSize: 1 << 20, Catalog: cat, Config: &Config{}, Now: "t", Workers: 1})
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		for _, fd := range f.Findings {
			m[fd.RuleID] = fd.AuthKind
		}
		return m
	}
	core := kinds(nil)
	if core["SIGN-ADES"] != "signature" {
		t.Errorf("core: %v", core)
	}
	if _, ok := core["ES-AFIRMA"]; ok {
		t.Errorf("regional rule without the pack: %v", core)
	}
	withPack := kinds([]string{filepath.Join("..", "..", "rulepacks", "es-public-sector.json")})
	if withPack["ES-AFIRMA"] != "signature" || withPack["SIGN-ADES"] != "signature" {
		t.Errorf("with pack: %v", withPack)
	}
}
