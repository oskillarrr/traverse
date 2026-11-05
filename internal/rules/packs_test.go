package rules

import (
	"path/filepath"
	"strings"
	"testing"
)

// Optional rule packs (regional or sector knowledge kept out of the core
// catalog) must load on top of it, pass their own examples and never
// override a core rule by accident.
func TestRulePacks(t *testing.T) {
	packs, _ := filepath.Glob(filepath.Join("..", "..", "rulepacks", "*.json"))
	if len(packs) == 0 {
		t.Skip("no rule packs")
	}
	core, err := Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	coreIDs := map[string]bool{}
	for _, r := range core.Rules {
		coreIDs[r.ID] = true
	}
	for _, p := range packs {
		only, err := Load(false, []string{p})
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		for _, r := range only.Rules {
			if coreIDs[r.ID] {
				t.Errorf("%s: rule %s overrides a core rule", p, r.ID)
			}
		}
		for _, e := range only.SelfTest() {
			t.Errorf("%s: %v", p, e)
		}
		all, err := Load(true, []string{p})
		if err != nil || !strings.Contains(all.Version, "+") {
			t.Errorf("%s: does not stack on the core catalog (%v, %q)", p, err, all.Version)
		}
	}
}
