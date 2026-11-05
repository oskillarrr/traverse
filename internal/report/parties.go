package report

import (
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// partyRow summarizes the relationship with one other application.
type partyRow struct {
	Counterpart string
	Scope       string
	Protocols   string
	Endpoints   int
	Transport   string // plain wins: a plaintext variant must be visible
	Auth        string
	Envs        string
	Evidence    string
}

// partiesView answers the two questions an architect asks first: which
// applications does this system call, and which ones call it.
type partiesView struct {
	Calls      []partyRow          // outbound, grouped by counterpart
	CalledBy   []partyRow          // inbound from identified systems
	SharesWith []partyRow          // shared data (no direction)
	Registered []model.Certificate // non-CA certificates of peers it trusts (e.g. registered clients)
	TrustedCAs []string            // certification authorities it trusts
	Identities []model.Certificate // its own certificates (key entries)
	UnknownAPI int                 // API endpoints with no identified consumer
	UnknownUI  int                 // screens with no identified consumer (users)
	UnknownAny int                 // other exposed interfaces with no identified consumer
}

func buildParties(f *model.Facts, p *model.Portfolio, lang string) partiesView {
	es := lang == "es"
	var v partiesView
	sec := buildSecurity(f, p, lang)
	in, out := es2(es, "entrante", "inbound"), es2(es, "saliente", "outbound")
	type agg struct {
		row    partyRow
		protos map[string]bool
		envs   map[string]bool
		auths  map[string]bool
	}
	groups := map[string]map[string]*agg{in: {}, out: {}, "shared": {}}
	for _, e := range sec.Externals {
		if e.Protocol == "deploys" {
			continue // a deployment relationship, not a call
		}
		dir := e.Direction
		if dir != in && dir != out {
			dir = "shared"
		}
		a := groups[dir][e.Counterpart]
		if a == nil {
			a = &agg{row: partyRow{Counterpart: e.Counterpart, Scope: e.Scope, Evidence: e.Evidence},
				protos: map[string]bool{}, envs: map[string]bool{}, auths: map[string]bool{}}
			groups[dir][e.Counterpart] = a
		}
		a.row.Endpoints++
		a.protos[e.Protocol] = true
		for _, x := range strings.Split(e.Envs, ", ") {
			if x != "" {
				a.envs[x] = true
			}
		}
		if e.Auth != "" {
			a.auths[e.Auth] = true
		}
		switch {
		case e.Transport == "plain":
			a.row.Transport = "plain"
		case a.row.Transport == "" || (a.row.Transport == "tls" && e.Transport == "mtls"):
			a.row.Transport = e.Transport
		}
	}
	flatten := func(m map[string]*agg) []partyRow {
		var rows []partyRow
		for _, a := range m {
			a.row.Protocols = joinKeys(a.protos)
			a.row.Envs = joinKeys(a.envs)
			a.row.Auth = joinKeys(a.auths)
			rows = append(rows, a.row)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Counterpart < rows[j].Counterpart })
		return rows
	}
	v.Calls, v.CalledBy, v.SharesWith = flatten(groups[out]), flatten(groups[in]), flatten(groups["shared"])

	// exposed interfaces nobody in the code base is known to call
	unknown := es2(es, "(consumidor no identificado)", "(no identified consumer)")
	for _, r := range integrations(f, p, lang) {
		if r.dir != in || r.other != unknown {
			continue
		}
		switch r.role {
		case "api":
			v.UnknownAPI++
		case "ui":
			v.UnknownUI++
		default:
			v.UnknownAny++
		}
	}

	cas := map[string]bool{}
	for _, c := range f.Certificates {
		switch {
		case c.Role == "identity":
			v.Identities = append(v.Identities, c)
		case c.CA:
			name := c.Subject
			if name == "" {
				name = c.Issuer
			}
			cas[strings.TrimPrefix(strings.SplitN(name, ",", 2)[0], "CN=")] = true
		default:
			v.Registered = append(v.Registered, c)
		}
	}
	v.TrustedCAs = sortedSet(cas)
	return v
}

func joinKeys(m map[string]bool) string { return strings.Join(sortedSet(m), ", ") }

func sortedSet(m map[string]bool) []string {
	var out []string
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// certLabel renders a certificate without personal data.
func certLabel(es bool, c model.Certificate) string {
	switch {
	case c.Personal:
		return es2(es, "certificado de persona física (datos omitidos)", "natural person certificate (data omitted)")
	case c.Host != "" && c.Subject != "":
		return c.Host + " · " + c.Subject
	case c.Host != "":
		return c.Host
	case c.Subject != "":
		return c.Subject
	}
	return es2(es, "(sin sujeto)", "(no subject)")
}

func certStatus(es bool, s string) string {
	switch s {
	case "expired":
		return es2(es, "CADUCADO", "EXPIRED")
	case "expiring":
		return es2(es, "caduca en < 90 días", "expires in < 90 days")
	}
	return ""
}

// linkConfidence renders a relationship's confidence, saying when the
// interface is implemented by several analyzed systems (the target is then
// decided by run-time configuration, not by the code).
func linkConfidence(es bool, l model.Link) string {
	if l.Candidates > 1 {
		return l.Confidence + " · " + es2(es, itin(l.Candidates)+" sistemas implementan la interfaz", "interface implemented by "+itin(l.Candidates)+" systems")
	}
	return l.Confidence
}
