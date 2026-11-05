package lite

import (
	"testing"

	"github.com/oskillarrr/traverse/internal/lang"
)

func TestPythonImports(t *testing.T) {
	src := "from fastapi import APIRouter\nfrom . import repo\nfrom .models import (Order,\n  Line)\nimport os, shop.db\n"
	f := Parse("Python", "shop/api/orders.py", lang.Strip(src, lang.Python))
	got := map[string]bool{}
	for _, im := range f.Imports {
		got[im.Spec] = true
	}
	for _, want := range []string{"repo", "models", "models.Order", "os", "shop.db"} {
		if !got[want] {
			t.Errorf("missing import %q in %+v", want, f.Imports)
		}
	}
}
