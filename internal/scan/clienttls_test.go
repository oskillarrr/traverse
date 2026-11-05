package scan

import (
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func TestMarkClientTLS(t *testing.T) {
	f := &model.Facts{
		Findings: []model.Finding{
			{RuleID: "TLS-CLIENT-IDENTITY", File: "A.java"},
			{RuleID: "AUTH-BASIC-OUTBOUND", File: "B.java"},
		},
		Contracts: []model.Contract{
			{Kind: "http", Direction: "calls", Transport: "tls", File: "A.java"},
			{Kind: "http", Direction: "calls", Transport: "plain", File: "A.java"},
			{Kind: "http", Direction: "calls", Transport: "tls", File: "B.java"},
			{Kind: "http", Direction: "exposes", File: "A.java"},
		},
	}
	markClientTLS(f)
	c := f.Contracts
	if c[0].Transport != "mtls" || c[0].Auth != "client-cert" {
		t.Errorf("tls call with client keystore must become mtls: %+v", c[0])
	}
	if c[1].Transport != "plain" {
		t.Errorf("a plaintext call must not be upgraded: %+v", c[1])
	}
	if c[2].Auth != "basic" || c[2].Transport != "tls" {
		t.Errorf("basic auth client: %+v", c[2])
	}
	if c[3].Transport != "" {
		t.Errorf("inbound contracts untouched: %+v", c[3])
	}
}
