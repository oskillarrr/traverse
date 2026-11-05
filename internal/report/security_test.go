package report

import (
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func TestInboundMTLSStates(t *testing.T) {
	api := model.Contract{Kind: "http", Direction: "exposes", Key: "POST /sign", Transport: "mtls-upstream"}
	cases := []struct {
		f    model.Facts
		want string
	}{
		{model.Facts{Contracts: []model.Contract{{Kind: "http", Direction: "exposes", Key: "GET /x"}}}, "not visible in the code"},
		{model.Facts{Contracts: []model.Contract{api}, Findings: []model.Finding{{RuleID: "TLS-UPSTREAM-CERT-HEADER"}}}, "inferred: 1 interface receives the client certificate"},
		{model.Facts{Contracts: []model.Contract{{Kind: "http", Direction: "exposes", Key: "GET /x"}}, Findings: []model.Finding{{RuleID: "DESC-WEB-MTLS"}}}, "required by the application server"},
		{model.Facts{Contracts: []model.Contract{{Kind: "http", Direction: "calls", Key: "GET /x"}}}, ""},
	}
	for i, c := range cases {
		got, _ := inboundMTLS(&c.f, exposesSomething(&c.f), false)
		if (c.want == "" && got != "") || !strings.Contains(got, c.want) {
			t.Errorf("case %d: %q, want %q", i, got, c.want)
		}
	}
	if transportLabel("es", "mtls-upstream") != "mTLS en intermediario (inferido)" {
		t.Error("label")
	}
}

func exposesSomething(f *model.Facts) bool {
	for _, c := range f.Contracts {
		if c.Direction == "exposes" {
			return true
		}
	}
	return false
}
