package report

import (
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// Area views: one portfolio page per group (business area), cut from the
// global portfolio so relationships with other areas stay visible. Systems of
// other areas appear only when they are related to the area.

// groups returns the distinct, non-empty groups of the portfolio.
func groups(p *model.Portfolio) []string {
	set := map[string]bool{}
	for _, s := range p.Systems {
		if s.Group != "" {
			set[s.Group] = true
		}
	}
	return sortedSet(set)
}

// areaPortfolio cuts the global portfolio to one group plus its neighbors.
func areaPortfolio(p *model.Portfolio, group string) (*model.Portfolio, map[string]bool) {
	members := map[string]bool{}
	for _, s := range p.Systems {
		if s.Group == group {
			members[s.Name] = true
		}
	}
	keep := map[string]bool{}
	for m := range members {
		keep[m] = true
	}
	a := *p
	a.Links = nil
	for _, l := range p.Links {
		if members[l.From] || members[l.To] {
			a.Links = append(a.Links, l)
			keep[l.From], keep[l.To] = true, true
		}
	}
	a.Systems = nil
	for _, s := range p.Systems {
		if keep[s.Name] {
			a.Systems = append(a.Systems, s)
		}
	}
	a.SharedTables = nil
	for _, st := range p.SharedTables {
		for _, s := range st.Systems {
			if members[s] {
				a.SharedTables = append(a.SharedTables, st)
				break
			}
		}
	}
	a.LibraryVersions = nil
	for _, lu := range p.LibraryVersions {
		v := map[string][]string{}
		for ver, ss := range lu.Versions {
			for _, s := range ss {
				if members[s] {
					v[ver] = append(v[ver], s)
				}
			}
		}
		if len(v) > 0 {
			lu.Versions = v
			a.LibraryVersions = append(a.LibraryVersions, lu)
		}
	}
	a.External = nil
	for _, e := range p.External {
		if ss := only(e.Systems, members); len(ss) > 0 {
			e.Systems = ss
			a.External = append(a.External, e)
		}
	}
	a.Technologies = nil
	for _, t := range p.Technologies {
		m := map[string]int{}
		for s, n := range t.Systems {
			if members[s] {
				m[s] = n
			}
		}
		if len(m) > 0 {
			t.Systems = m
			a.Technologies = append(a.Technologies, t)
		}
	}
	a.Vulnerabilities = nil
	for _, vu := range p.Vulnerabilities {
		var aff []string
		for _, x := range vu.Affected {
			if members[strings.SplitN(x, "@", 2)[0]] {
				aff = append(aff, x)
			}
		}
		if len(aff) > 0 {
			vu.Affected = aff
			a.Vulnerabilities = append(a.Vulnerabilities, vu)
		}
	}
	sort.Slice(a.Systems, func(i, j int) bool {
		if (a.Systems[i].Group == group) != (a.Systems[j].Group == group) {
			return a.Systems[i].Group == group
		}
		return a.Systems[i].Name < a.Systems[j].Name
	})
	return &a, keep
}

func only(ss []string, set map[string]bool) []string {
	var out []string
	for _, s := range ss {
		if set[s] {
			out = append(out, s)
		}
	}
	return out
}
