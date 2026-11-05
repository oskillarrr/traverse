package report

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// A validation sample: a fixed, stratified selection of the report's claims
// for a person who knows the applications to mark as correct or not. The
// precision of the analysis is then a number, not an impression. The
// selection is deterministic (same facts, same sample) so two reviewers can
// work on the same list.

type claim struct {
	system, kind, text, evidence string
}

// strata and how many claims each contributes.
var sampleQuota = []struct {
	kind string
	n    int
}{
	{"relationship", 5}, {"outbound", 5}, {"authentication", 4}, {"high-risk", 3}, {"structure", 3},
}

func validationSample(all []*model.Facts, p *model.Portfolio, lang string) [][]string {
	es := lang == "es"
	by := map[string][]claim{}
	add := func(c claim) { by[c.kind] = append(by[c.kind], c) }
	for _, l := range p.Links {
		add(claim{l.From + " → " + l.To, "relationship",
			fmt.Sprintf(es2(es, "%s se relaciona con %s por %s (%s; confianza %s)", "%s relates to %s through %s (%s; confidence %s)"),
				l.From, l.To, l.Kind, strings.Join(firstN(l.Keys, 3), ", "), l.Confidence), ""})
	}
	for _, f := range all {
		pv := buildParties(f, p, lang)
		for _, r := range pv.Calls {
			verb := es2(es, "llama a %s (%s; %d endpoints)", "calls %s (%s; %d endpoints)")
			if r.Protocols == "platform-module" || r.Protocols == "jndi" {
				// a server module or resource is used, not called
				verb = es2(es, "usa %s (%s; %d referencias)", "uses %s (%s; %d references)")
			}
			add(claim{f.Repo.Name, "outbound", fmt.Sprintf(verb, r.Counterpart, r.Protocols, r.Endpoints), r.Evidence})
		}
		sec := buildSecurity(f, p, lang)
		for _, m := range sec.Mechanisms {
			add(claim{f.Repo.Name, "authentication", m.KindLabel + ": " + m.Mechanism, m.Evidence})
		}
		if sec.NoUserAuth {
			add(claim{f.Repo.Name, "authentication", es2(es, "login de usuarios no visible en el código", "user login not visible in the code"), ""})
		}
		for _, fd := range f.Findings {
			if fd.Severity == "high" && fd.SourceSet == "main" {
				ev := fd.File
				if len(fd.Lines) > 0 {
					ev = fmt.Sprintf("%s:%d", fd.File, fd.Lines[0])
				}
				add(claim{f.Repo.Name, "high-risk", fd.Title, ev})
			}
		}
		if sv := f.Repo.Survey; sv != nil {
			for _, r := range sv.Roots {
				if r.Main {
					add(claim{f.Repo.Name, "structure", fmt.Sprintf(es2(es, "la línea principal es %s (%s)", "the main line is %s (%s)"),
						dispPath(r.Path), r.Version), r.Path})
				}
			}
			for _, b := range sv.BinaryOnly {
				add(claim{f.Repo.Name, "structure", es2(es, "módulo sin código fuente: ", "module without source code: ") + dispPath(b.Path), b.Path})
			}
		}
	}
	var rows [][]string
	n := 0
	for _, q := range sampleQuota {
		cs := by[q.kind]
		sort.Slice(cs, func(i, j int) bool { return claimHash(cs[i]) < claimHash(cs[j]) })
		// one claim per system first, so every system is represented
		seen := map[string]bool{}
		var pick []claim
		for _, c := range cs {
			if len(pick) < q.n && !seen[c.system] {
				seen[c.system] = true
				pick = append(pick, c)
			}
		}
		for _, c := range cs {
			if len(pick) >= q.n {
				break
			}
			dup := false
			for _, x := range pick {
				dup = dup || x == c
			}
			if !dup {
				pick = append(pick, c)
			}
		}
		for _, c := range pick {
			n++
			rows = append(rows, []string{itoa(n), c.system, sampleKind(es, c.kind), c.text, c.evidence, "", ""})
		}
	}
	return rows
}

func claimHash(c claim) string {
	h := sha256.Sum256([]byte(c.kind + "|" + c.system + "|" + c.text + "|" + c.evidence))
	return hex.EncodeToString(h[:8])
}

func firstN(xs []string, n int) []string {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

func sampleKind(es bool, k string) string {
	switch k {
	case "relationship":
		return es2(es, "relación entre sistemas", "relationship between systems")
	case "outbound":
		return es2(es, "llamada saliente", "outbound call")
	case "authentication":
		return es2(es, "autenticación", "authentication")
	case "high-risk":
		return es2(es, "riesgo alto", "high risk")
	}
	return es2(es, "estructura", "structure")
}

func validationHeader(lang string) []string {
	es := lang == "es"
	return []string{"#", es2(es, "sistema", "system"), es2(es, "tipo", "kind"), es2(es, "afirmación", "claim"), es2(es, "evidencia", "evidence"),
		es2(es, "veredicto (correcto | falso positivo | incompleto | no verificable)", "verdict (correct | false positive | incomplete | not verifiable)"),
		es2(es, "notas", "notes")}
}
