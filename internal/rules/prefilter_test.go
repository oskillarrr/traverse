package rules

import (
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/lang"
)

func TestRequiredLiterals(t *testing.T) {
	if l, _ := requiredLiterals(`\b(javax|jakarta)\.ws\.rs\b`); len(l) != 1 || l[0] != ".ws.rs" {
		t.Errorf("got %q", l)
	}
	if l, f := requiredLiterals(`(?i)\bEXEC\s+CICS\b`); len(l) != 1 || !f {
		t.Errorf("got %q fold=%v", l, f)
	}
	if l, _ := requiredLiterals(`\b(?:\d{1,3}\.){3}\d{1,3}\b`); l != nil {
		t.Errorf("no literal set expected, got %q", l)
	}
}

// The prefilter must never hide a real match: every positive example of
// every rule has to pass it.
func TestPrefilterIsSafeForCatalog(t *testing.T) {
	c, err := Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range c.Rules {
		for _, ex := range r.Examples.Match {
			code := lang.Strip(ex, lang.None).Code
			if !r.mayMatch(code, strings.ToLower(code)) {
				t.Errorf("%s: prefilter %q rejects positive example %q", r.ID, r.need, ex)
			}
		}
	}
}
