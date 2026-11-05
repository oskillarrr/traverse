package report

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/portfolio"
	"github.com/oskillarrr/traverse/internal/rules"
	"github.com/oskillarrr/traverse/internal/scan"
)

// Two runs over the same input must produce byte-identical outputs.
func TestReportsAreDeterministic(t *testing.T) {
	cat, err := rules.Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir string) {
		var all []*model.Facts
		for _, fx := range []string{"poc-example", "legacy-es", "polyglot"} {
			f, err := scan.Repo("../scan/testdata/"+fx, fx, "g", scan.Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &scan.Config{}, Now: "fixed", Workers: 4})
			if err != nil {
				t.Fatal(err)
			}
			all = append(all, f)
		}
		p := portfolio.Build(all, model.ToolInfo{Version: "t"}, "fixed")
		if err := Write(dir, all, p, Options{Lang: "es", MaxNodes: 60}); err != nil {
			t.Fatal(err)
		}
	}
	a, b := t.TempDir(), t.TempDir()
	run(a)
	run(b)
	n := 0
	filepath.Walk(a, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(a, p)
		x, _ := os.ReadFile(p)
		y, err := os.ReadFile(filepath.Join(b, rel))
		if err != nil || !bytes.Equal(x, y) {
			t.Errorf("%s differs between runs", rel)
		}
		n++
		return nil
	})
	if n < 40 {
		t.Errorf("only %d files produced", n)
	}
}
