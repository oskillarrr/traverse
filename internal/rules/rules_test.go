package rules

import "testing"

func TestCatalogExamples(t *testing.T) {
	c, err := Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Rules) < 100 {
		t.Fatalf("expected a full catalog, got %d rules", len(c.Rules))
	}
	for _, e := range c.SelfTest() {
		t.Error(e)
	}
}

func TestExcludeDropsOnlyMatchingHits(t *testing.T) {
	r := &Rule{ID: "T", Pattern: `pwd\s*=\s*\S+`, Exclude: `=\s*https?://`}
	if err := r.compile(); err != nil {
		t.Fatal(err)
	}
	got := r.matches("a.pwd=s3cr3t\nb.pwd=https://x/err\n")
	if len(got) != 1 {
		t.Fatalf("%v", got)
	}
	if err := (&Rule{ID: "U", PathPattern: "x", Exclude: "y"}).compile(); err == nil {
		t.Error("exclude without pattern must be rejected")
	}
}
