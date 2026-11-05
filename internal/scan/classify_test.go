package scan

import (
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/rules"
)

func TestCounterpartFromCode(t *testing.T) {
	for comp, want := range map[string]string{
		"orders.negocio.payrollWS.ServiciosTurnosImplService": "payroll",
		"es.acme.crm.client.CrmRestClient":                    "crm",
		"orders.negocio.impl.UsuarioServiceImpl":              "",
		"orders.negocio.impl.ClientePayroll":                  "payroll",
		"es.x.integracion.LedgerClient":                       "ledger",
		"x.Y":                                                 "",
	} {
		if got := counterpartFromCode(comp, "orders"); got != want {
			t.Errorf("%s → %q want %q", comp, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	f := &model.Facts{Repo: model.RepoInfo{Name: "orders"}}
	f.Contracts = []model.Contract{
		{Kind: "db-table", Direction: "defines", Key: "LEX.MATERIA"},
		{Kind: "db-table", Direction: "read", Key: "LEX.MATERIA"},
		{Kind: "db-table", Direction: "read", Key: "ORDERS.PERSONAS"},
		{Kind: "db-table", Direction: "write", Key: "AUDIT.ORDERS_PROCESS"},
		{Kind: "db-table", Direction: "read", Key: "SYS.DBA_SYS_PRIVS"},
		{Kind: "platform-module", Direction: "uses", Key: "es.acme.ledger"},
		{Kind: "platform-module", Direction: "uses", Key: "org.apache.cxf"},
		{Kind: "jndi", Direction: "uses", Key: "java:/Mail"},
		{Kind: "http", Direction: "exposes", Key: "GET /orders/ws/consultarPersona"},
		{Kind: "http", Direction: "exposes", Key: "ANY /orders/administracion/personas"},
	}
	// kinds come from the rule data, as the scan copies them into findings
	f.Findings = []model.Finding{{RuleID: "SIGN-ADES", AuthKind: "signature", File: "WEB-INF/jsp/login.jsp"},
		{RuleID: "SIGN-ADES", AuthKind: "signature", File: "js/signDocument.js"},
		{RuleID: "AUTH-CUSTOM-PROVIDER", AuthKind: "user", File: "AcmeAuthenticationProvider.java"},
		{RuleID: "AUTH-WS-SECURITY", AuthKind: "service"},
		{RuleID: "DESC-WEB-MTLS", Category: "Authentication"},
		{RuleID: "CUSTOM-LOGIN", Category: "Authentication"}}
	classify(f)
	want := []string{"", "", "", "schema:audit", "", "module:es.acme.ledger", "", "mail", "", ""}
	for i, w := range want {
		if f.Contracts[i].Counterpart != w {
			t.Errorf("%s: %q want %q", f.Contracts[i].Key, f.Contracts[i].Counterpart, w)
		}
	}
	if f.Contracts[8].Role != "api" || f.Contracts[9].Role != "" {
		t.Errorf("roles: %q %q", f.Contracts[8].Role, f.Contracts[9].Role)
	}
	kinds := []string{"user", "signature", "user", "service", "transport", "user"}
	for i, k := range kinds {
		if f.Findings[i].AuthKind != k {
			t.Errorf("finding %d: %q want %q", i, f.Findings[i].AuthKind, k)
		}
	}
}

func TestCatalogDeclaresAuthKinds(t *testing.T) {
	cat, err := rules.Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"AUTH-OAUTH2": "user", "AUTH-APIKEY": "service", "SIGN-XMLDSIG": "signature", "AUTH-X509": "transport"}
	for _, r := range cat.Rules {
		if k, ok := want[r.ID]; ok {
			if r.AuthKind != k {
				t.Errorf("%s: %q want %q", r.ID, r.AuthKind, k)
			}
			delete(want, r.ID)
		}
		if strings.HasPrefix(r.ID, "GOV-") || strings.HasPrefix(r.ID, "ES-") {
			t.Errorf("country-specific rule %s in the core catalog", r.ID)
		}
	}
	if len(want) > 0 {
		t.Errorf("missing rules: %v", want)
	}
}

func TestUpstreamMTLSMarksEndpoints(t *testing.T) {
	f := &model.Facts{
		Modules: []model.Module{{Path: "api"}, {Path: "web"}},
		Findings: []model.Finding{
			{RuleID: "TLS-UPSTREAM-CERT-HEADER", File: "api/src/main/java/x/SignController.java", SourceSet: "main"},
			{RuleID: "TLS-UPSTREAM-CERT-HEADER", File: "web/src/main/java/x/ClientCertFilter.java", SourceSet: "main"},
		},
		Contracts: []model.Contract{
			{Kind: "http", Direction: "exposes", Key: "POST /sign", File: "api/src/main/java/x/SignController.java"},
			{Kind: "http", Direction: "exposes", Key: "GET /status", File: "api/src/main/java/x/StatusController.java"},
			{Kind: "http", Direction: "exposes", Key: "GET /api/cases", Role: "api", File: "web/src/main/java/x/CasesResource.java"},
			{Kind: "http", Direction: "exposes", Key: "GET /home", Role: "ui", File: "web/src/main/java/x/HomeController.java"},
			{Kind: "http", Direction: "calls", Key: "ANY /other", File: "api/src/main/java/x/SignController.java"},
		},
	}
	markUpstreamMTLS(f)
	want := []string{"mtls-upstream", "", "mtls-upstream", "", ""}
	for i, c := range f.Contracts {
		if c.Transport != want[i] {
			t.Errorf("%s %s: transport %q want %q", c.Direction, c.Key, c.Transport, want[i])
		}
	}
	if f.Contracts[0].Role != "api" || f.Contracts[0].Auth != "client-cert (header)" {
		t.Errorf("controller endpoint: %+v", f.Contracts[0])
	}
}
