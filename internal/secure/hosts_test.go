package secure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokensVaultAndReveal(t *testing.T) {
	dir := t.TempDir()
	k, _ := GenerateKey()
	kf := filepath.Join(dir, "team.key")
	os.WriteFile(kf, []byte(k), 0o600)
	s, err := FromKeyFile(kf)
	if err != nil {
		t.Fatal(err)
	}
	a, b := s.Token("API.Internal.Example"), s.Token("api.internal.example")
	if a != b || !strings.HasPrefix(a, "host-") {
		t.Fatalf("tokens must be stable and case-insensitive: %s %s", a, b)
	}
	vb, _ := json.Marshal(s.Vault())
	if strings.Contains(string(vb), "example") {
		t.Fatal("vault must not contain clear text")
	}
	vf := filepath.Join(dir, "hosts.vault.json")
	os.WriteFile(vf, vb, 0o600)
	got, err := s.Reveal(vf)
	if err != nil || got[a] != "api.internal.example" {
		t.Fatalf("reveal: %v %v", got, err)
	}
	other, _ := FromPassphrase("another passphrase 123")
	if _, err := other.Reveal(vf); err == nil {
		t.Fatal("a different key must not open the vault")
	}
	if other.Token("api.internal.example") == a {
		t.Fatal("pseudonyms must depend on the key")
	}
}

func TestShortPassphraseRejected(t *testing.T) {
	if _, err := FromPassphrase("short"); err == nil {
		t.Fatal("expected error")
	}
}
