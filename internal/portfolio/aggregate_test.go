package portfolio

import (
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func TestContractMatchPrefersLiteral(t *testing.T) {
	call := model.Contract{Kind: "http", Direction: "calls", Key: "ANY /ctx/api/messages/send", Match: "/ctx/api/messages/send"}
	lit := model.Contract{Kind: "http", Key: "POST /ctx/api/messages/send", Match: "/api/messages/send"}
	vr := model.Contract{Kind: "http", Key: "GET /ctx/api/messages/{id}", Match: "/api/messages/{}"}
	if contractMatch(call, lit) != 0 || contractMatch(call, vr) != 1 { // literal endpoint is more specific
		t.Errorf("scores: %d %d", contractMatch(call, lit), contractMatch(call, vr))
	}
	generic := model.Contract{Kind: "http", Key: "GET /{id}", Match: "/{}"}
	if contractMatch(call, generic) != -1 {
		t.Errorf("generic paths must not match")
	}
}

func TestBuildLinks(t *testing.T) {
	a := &model.Facts{Repo: model.RepoInfo{Name: "a"},
		Modules:   []model.Module{{ID: "maven:g:a", Deps: []model.Dependency{{ID: "maven:g:lib", Version: "1"}}}},
		Contracts: []model.Contract{{Kind: "jms", Direction: "uses", Key: "q", Match: "q"}, {Kind: "db-table", Direction: "write", Key: "T", Match: "T"}}}
	b := &model.Facts{Repo: model.RepoInfo{Name: "b"},
		Modules:   []model.Module{{ID: "maven:g:lib", Name: "lib"}},
		Contracts: []model.Contract{{Kind: "jms", Direction: "subscribes", Key: "q", Match: "q"}, {Kind: "db-table", Direction: "read", Key: "S.T", Match: "T"}}}
	p := Build([]*model.Facts{a, b}, model.ToolInfo{}, "now")
	kinds := map[string]bool{}
	for _, l := range p.Links {
		kinds[l.From+">"+l.To+":"+l.Kind] = true
	}
	for _, want := range []string{"a>b:library", "a>b:jms", "a>b:shared-db"} {
		if !kinds[want] {
			t.Errorf("missing link %s in %v", want, kinds)
		}
	}
	if len(p.SharedTables) != 1 {
		t.Errorf("shared tables: %+v", p.SharedTables)
	}
}

func TestConsumerVariableIsWeak(t *testing.T) {
	call := model.Contract{Kind: "http", Direction: "calls", Key: "ANY /owners/{}", Match: "/owners/{}"}
	lit := model.Contract{Kind: "http", Key: "GET /app/owners/find", Match: "/owners/find"}
	if s := contractMatch(call, lit); s < weakScore {
		t.Errorf("consumer variable vs provider literal must be weak, got %d", s)
	}
	concrete := model.Contract{Kind: "http", Direction: "calls", Key: "ANY /cases/api/x/42", Match: "/cases/api/x/42"}
	param := model.Contract{Kind: "http", Key: "GET /cases/api/x/{id}", Match: "/x/{}"}
	if s := contractMatch(concrete, param); s != 1 {
		t.Errorf("concrete value vs path parameter is a normal match, got %d", s)
	}
}

func TestConfiguredBaseURLLinksToProvider(t *testing.T) {
	a := &model.Facts{Repo: model.RepoInfo{Name: "orders"}, Contracts: []model.Contract{
		{Kind: "http", Direction: "exposes", Key: "GET /orders/ws/consultarPersona", Match: "/ws/consultarPersona"},
		{Kind: "http", Direction: "exposes", Key: "GET /orders/ws/consultarUnidades", Match: "/ws/consultarUnidades"},
	}}
	b := &model.Facts{Repo: model.RepoInfo{Name: "directorio"}, Contracts: []model.Contract{
		{Kind: "http", Direction: "calls", Key: "ANY /orders/ws", Match: "/orders/ws", Detail: "configuration: orders.ws.url"},
	}}
	p := Build([]*model.Facts{a, b}, model.ToolInfo{}, "now")
	if len(p.Links) != 1 || p.Links[0].From != "directorio" || p.Links[0].To != "orders" {
		t.Fatalf("links: %+v", p.Links)
	}
}

func TestVersionedContextRootLinks(t *testing.T) {
	signgw := &model.Facts{Repo: model.RepoInfo{Name: "signgw"}, Contracts: []model.Contract{
		{Kind: "http", Direction: "exposes", Key: "ANY /sign-gateway-2.3/signService", Match: "/signService"},
	}}
	portal := &model.Facts{Repo: model.RepoInfo{Name: "portal_web"}, Contracts: []model.Contract{
		{Kind: "http", Direction: "calls", Key: "ANY /sign-gateway/signService", Match: "/sign-gateway/signService", Detail: "configuration: signUrl"},
	}}
	p := Build([]*model.Facts{signgw, portal}, model.ToolInfo{}, "now")
	if len(p.Links) != 1 || p.Links[0].From != "portal_web" || p.Links[0].To != "signgw" {
		t.Fatalf("links: %+v", p.Links)
	}
	if got := stripVersionedRoot("/sign-gateway-2.3/signService"); got != "/sign-gateway/signService" {
		t.Errorf("strip: %s", got)
	}
	if got := stripVersionedRoot("/api/v2/x"); got != "/api/v2/x" {
		t.Errorf("must not strip: %s", got)
	}
}

func TestInterfaceWithSeveralProviders(t *testing.T) {
	notifier := &model.Facts{Repo: model.RepoInfo{Name: "notifier"}, Contracts: []model.Contract{
		{Kind: "soap", Direction: "calls", Key: "CallbackWSService", Match: "CallbackWSService", Confidence: "high"},
	}}
	a := &model.Facts{Repo: model.RepoInfo{Name: "callback"}, Contracts: []model.Contract{
		{Kind: "soap", Direction: "exposes", Key: "callbackWS", Match: "callbackWS", Confidence: "high"},
	}}
	b := &model.Facts{Repo: model.RepoInfo{Name: "portal"}, Contracts: []model.Contract{
		{Kind: "soap", Direction: "exposes", Key: "callbackWS", Match: "callbackWS", Confidence: "high"},
	}}
	p := Build([]*model.Facts{notifier, a, b}, model.ToolInfo{}, "now")
	if len(p.Links) != 2 {
		t.Fatalf("links: %+v", p.Links)
	}
	for _, l := range p.Links {
		if l.Candidates != 2 || l.Confidence != "medium" {
			t.Errorf("%+v", l)
		}
	}
}
