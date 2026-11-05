package report

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/portfolio"
)

func sample() []*model.Facts {
	return []*model.Facts{{
		Repo:    model.RepoInfo{Name: "app-a", Revision: "abc"},
		Modules: []model.Module{{ID: "maven:g:app-a", Name: "app-a", Version: "1.0", Packaging: "war", Deps: []model.Dependency{{ID: "maven:org.x:lib", Version: "2.1", Kind: "declared", File: "pom.xml", Line: 9}, {ID: "npm:@scope/ui", Version: "^1.2.0", Kind: "declared"}}}},
		Contracts: []model.Contract{{Kind: "http", Direction: "exposes", Key: "GET /a/x", Match: "/x"},
			{Kind: "db-table", Direction: "read", Key: "S.T", Match: "T"}},
		Findings: []model.Finding{{ID: "f1", RuleID: "JAVA-LOG4J1", Title: "Log4j 1.x", Severity: "high", File: "src/A B.java", Lines: []int{3, 7}, Count: 2},
			{ID: "f2", RuleID: "JAVA-JPA", Title: "JPA", Severity: "info", File: "x"}},
	}}
}

func TestSARIFStructure(t *testing.T) {
	b, err := SARIF(sample(), model.ToolInfo{Version: "t"})
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil || v["version"] != "2.1.0" {
		t.Fatalf("invalid sarif: %v", err)
	}
	if strings.Contains(string(b), "JAVA-JPA") {
		t.Errorf("info findings must not be SARIF results")
	}
	if !strings.Contains(string(b), `"uri": "src/A%20B.java"`) {
		t.Errorf("artifact URI must be percent-encoded: %s", b)
	}
}

func TestCycloneDXPurls(t *testing.T) {
	b, err := CycloneDX(sample()[0], model.ToolInfo{Version: "t"})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"specVersion": "1.6"`, `"pkg:maven/org.x/lib@2.1"`, `"pkg:npm/%40scope/ui"`, `"traverse:declaredVersion"`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
}

func TestArchiMateWellFormed(t *testing.T) {
	all := sample()
	p := portfolio.Build(all, model.ToolInfo{Version: "t"}, "now")
	b, err := ArchiMate(all, p, "en")
	if err != nil {
		t.Fatal(err)
	}
	d := xml.NewDecoder(strings.NewReader(string(b)))
	for {
		if _, err := d.Token(); err != nil {
			if err.Error() != "EOF" {
				t.Fatalf("not well-formed: %v", err)
			}
			break
		}
	}
	for _, want := range []string{`xsi:type="ApplicationComponent"`, `xsi:type="DataObject"`, `accessType="Read"`, `xsi:type="Diagram"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(string(b), "<properties></properties>") {
		t.Errorf("empty properties element is invalid per XSD")
	}
}

func TestMinConfidenceFilter(t *testing.T) {
	p := &model.Portfolio{Systems: []model.System{{Name: "a"}, {Name: "b"}, {Name: "c"}}, Links: []model.Link{{From: "a", To: "b", Kind: "http", Confidence: "low"}, {From: "a", To: "c", Kind: "jms", Confidence: "high"}}}
	if len(diagramPortfolio(p, "low").Links) != 2 || len(diagramPortfolio(p, "medium").Links) != 1 {
		t.Errorf("filter")
	}
	if !strings.Contains(LandscapeSVG(p, "es"), "lk k-http lo") {
		t.Errorf("low-confidence link must be attenuated")
	}
	if !strings.Contains(MermaidLandscape(p), "low confidence") {
		t.Errorf("mermaid label")
	}
}
