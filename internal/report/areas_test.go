package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func TestAreaViews(t *testing.T) {
	p := &model.Portfolio{Systems: []model.System{
		{Name: "orders", Group: "INVENTORY"}, {Name: "hrinfo", Group: "INVENTORY"},
		{Name: "signgw", Group: "Integration"}, {Name: "portal_web", Group: "Web portals"},
	}, Links: []model.Link{
		{From: "orders", To: "hrinfo", Kind: "shared-db"},
		{From: "portal_web", To: "signgw", Kind: "http"},
	}}
	ap, keep := areaPortfolio(p, "Integration")
	if len(ap.Systems) != 2 || !keep["portal_web"] || keep["orders"] || len(ap.Links) != 1 {
		t.Fatalf("area cut: %+v keep=%v", ap, keep)
	}
	if ap.Systems[0].Name != "signgw" {
		t.Errorf("own systems first: %+v", ap.Systems)
	}
	var all []*model.Facts
	for _, s := range p.Systems {
		all = append(all, &model.Facts{Repo: model.RepoInfo{Name: s.Name, Group: s.Group}})
	}
	dir := t.TempDir()
	if err := Write(dir, all, p, Options{Lang: "es"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "areas", "integration", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `href="../../systems/signgw/report.html"`) || !strings.Contains(string(b), "../../index.html") {
		t.Error("area page links must point to the output root")
	}
	idx, _ := os.ReadFile(filepath.Join(dir, "index.html"))
	if !strings.Contains(string(idx), `href="areas/inventory/index.html"`) {
		t.Error("global page must link the area views")
	}
	md, _ := os.ReadFile(filepath.Join(dir, "areas", "inventory", "portfolio.md"))
	if !strings.Contains(string(md), "# Portfolio de aplicaciones — INVENTORY") || !strings.Contains(string(md), "(../../systems/orders/arc42.md)") {
		t.Error("area markdown")
	}
}
