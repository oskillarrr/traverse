// Package portfolio links the facts of several repositories into a landscape:
// shared libraries, matching provided/consumed contracts and shared data.
package portfolio

import (
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

type linkKey struct{ from, to, kind string }

// Build aggregates facts. Systems are identified by repository name.
func Build(all []*model.Facts, tool model.ToolInfo, now string) *model.Portfolio {
	p := &model.Portfolio{SchemaVersion: model.SchemaVersion, Tool: tool, GeneratedAt: now}
	links := map[linkKey]*model.Link{}
	addLink := func(from, to, kind, key, conf string) {
		if from == to {
			return
		}
		k := linkKey{from, to, kind}
		l, ok := links[k]
		if !ok {
			l = &model.Link{From: from, To: to, Kind: kind, Confidence: conf}
			links[k] = l
		}
		l.Count++
		if len(l.Keys) < 25 && !contains(l.Keys, key) {
			l.Keys = append(l.Keys, key)
		}
		if rank(conf) > rank(l.Confidence) {
			l.Confidence = conf
		}
	}

	// ---- systems ----
	for _, f := range all {
		s := model.System{Name: f.Repo.Name, Group: f.Repo.Group, VCS: f.Repo.VCS, Revision: f.Repo.Revision,
			Modules: len(f.Modules), Risks: map[string]int{}, Cycles: len(f.Cycles), LayerIssues: len(f.LayerIssues)}
		for _, l := range f.Languages {
			s.Files += l.Files
			if l.Category == "code" || l.Category == "markup" {
				s.Code += l.Code
				if len(s.Languages) < 4 && l.Code > 0 {
					s.Languages = append(s.Languages, l.Language)
				}
			}
		}
		for _, c := range f.Components {
			if c.SourceSet == "main" {
				s.Components++
			}
		}
		tables := map[string]bool{}
		for _, c := range f.Contracts {
			switch {
			case c.Kind == "db-table":
				tables[c.Match] = true
			case c.Direction == "exposes":
				s.Exposes++
			case c.Direction == "calls":
				s.Calls++
			}
		}
		s.Tables = len(tables)
		rt := map[string]bool{}
		for _, m := range f.Modules {
			if m.Runtime != "" && !rt[m.Runtime] {
				rt[m.Runtime] = true
				s.Runtimes = append(s.Runtimes, m.Runtime)
			}
		}
		sort.Strings(s.Runtimes)
		for _, d := range f.Deployables {
			s.Deployables = append(s.Deployables, strings.ToUpper(d.Kind)+" "+shortModule(d.Module))
		}
		for _, fd := range f.Findings {
			if fd.SourceSet == "main" && fd.Severity != "info" {
				s.Risks[fd.Severity]++
			}
		}
		for _, v := range f.Vulnerabilities {
			if s.Vulns == nil {
				s.Vulns = map[string]int{}
			}
			s.Vulns[v.Severity]++
		}
		p.Systems = append(p.Systems, s)
	}

	// ---- libraries ----
	producer := map[string]string{}   // module id -> system
	byArtifact := map[string]string{} // artifact/name -> system (for vendored jars)
	for _, f := range all {
		for _, m := range f.Modules {
			if strings.HasPrefix(m.ID, "repo:") || strings.Contains(m.ID, "-requirements:") || strings.HasPrefix(m.ID, "nuget-packages:") {
				continue
			}
			producer[m.ID] = f.Repo.Name
			if m.Name != "" {
				byArtifact[strings.ToLower(m.Name)] = f.Repo.Name
			}
		}
	}
	libs := map[string]*model.LibraryUsage{}
	for _, f := range all {
		for _, m := range f.Modules {
			for _, d := range m.Deps {
				if d.Managed || d.Kind == "project-ref" {
					continue
				}
				if sys, ok := producer[d.ID]; ok && sys != f.Repo.Name {
					addLink(f.Repo.Name, sys, "library", d.ID, "high")
				} else if d.Kind == "vendored-jar" {
					if sys, ok := byArtifact[strings.TrimPrefix(d.ID, "jar:")]; ok && sys != f.Repo.Name {
						addLink(f.Repo.Name, sys, "library", d.ID, "medium")
					}
				}
				lu := libs[d.ID]
				if lu == nil {
					lu = &model.LibraryUsage{Library: d.ID, Versions: map[string][]string{}}
					libs[d.ID] = lu
				}
				_, lu.Internal = producer[d.ID]
				v := d.Version
				if d.Locked != "" {
					v = d.Locked
				}
				if v == "" {
					v = "(unspecified)"
				}
				if !contains(lu.Versions[v], f.Repo.Name) {
					lu.Versions[v] = append(lu.Versions[v], f.Repo.Name)
				}
			}
		}
	}
	for _, lu := range libs {
		systems := map[string]bool{}
		for _, ss := range lu.Versions {
			for _, s := range ss {
				systems[s] = true
			}
		}
		if len(systems) >= 2 && (len(lu.Versions) >= 2 || lu.Internal) {
			p.LibraryVersions = append(p.LibraryVersions, *lu)
		}
	}
	sort.Slice(p.LibraryVersions, func(i, j int) bool { return p.LibraryVersions[i].Library < p.LibraryVersions[j].Library })

	// ---- contracts ----
	type side struct {
		sys string
		c   model.Contract
	}
	var exposes, calls, subs, pubs []side
	tableUse := map[string]map[string]map[string]bool{}    // table -> system -> ops
	tableSchema := map[string]map[string]map[string]bool{} // table -> system -> schemas
	dsUse := map[string]map[string]bool{}
	dataUse := map[string]map[string]bool{} // mainframe dataset -> systems
	for _, f := range all {
		for _, c := range f.Contracts {
			s := side{f.Repo.Name, c}
			switch {
			case c.Kind == "db-table":
				if tableUse[c.Match] == nil {
					tableUse[c.Match] = map[string]map[string]bool{}
				}
				if tableUse[c.Match][s.sys] == nil {
					tableUse[c.Match][s.sys] = map[string]bool{}
				}
				tableUse[c.Match][s.sys][c.Direction] = true
				if i := strings.LastIndexByte(c.Key, '.'); i > 0 {
					if tableSchema[c.Match] == nil {
						tableSchema[c.Match] = map[string]map[string]bool{}
					}
					if tableSchema[c.Match][s.sys] == nil {
						tableSchema[c.Match][s.sys] = map[string]bool{}
					}
					tableSchema[c.Match][s.sys][c.Key[:i]] = true
				}
			case c.Kind == "dataset":
				if dataUse[c.Match] == nil {
					dataUse[c.Match] = map[string]bool{}
				}
				dataUse[c.Match][s.sys] = true
			case c.Kind == "cics-file" || c.Kind == "cics-queue":
				// region-local resources: not linked across systems
			case c.Kind == "jndi" && c.Direction == "uses" && (strings.Contains(c.Match, "jdbc/") || strings.Contains(c.Match, "datasources/") || strings.Contains(c.Detail, "datasource")):
				if dsUse[c.Match] == nil {
					dsUse[c.Match] = map[string]bool{}
				}
				dsUse[c.Match][s.sys] = true
			case c.Direction == "exposes":
				exposes = append(exposes, s)
			case c.Direction == "calls" || (c.Kind == "soap" && c.Direction == "defines"):
				calls = append(calls, s)
			case c.Direction == "subscribes":
				subs = append(subs, s)
			case (c.Kind == "jms" || c.Kind == "kafka") && (c.Direction == "publishes" || c.Direction == "uses"):
				pubs = append(pubs, s)
			}
		}
	}
	external := map[string]map[string]bool{}
	addExternal := func(kind, key, host, sys string) {
		k := kind + "\x00" + key + "\x00" + host
		if external[k] == nil {
			external[k] = map[string]bool{}
		}
		external[k][sys] = true
	}
	for _, c := range calls {
		// A call that matches an endpoint of its own system is internal.
		internal := false
		for _, e := range exposes {
			if e.sys == c.sys && matchWith(c.c, e.c, 1) >= 0 {
				internal = true
				break
			}
		}
		if internal {
			continue
		}
		// keep only the most specific matches (fewest path variables used)
		best := -1
		var hits []side
		for _, e := range exposes {
			if e.sys == c.sys {
				continue
			}
			score := contractMatch(c.c, e.c)
			if score < 0 {
				continue
			}
			if best < 0 || score < best {
				best, hits = score, nil
			}
			if score == best {
				hits = append(hits, e)
			}
		}
		providers := map[string]bool{}
		for _, e := range hits {
			providers[e.sys] = true
		}
		for _, e := range hits {
			conf := c.c.Confidence
			if c.c.Direction == "defines" || best >= weakScore {
				conf = "low" // only a variable in the call matched a literal endpoint
			}
			if len(providers) > 1 && conf == "high" {
				conf = "medium" // several systems implement it: the target is chosen at run time
			}
			addLink(c.sys, e.sys, c.c.Kind, e.c.Key, conf)
			if l := links[linkKey{c.sys, e.sys, c.c.Kind}]; l != nil && len(providers) > l.Candidates && len(providers) > 1 {
				l.Candidates = len(providers)
			}
		}
		if len(hits) == 0 && c.c.Kind == "http" && strings.HasPrefix(c.c.Detail, "configuration") && model.LiteralSegments(c.c.Match) >= 1 {
			// a base URL declared in configuration: the code appends the
			// resource, so a system exposing paths under it is the target
			prefix := strings.ToLower(strings.TrimSuffix(c.c.Match, "/")) + "/"
			linked := map[string]bool{}
			for _, e := range exposes {
				if e.sys == c.sys || linked[e.sys] || e.c.Kind != "http" {
					continue
				}
				full := strings.ToLower(model.NormalizePath(pathOf(e.c.Key)))
				if strings.HasPrefix(full+"/", prefix) || strings.HasPrefix(strings.ToLower(e.c.Match)+"/", prefix) {
					linked[e.sys] = true
					addLink(c.sys, e.sys, "http", c.c.Match+"/*", "medium")
				}
			}
			if len(linked) > 0 {
				continue
			}
		}
		if len(hits) == 0 && c.c.Direction == "calls" {
			addExternal(c.c.Kind, c.c.Key, c.c.Host, c.sys)
		}
	}
	for _, pb := range pubs {
		matched := false
		for _, sb := range subs {
			if sb.sys != pb.sys && sb.c.Kind == pb.c.Kind && sb.c.Match == pb.c.Match && pb.c.Match != "" {
				addLink(pb.sys, sb.sys, pb.c.Kind, pb.c.Key, "high")
				matched = true
			}
		}
		if !matched {
			for _, other := range pubs {
				if other.sys < pb.sys && other.c.Kind == pb.c.Kind && other.c.Match == pb.c.Match {
					addLink(other.sys, pb.sys, pb.c.Kind, pb.c.Key+" (shared destination)", "low")
					matched = true
				}
			}
		}
		if !matched {
			addExternal(pb.c.Kind, pb.c.Key, "", pb.sys)
		}
	}
	for t, bySys := range tableUse {
		if len(bySys) < 2 {
			continue
		}
		st := model.SharedTable{Table: t, Basis: "name-only"}
		for sys, ops := range bySys {
			st.Systems = append(st.Systems, sys)
			var o []string
			for op := range ops {
				o = append(o, op)
			}
			sort.Strings(o)
			st.Access = append(st.Access, sys+":"+strings.Join(o, "/"))
		}
		sort.Strings(st.Systems)
		sort.Strings(st.Access)
		// Same table name alone is weak evidence (USERS exists everywhere):
		// require the same schema qualifier or a shared datasource name.
		conf := "low"
		qualified := false
		for i := 0; i < len(st.Systems) && conf == "low"; i++ {
			for j := i + 1; j < len(st.Systems); j++ {
				a, b := st.Systems[i], st.Systems[j]
				for sc := range tableSchema[t][a] {
					if tableSchema[t][b][sc] {
						conf, st.Basis, qualified = "medium", "schema", true
					}
				}
				if conf == "low" && sharesDatasource(dsUse, a, b) {
					conf, st.Basis = "medium", "datasource"
				}
			}
		}
		if genericTables[strings.ToUpper(t)] && !qualified {
			continue // a generic name in several systems is not evidence of shared data
		}
		p.SharedTables = append(p.SharedTables, st)
		for i := 0; i < len(st.Systems); i++ {
			for j := i + 1; j < len(st.Systems); j++ {
				addLink(st.Systems[i], st.Systems[j], "shared-db", t, conf)
			}
		}
	}
	sort.Slice(p.SharedTables, func(i, j int) bool {
		if len(p.SharedTables[i].Systems) != len(p.SharedTables[j].Systems) {
			return len(p.SharedTables[i].Systems) > len(p.SharedTables[j].Systems)
		}
		return p.SharedTables[i].Table < p.SharedTables[j].Table
	})
	for dsn, bySys := range dataUse {
		var ss []string
		for s := range bySys {
			ss = append(ss, s)
		}
		sort.Strings(ss)
		for i := 0; i < len(ss); i++ {
			for j := i + 1; j < len(ss); j++ {
				addLink(ss[i], ss[j], "shared-dataset", dsn, "medium") // fully qualified names
			}
		}
	}
	for ds, bySys := range dsUse {
		var ss []string
		for s := range bySys {
			ss = append(ss, s)
		}
		sort.Strings(ss)
		for i := 0; i < len(ss); i++ {
			for j := i + 1; j < len(ss); j++ {
				addLink(ss[i], ss[j], "shared-datasource", ds, "low")
			}
		}
	}
	for k, ss := range external {
		parts := strings.SplitN(k, "\x00", 3)
		e := model.ExternalNeed{Kind: parts[0], Key: parts[1], Host: parts[2]}
		for s := range ss {
			e.Systems = append(e.Systems, s)
		}
		sort.Strings(e.Systems)
		p.External = append(p.External, e)
	}
	sort.Slice(p.External, func(i, j int) bool {
		if p.External[i].Kind != p.External[j].Kind {
			return p.External[i].Kind < p.External[j].Kind
		}
		return p.External[i].Key < p.External[j].Key
	})

	// ---- deployment: a repository that deploys images built from another
	// system (e.g. a GitOps or platform repository) ----
	imageOwner := map[string]string{}
	for _, f := range all {
		imageOwner[strings.ToLower(f.Repo.Name)] = f.Repo.Name
		for _, m := range f.Modules {
			if m.Name != "" {
				imageOwner[strings.ToLower(m.Name)] = f.Repo.Name
			}
		}
	}
	for _, f := range all {
		for _, n := range f.Deployment.Nodes {
			if n.Module != "" {
				continue // built from this same repository
			}
			for _, img := range n.Images {
				name := strings.ToLower(imageName(img))
				if owner, ok := imageOwner[name]; ok && owner != f.Repo.Name {
					addLink(f.Repo.Name, owner, "deploys", n.Platform+" "+n.Kind+" "+n.Name+" ← "+img, "medium")
				}
			}
		}
	}

	// ---- technologies ----
	tech := map[string]*model.TechUsage{}
	for _, f := range all {
		for _, fd := range f.Findings {
			if fd.SourceSet != "main" || fd.Technology == "" || fd.Category == "Security" {
				continue
			}
			t := tech[fd.Technology]
			if t == nil {
				t = &model.TechUsage{Technology: fd.Technology, Category: fd.Category, Severity: fd.Severity, Systems: map[string]int{}}
				tech[fd.Technology] = t
			}
			if rankSev(fd.Severity) > rankSev(t.Severity) {
				t.Severity = fd.Severity
			}
			t.Systems[f.Repo.Name]++
		}
	}
	for _, t := range tech {
		p.Technologies = append(p.Technologies, *t)
	}
	sort.Slice(p.Technologies, func(i, j int) bool {
		a, b := p.Technologies[i], p.Technologies[j]
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Technology < b.Technology
	})

	for _, l := range links {
		sort.Strings(l.Keys)
		p.Links = append(p.Links, *l)
	}
	sort.Slice(p.Links, func(i, j int) bool {
		a, b := p.Links[i], p.Links[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Kind < b.Kind
	})
	p.Vulnerabilities, p.VulnDB, p.VulnAsOf = vulnerabilities(all)
	sort.Slice(p.Systems, func(i, j int) bool {
		if p.Systems[i].Group != p.Systems[j].Group {
			return p.Systems[i].Group < p.Systems[j].Group
		}
		return p.Systems[i].Name < p.Systems[j].Name
	})
	return p
}

// contractMatch compares a consumed contract with an exposed one and returns
// -1 when they do not match, otherwise the number of path variables needed
// (0 = literal match). Lower is more specific.
func contractMatch(c, e model.Contract) int { return matchWith(c, e, 2) }

// matchWith compares contracts requiring minLit literal path segments on the
// exposed side. Internal-call detection uses 1: any plausible own endpoint wins.
func matchWith(c, e model.Contract, minLit int) int {
	if c.Kind != e.Kind {
		return -1
	}
	switch c.Kind {
	case "http":
		cv, ev := verb(c.Key), verb(e.Key)
		if cv != "ANY" && ev != "ANY" && cv != ev {
			return -1
		}
		// Prefer the full exposed path (context root + application path + resource)
		// and fall back to the resource path; both need two literal segments to
		// avoid linking on generic paths such as /api/{id}.
		if full := model.NormalizePath(pathOf(e.Key)); model.LiteralSegments(full) >= minLit {
			if w := pathSuffix(c.Match, full); w >= 0 {
				return w
			}
			// a context root inferred from the Maven artifact name carries the
			// version (/sign-gateway-2.3); deployments usually drop it
			if alt := stripVersionedRoot(full); alt != full {
				if w := pathSuffix(c.Match, alt); w >= 0 {
					return w
				}
			}
		}
		if e.Match == "" || e.Match == "/" || model.LiteralSegments(e.Match) < minLit {
			return -1
		}
		return pathSuffix(c.Match, e.Match)
	case "soap":
		if c.Match != "" && soapName(c.Match) == soapName(e.Match) {
			return 0
		}
		return -1
	default:
		if c.Match != "" && strings.EqualFold(c.Match, e.Match) {
			return 0
		}
		return -1
	}
}

var reVersionedRoot = regexp.MustCompile(`^/([^/]+?)-v?\d+(?:\.\d+)+(?:[-.][A-Za-z0-9]+)*(/|$)`)

// stripVersionedRoot turns /sign-gateway-2.3/signService into /sign-gateway/signService.
func stripVersionedRoot(p string) string {
	return reVersionedRoot.ReplaceAllString(p, "/$1$2")
}

// soapName strips the suffixes generated clients and WSDLs add to the same
// service (NotificationCallbackWSService ≡ notificationCallbackWS).
func soapName(s string) string {
	s = strings.ToLower(s)
	for changed := true; changed; {
		changed = false
		for _, suf := range []string{"service", "impl", "port", "endpoint", "binding", "soap"} {
			if len(s) > len(suf)+2 && strings.HasSuffix(s, suf) {
				s, changed = strings.TrimSuffix(s, suf), true
			}
		}
	}
	return s
}

// genericTables are names every application has: an unqualified match on
// them says nothing about shared data.
var genericTables = map[string]bool{"ROL": true, "ROLES": true, "USUARIO": true, "USUARIOS": true, "USER": true, "USERS": true,
	"AUDITORIA": true, "AUDITORIAS": true, "AUDIT": true, "CONFIGURACION": true, "CONFIG": true, "PARAMETRO": true,
	"PARAMETROS": true, "PARAMETERS": true, "LOG": true, "LOGS": true, "ERROR": true, "ERRORES": true, "PERFIL": true,
	"PERFILES": true, "PERMISO": true, "PERMISOS": true, "MENU": true, "MENUS": true, "SESION": true, "SESIONES": true,
	"PROPIEDADES": true, "VARIABLES": true, "VARIABLE": true, "IDIOMA": true, "IDIOMAS": true, "DUAL": true,
	"PROVINCIA": true, "PROVINCIAS": true, "MUNICIPIO": true, "MUNICIPIOS": true, "PAIS": true, "PAISES": true,
	"FETCH": true, "TEMP": true, "TMP": true, "SEQUENCE": true, "HIBERNATE_SEQUENCE": true, "DATABASECHANGELOG": true,
	"DATABASECHANGELOGLOCK": true, "FLYWAY_SCHEMA_HISTORY": true, "AUTHORITIES": true, "GROUPS": true, "TRAZA": true, "TRAZAS": true}

func imageName(image string) string {
	img := strings.SplitN(image, "@", 2)[0]
	if i := strings.LastIndexByte(img, '/'); i >= 0 {
		img = img[i+1:]
	}
	if i := strings.IndexByte(img, ':'); i >= 0 {
		img = img[:i]
	}
	return img
}

func sharesDatasource(ds map[string]map[string]bool, a, b string) bool {
	for _, users := range ds {
		if users[a] && users[b] {
			return true
		}
	}
	return false
}

func pathOf(key string) string {
	if i := strings.IndexByte(key, ' '); i > 0 {
		return key[i+1:]
	}
	return key
}

func verb(key string) string {
	if i := strings.IndexByte(key, ' '); i > 0 {
		return key[:i]
	}
	return "ANY"
}

// pathSuffix returns -1 if path p does not end with the segments of s,
// otherwise how many segments matched only through a variable or wildcard.
const weakScore = 100

func pathSuffix(p, s string) int {
	ps := strings.Split(strings.Trim(p, "/"), "/")
	ss := strings.Split(strings.Trim(s, "/"), "/")
	if len(ss) > len(ps) {
		return -1
	}
	// Score: provider variables/wildcards count 1 (less specific endpoint);
	// a consumer-side variable matching a provider literal counts weakScore
	// (the call itself does not say which endpoint it targets).
	off, vars := len(ps)-len(ss), 0
	for i, seg := range ss {
		q := ps[off+i]
		if q == "{}" && seg != "{}" && !strings.Contains(seg, "*") {
			vars += weakScore
			continue
		}
		if seg == "{}" || q == "{}" || strings.Contains(seg, "*") {
			vars++
			continue
		}
		if !strings.EqualFold(seg, q) {
			return -1
		}
	}
	return vars
}

func shortModule(id string) string {
	if i := strings.LastIndexByte(id, ':'); i >= 0 {
		return id[i+1:]
	}
	return id
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func rank(c string) int {
	switch c {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	}
	return 0
}

func rankSev(s string) int {
	switch s {
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	}
	return 0
}

// MatchScore exposes the contract matching used for links so that reports
// can find the consumer-side evidence of a relationship (-1 = no match).
func MatchScore(call, exposed model.Contract) int { return matchWith(call, exposed, 1) }

// vulnerabilities aggregates the advisories found in every system.
func vulnerabilities(all []*model.Facts) ([]model.VulnUsage, string, string) {
	byKey := map[string]*model.VulnUsage{}
	var keys []string
	db, asOf := "", ""
	for _, f := range all {
		if f.VulnScan != nil {
			db, asOf = f.VulnScan.Database, f.VulnScan.AsOf
		}
		for _, v := range f.Vulnerabilities {
			k := v.ID + "|" + v.Package
			u := byKey[k]
			if u == nil {
				u = &model.VulnUsage{ID: v.ID, Aliases: v.Aliases, Package: v.Package, Severity: v.Severity, Score: v.Score,
					Malicious: v.Malicious, Summary: v.Summary}
				byKey[k] = u
				keys = append(keys, k)
			}
			a := f.Repo.Name + "@" + v.Version
			found := false
			for _, x := range u.Affected {
				found = found || x == a
			}
			if !found {
				u.Affected = append(u.Affected, a)
			}
			for _, fx := range v.Fixed {
				dup := false
				for _, y := range u.Fixed {
					dup = dup || y == fx
				}
				if !dup {
					u.Fixed = append(u.Fixed, fx)
				}
			}
		}
	}
	out := make([]model.VulnUsage, 0, len(keys))
	for _, k := range keys {
		u := byKey[k]
		sort.Strings(u.Affected)
		sort.Strings(u.Fixed)
		out = append(out, *u)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := sevRank(out[i].Severity), sevRank(out[j].Severity); ri != rj {
			return ri > rj
		}
		if len(out[i].Affected) != len(out[j].Affected) {
			return len(out[i].Affected) > len(out[j].Affected)
		}
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}
		return out[i].ID < out[j].ID
	})
	if len(out) == 0 {
		out = nil
	}
	return out, db, asOf
}

func sevRank(s string) int {
	switch s {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "unknown":
		return 1
	}
	return 0
}
