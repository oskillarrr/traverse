package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
)

// writeCSV writes UTF-8 with BOM so spreadsheet tools detect the encoding.
func writeCSV(dir, name string, header []string, rows [][]string) error {
	fh, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer fh.Close()
	fh.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(fh)
	w.Write(header)
	for _, r := range rows {
		for i, c := range r {
			r[i] = csvSafe(c)
		}
		w.Write(r)
	}
	w.Flush()
	return w.Error()
}

// csvSafe neutralizes spreadsheet formula injection.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func itoa(i int) string { return strconv.Itoa(i) }

func joinInts(xs []int) string {
	var s []string
	for _, x := range xs {
		s = append(s, strconv.Itoa(x))
	}
	return strings.Join(s, " ")
}

func repoCSVs(dir string, f *model.Facts) error {
	var rows [][]string
	w := func(name string, h []string) error {
		err := writeCSV(dir, name, h, rows)
		rows = nil
		return err
	}
	for _, x := range f.Languages {
		rows = append(rows, []string{x.Language, x.Category, itoa(x.Files), itoa(x.Code), itoa(x.Comment), itoa(x.Blank)})
	}
	if err := w("languages.csv", []string{"language", "category", "files", "code", "comment", "blank"}); err != nil {
		return err
	}
	for _, x := range f.Files {
		rows = append(rows, []string{x.Path, x.Language, x.Category, x.SourceSet, x.Module, fmt.Sprint(x.Bytes), itoa(x.Code), itoa(x.Comment), itoa(x.Blank), x.Encoding})
	}
	w("files.csv", []string{"path", "language", "category", "sourceSet", "module", "bytes", "code", "comment", "blank", "encoding"})
	for _, m := range f.Modules {
		rows = append(rows, []string{m.ID, m.Build, m.Name, m.Version, m.Packaging, m.Runtime, m.Parent, m.Path, m.Descriptor, itoa(len(m.Deps))})
	}
	w("modules.csv", []string{"id", "build", "name", "version", "packaging", "runtime", "parent", "path", "descriptor", "dependencies"})
	for _, m := range f.Modules {
		for _, d := range m.Deps {
			rows = append(rows, []string{m.ID, d.ID, d.Version, d.Scope, d.Kind, fmt.Sprint(d.Managed), fmt.Sprint(d.Resolved), d.File, itoa(d.Line), d.Locked, d.LockFile})
		}
		for _, d := range m.Transitive {
			rows = append(rows, []string{m.ID, d.ID, d.Version, "", d.Kind, "false", "true", d.File, "", d.Version, d.File})
		}
	}
	w("build-dependencies.csv", []string{"module", "dependency", "version", "scope", "kind", "managed", "resolved", "file", "line", "locked", "lockFile"})
	for _, d := range f.Deployables {
		rows = append(rows, []string{d.Module, d.Kind, d.ContextRoot, strings.Join(d.AppPaths, " "), strings.Join(d.Includes, " "), d.Descriptor})
	}
	w("deployables.csv", []string{"module", "kind", "contextRoot", "appPaths", "includes", "descriptor"})
	for _, c := range f.Components {
		rows = append(rows, []string{c.FQN, c.Kind, c.Language, c.Package, c.Module, c.SourceSet, c.Layer, c.LayerReason, strings.Join(c.Stereotypes, " "), strings.Join(c.Extends, " "), c.File, itoa(c.Line)})
	}
	w("components.csv", []string{"fqn", "kind", "language", "package", "module", "sourceSet", "layer", "layerReason", "stereotypes", "extends", "file", "line"})
	for _, d := range f.Dependencies {
		rows = append(rows, []string{d.From, d.To, d.Kind, d.Confidence, d.File, itoa(d.Line)})
	}
	w("code-dependencies.csv", []string{"from", "to", "kind", "confidence", "file", "line"})
	for _, c := range f.Contracts {
		rows = append(rows, []string{c.Kind, c.Direction, c.Key, c.Match, c.Component, c.Detail, c.Host, c.Confidence, c.File, itoa(c.Line),
			c.Transport, c.Auth, c.Role, c.Counterpart})
	}
	w("contracts.csv", []string{"kind", "direction", "key", "match", "component", "detail", "host", "confidence", "file", "line",
		"transport", "auth", "role", "counterpart"})
	for _, n := range f.Deployment.Nodes {
		rows = append(rows, []string{n.Platform, n.Kind, n.Name, n.Namespace, strings.Join(n.Images, " "), strings.Join(n.Base, " "), strings.Join(n.Ports, " "),
			n.Replicas, n.Schedule, strings.Join(n.EnvKeys, " "), strings.Join(n.Hosts, " "), strings.Join(n.Paths, " "), n.User, n.Module, n.File, itoa(n.Line)})
	}
	w("deployment.csv", []string{"platform", "kind", "name", "namespace", "images", "baseImages", "ports", "replicas", "schedule", "envKeys", "hosts", "paths", "user", "module", "file", "line"})
	for _, x := range f.Findings {
		rows = append(rows, []string{x.ID, x.Severity, x.RuleID, x.Title, x.Category, x.Technology, x.Confidence, x.File, joinInts(x.Lines), itoa(x.Count), x.SourceSet, x.Message, ""})
	}
	w("findings.csv", []string{"id", "severity", "ruleId", "title", "category", "technology", "confidence", "file", "lines", "count", "sourceSet", "message", "validation"})
	for _, p := range f.PackageStats {
		rows = append(rows, []string{p.Package, p.Module, p.Layer, itoa(p.Types), itoa(p.Afferent), itoa(p.Efferent), strconv.FormatFloat(p.Instability, 'f', 2, 64)})
	}
	w("package-metrics.csv", []string{"package", "module", "layer", "types", "ca", "ce", "instability"})
	for _, c := range f.Cycles {
		rows = append(rows, []string{c.Level, itoa(len(c.Members)), strings.Join(c.Members, " ")})
	}
	w("cycles.csv", []string{"level", "size", "members"})
	for _, l := range f.LayerIssues {
		rows = append(rows, []string{l.FromLayer, l.ToLayer, l.From, l.To, l.File, itoa(l.Line)})
	}
	w("layer-violations.csv", []string{"fromLayer", "toLayer", "from", "to", "file", "line"})
	if vs := f.VulnScan; vs != nil {
		for _, v := range f.Vulnerabilities {
			rows = append(rows, []string{v.Severity, score(v.Score), v.ID, strings.Join(v.Aliases, " "), v.Package, v.Version, strings.Join(v.Fixed, " "),
				fmt.Sprint(v.Malicious), v.Confidence, v.Kind, v.Scope, strings.Join(v.Modules, " "), v.File, itoa(v.Line), v.Summary, OSVURL(v.ID), ""})
		}
		w("vulnerabilities.csv", []string{"severity", "cvss", "id", "aliases", "package", "version", "fixedIn", "malicious", "confidence", "kind", "scope",
			"modules", "file", "line", "summary", "url", "validation"})
		for _, u := range vs.Unevaluated {
			rows = append(rows, []string{u.Package, u.Version, u.Reason, u.File, itoa(u.Line)})
		}
		w("vulnerabilities-unevaluated.csv", []string{"package", "declaredVersion", "reason", "file", "line"})
	}
	for _, d := range f.Diagnostics {
		rows = append(rows, []string{d.Kind, d.File, d.Message})
	}
	return w("diagnostics.csv", []string{"kind", "file", "message"})
}

func score(x float64) string {
	if x == 0 {
		return ""
	}
	return strconv.FormatFloat(x, 'f', 1, 64)
}

func portfolioCSVs(dir string, p *model.Portfolio) error {
	var rows [][]string
	for _, s := range p.Systems {
		rows = append(rows, []string{s.Name, s.Group, s.VCS, s.Revision, strings.Join(s.Languages, " "), itoa(s.Code), itoa(s.Files), itoa(s.Modules),
			itoa(s.Components), itoa(s.Exposes), itoa(s.Calls), itoa(s.Tables), strings.Join(s.Runtimes, " "), strings.Join(s.Deployables, " "),
			itoa(s.Risks["high"]), itoa(s.Risks["medium"]), itoa(s.Risks["low"]), itoa(s.Cycles), itoa(s.LayerIssues)})
	}
	if err := writeCSV(dir, "systems.csv", []string{"system", "group", "vcs", "revision", "languages", "code", "files", "modules", "components",
		"exposes", "calls", "tables", "runtimes", "deployables", "risksHigh", "risksMedium", "risksLow", "cycles", "layerViolations"}, rows); err != nil {
		return err
	}
	rows = nil
	for _, l := range p.Links {
		cand := ""
		if l.Candidates > 1 {
			cand = itoa(l.Candidates)
		}
		rows = append(rows, []string{l.From, l.To, l.Kind, itoa(l.Count), l.Confidence, strings.Join(l.Keys, " | "), cand})
	}
	writeCSV(dir, "links.csv", []string{"from", "to", "kind", "evidences", "confidence", "keys", "candidateProviders"}, rows)
	rows = nil
	for _, t := range p.SharedTables {
		rows = append(rows, []string{t.Table, itoa(len(t.Systems)), strings.Join(t.Access, " ")})
	}
	writeCSV(dir, "shared-tables.csv", []string{"table", "systems", "access"}, rows)
	rows = nil
	for _, l := range p.LibraryVersions {
		for _, v := range sortedKeys(l.Versions) {
			rows = append(rows, []string{l.Library, fmt.Sprint(l.Internal), v, strings.Join(l.Versions[v], " ")})
		}
	}
	writeCSV(dir, "library-versions.csv", []string{"library", "internal", "version", "systems"}, rows)
	rows = nil
	for _, e := range p.External {
		rows = append(rows, []string{e.Kind, e.Key, e.Host, strings.Join(e.Systems, " ")})
	}
	writeCSV(dir, "external-dependencies.csv", []string{"kind", "key", "host", "systems"}, rows)
	rows = nil
	if p.VulnDB != "" {
		for _, v := range p.Vulnerabilities {
			rows = append(rows, []string{v.Severity, score(v.Score), v.ID, strings.Join(v.Aliases, " "), v.Package, strings.Join(v.Fixed, " "),
				fmt.Sprint(v.Malicious), itoa(len(v.Affected)), strings.Join(v.Affected, " "), v.Summary, OSVURL(v.ID)})
		}
		writeCSV(dir, "vulnerabilities.csv", []string{"severity", "cvss", "id", "aliases", "package", "fixedIn", "malicious", "systems", "affected", "summary", "url"}, rows)
		rows = nil
	}
	for _, t := range p.Technologies {
		for _, s := range sortedKeys(t.Systems) {
			rows = append(rows, []string{t.Category, t.Technology, t.Severity, s, itoa(t.Systems[s])})
		}
	}
	return writeCSV(dir, "technologies.csv", []string{"category", "technology", "severity", "system", "files"}, rows)
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
