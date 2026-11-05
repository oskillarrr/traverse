package report

import (
	"regexp"
	"testing"
)

// Every {{t "key"}} used by the templates must exist in both languages.
func TestTemplateKeysTranslated(t *testing.T) {
	re := regexp.MustCompile(`\{\{t "([^"]+)"\}\}`)
	for _, tpl := range []string{repoTemplate, portfolioTemplate} {
		for _, m := range re.FindAllStringSubmatch(tpl, -1) {
			for _, lang := range []string{"en", "es"} {
				if _, ok := labels[lang][m[1]]; !ok {
					t.Errorf("missing %s translation for %q", lang, m[1])
				}
			}
		}
	}
	for _, k := range []string{"lowConfidence", "scopeNote", "portfolio", "landscape"} {
		if _, ok := labels["es"][k]; !ok {
			t.Errorf("missing es %q", k)
		}
	}
}
