package report

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/vuln"
)

const toolURI = "https://github.com/oskillarrr/traverse"

// ---------------------------------------------------------------------------
// SARIF 2.1.0 (OASIS Standard) — findings for code-scanning platforms.
// ---------------------------------------------------------------------------

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool              sarifTool         `json:"tool"`
	AutomationDetails map[string]string `json:"automationDetails"`
	Results           []sarifResult     `json:"results"`
	Properties        map[string]any    `json:"properties,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifRule struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	ShortDescription     sarifText         `json:"shortDescription"`
	FullDescription      *sarifText        `json:"fullDescription,omitempty"`
	HelpURI              string            `json:"helpUri,omitempty"`
	DefaultConfiguration map[string]string `json:"defaultConfiguration"`
	Properties           map[string]any    `json:"properties"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             sarifText         `json:"message"`
	Locations           []sarifLocation   `json:"locations"`
	RelatedLocations    []sarifLocation   `json:"relatedLocations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          map[string]any    `json:"properties"`
}

type sarifLocation struct {
	ID               int               `json:"id,omitempty"`
	PhysicalLocation sarifPhysLocation `json:"physicalLocation"`
}

type sarifPhysLocation struct {
	ArtifactLocation map[string]string `json:"artifactLocation"`
	Region           *sarifRegion      `json:"region,omitempty"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

var sarifLevel = map[string]string{"high": "error", "medium": "warning", "low": "note", "info": "none"}

func sarifURI(p string) string {
	parts := strings.Split(p, "/")
	for i, s := range parts {
		parts[i] = url.PathEscape(s)
	}
	return strings.Join(parts, "/")
}

// SARIF returns one run per system. Technology inventory (severity info) is
// left out: SARIF consumers present every result as an alert.
func SARIF(all []*model.Facts, tool model.ToolInfo) ([]byte, error) {
	log := sarifLog{Schema: "https://json.schemastore.org/sarif-2.1.0.json", Version: "2.1.0"}
	for _, f := range all {
		run := sarifRun{Tool: sarifTool{Driver: sarifDriver{Name: "Traverse", Version: tool.Version, InformationURI: toolURI}},
			AutomationDetails: map[string]string{"id": "traverse/" + f.Repo.Name + "/"},
			Properties:        map[string]any{"system": f.Repo.Name, "rulesVersion": tool.RulesVersion, "vcs": f.Repo.VCS, "revision": f.Repo.Revision}}
		ruleIdx := map[string]int{}
		for _, fd := range f.Findings {
			if fd.Severity == "info" {
				continue
			}
			idx, ok := ruleIdx[fd.RuleID]
			if !ok {
				idx = len(run.Tool.Driver.Rules)
				ruleIdx[fd.RuleID] = idx
				r := sarifRule{ID: fd.RuleID, Name: ruleName(fd.RuleID), ShortDescription: sarifText{fd.Title},
					DefaultConfiguration: map[string]string{"level": sarifLevel[fd.Severity]},
					Properties:           map[string]any{"category": fd.Category, "technology": fd.Technology, "confidence": fd.Confidence, "tags": []string{fd.Category}}}
				if fd.Message != "" {
					r.FullDescription = &sarifText{fd.Message}
				}
				run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, r)
			}
			msg := fd.Title
			if fd.Message != "" {
				msg += ". " + fd.Message
			}
			if fd.Count > 1 {
				msg += fmt.Sprintf(" (%d occurrences)", fd.Count)
			}
			loc := sarifLocation{PhysicalLocation: sarifPhysLocation{ArtifactLocation: map[string]string{"uri": sarifURI(fd.File), "uriBaseId": "SRCROOT"}}}
			if len(fd.Lines) > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{StartLine: fd.Lines[0]}
			}
			res := sarifResult{RuleID: fd.RuleID, RuleIndex: idx, Level: sarifLevel[fd.Severity], Message: sarifText{msg},
				Locations: []sarifLocation{loc}, PartialFingerprints: map[string]string{"traverseFindingId/v1": fd.ID},
				Properties: map[string]any{"count": fd.Count, "sourceSet": fd.SourceSet, "confidence": fd.Confidence}}
			for i, l := range fd.Lines {
				if i == 0 || i > 10 {
					continue
				}
				res.RelatedLocations = append(res.RelatedLocations, sarifLocation{ID: i, PhysicalLocation: sarifPhysLocation{
					ArtifactLocation: map[string]string{"uri": sarifURI(fd.File), "uriBaseId": "SRCROOT"}, Region: &sarifRegion{StartLine: l}}})
			}
			run.Results = append(run.Results, res)
		}
		for _, v := range f.Vulnerabilities {
			rid := "VULN/" + v.ID
			lvl := vulnLevel(v.Severity)
			idx, ok := ruleIdx[rid]
			if !ok {
				idx = len(run.Tool.Driver.Rules)
				ruleIdx[rid] = idx
				props := map[string]any{"category": "vulnerability", "confidence": v.Confidence, "tags": []string{"security", "vulnerability"},
					"aliases": v.Aliases, "source": "OSV.dev"}
				if v.Score > 0 {
					props["security-severity"] = fmt.Sprintf("%.1f", v.Score)
				}
				r := sarifRule{ID: rid, Name: "KnownVulnerability", ShortDescription: sarifText{firstNonEmpty(v.Summary, v.ID)},
					HelpURI: OSVURL(v.ID), DefaultConfiguration: map[string]string{"level": lvl}, Properties: props}
				run.Tool.Driver.Rules = append(run.Tool.Driver.Rules, r)
			}
			msg := fmt.Sprintf("%s %s is affected by %s", v.Package, v.Version, v.ID)
			if len(v.Aliases) > 0 {
				msg += " (" + strings.Join(v.Aliases, ", ") + ")"
			}
			msg += " [" + v.Severity + "]"
			if len(v.Fixed) > 0 {
				msg += ". Fixed in " + strings.Join(v.Fixed, ", ")
			}
			loc := sarifLocation{PhysicalLocation: sarifPhysLocation{ArtifactLocation: map[string]string{"uri": sarifURI(v.File), "uriBaseId": "SRCROOT"}}}
			if v.Line > 0 {
				loc.PhysicalLocation.Region = &sarifRegion{StartLine: v.Line}
			}
			run.Results = append(run.Results, sarifResult{RuleID: rid, RuleIndex: idx, Level: lvl, Message: sarifText{msg},
				Locations:           []sarifLocation{loc},
				PartialFingerprints: map[string]string{"traverseFindingId/v1": stableUUID(v.ID + "|" + v.Package + "|" + v.Version + "|" + v.File)},
				Properties:          map[string]any{"package": v.Package, "version": v.Version, "modules": v.Modules, "confidence": v.Confidence, "kind": v.Kind}})
		}
		if run.Results == nil {
			run.Results = []sarifResult{}
		}
		if run.Tool.Driver.Rules == nil {
			run.Tool.Driver.Rules = []sarifRule{}
		}
		log.Runs = append(log.Runs, run)
	}
	return json.MarshalIndent(log, "", " ")
}

// OSVURL is the public page of an OSV advisory.
func OSVURL(id string) string { return "https://osv.dev/vulnerability/" + url.PathEscape(id) }

func vulnLevel(sev string) string {
	switch sev {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	}
	return "note"
}

func ruleName(id string) string {
	var b strings.Builder
	for _, p := range strings.FieldsFunc(strings.ToLower(id), func(r rune) bool { return r == '-' || r == '_' }) {
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// CycloneDX 1.6 (ECMA-424) — SBOM of declared dependencies per system.
// ---------------------------------------------------------------------------

type cdxBOM struct {
	BOMFormat       string          `json:"bomFormat"`
	SpecVersion     string          `json:"specVersion"`
	SerialNumber    string          `json:"serialNumber"`
	Version         int             `json:"version"`
	Metadata        cdxMetadata     `json:"metadata"`
	Components      []cdxComponent  `json:"components"`
	Dependencies    []cdxDependency `json:"dependencies"`
	Vulnerabilities []cdxVuln       `json:"vulnerabilities,omitempty"`
}

type cdxSource struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type cdxVulnRef struct {
	ID     string    `json:"id"`
	Source cdxSource `json:"source"`
}

type cdxRating struct {
	Source   *cdxSource `json:"source,omitempty"`
	Score    float64    `json:"score,omitempty"`
	Severity string     `json:"severity"`
	Method   string     `json:"method,omitempty"`
	Vector   string     `json:"vector,omitempty"`
}

type cdxAffectVersion struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

type cdxAffect struct {
	Ref      string             `json:"ref"`
	Versions []cdxAffectVersion `json:"versions,omitempty"`
}

type cdxVuln struct {
	BOMRef         string        `json:"bom-ref"`
	ID             string        `json:"id"`
	Source         cdxSource     `json:"source"`
	References     []cdxVulnRef  `json:"references,omitempty"`
	Ratings        []cdxRating   `json:"ratings"`
	Description    string        `json:"description,omitempty"`
	Recommendation string        `json:"recommendation,omitempty"`
	Published      string        `json:"published,omitempty"`
	Affects        []cdxAffect   `json:"affects"`
	Properties     []cdxProperty `json:"properties,omitempty"`
}

func aliasSource(id string) cdxSource {
	switch {
	case strings.HasPrefix(id, "CVE-"):
		return cdxSource{"NVD", "https://nvd.nist.gov/vuln/detail/" + id}
	case strings.HasPrefix(id, "GHSA-"):
		return cdxSource{"GitHub Advisories", "https://github.com/advisories/" + id}
	}
	return cdxSource{"OSV", OSVURL(id)}
}

type cdxMetadata struct {
	Timestamp  string         `json:"timestamp,omitempty"`
	Tools      cdxTools       `json:"tools"`
	Component  cdxComponent   `json:"component"`
	Properties []cdxProperty  `json:"properties,omitempty"`
	Lifecycles []cdxLifecycle `json:"lifecycles"`
}

type cdxLifecycle struct {
	Phase string `json:"phase"`
}

type cdxTools struct {
	Components []cdxComponent `json:"components"`
}

type cdxComponent struct {
	Type       string        `json:"type"`
	BOMRef     string        `json:"bom-ref,omitempty"`
	Group      string        `json:"group,omitempty"`
	Name       string        `json:"name"`
	Version    string        `json:"version,omitempty"`
	Scope      string        `json:"scope,omitempty"`
	Purl       string        `json:"purl,omitempty"`
	Properties []cdxProperty `json:"properties,omitempty"`
	Evidence   *cdxEvidence  `json:"evidence,omitempty"`
}

type cdxProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cdxEvidence struct {
	Occurrences []cdxOccurrence `json:"occurrences"`
}

type cdxOccurrence struct {
	Location string `json:"location"`
	Line     int    `json:"line,omitempty"`
}

type cdxDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

var reExactVersion = regexp.MustCompile(`^v?\d+(\.[0-9A-Za-z-]+)*([.+-][0-9A-Za-z.-]+)?$`)

// purlOf maps an Traverse coordinate to a Package URL (ECMA-427 / purl-spec).
func purlOf(id, version string) (purl, group, name string) {
	eco, rest, _ := strings.Cut(id, ":")
	v := ""
	if version != "" && reExactVersion.MatchString(version) {
		v = "@" + url.PathEscape(version)
	}
	switch eco {
	case "maven":
		g, a, _ := strings.Cut(rest, ":")
		return "pkg:maven/" + g + "/" + a + v, g, a
	case "npm":
		if strings.HasPrefix(rest, "@") {
			scope, n, _ := strings.Cut(rest, "/")
			return "pkg:npm/%40" + scope[1:] + "/" + n + v, scope, n
		}
		return "pkg:npm/" + rest + v, "", rest
	case "nuget":
		return "pkg:nuget/" + rest + v, "", rest
	case "pypi":
		n := strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(rest))
		return "pkg:pypi/" + n + v, "", rest
	case "composer":
		return "pkg:composer/" + rest + v, "", rest
	case "go":
		return "pkg:golang/" + rest + v, "", rest
	}
	return "", "", rest
}

func stableUUID(s string) string {
	h := sha1.Sum([]byte(s))
	h[6] = (h[6] & 0x0f) | 0x50 // version 5 (name-based, SHA-1)
	h[8] = (h[8] & 0x3f) | 0x80
	x := hex.EncodeToString(h[:16])
	return x[0:8] + "-" + x[8:12] + "-" + x[12:16] + "-" + x[16:20] + "-" + x[20:32]
}

// CycloneDX returns the SBOM of one system: its modules and the libraries
// they declare (dependencyManagement entries excluded), with evidence.
func CycloneDX(f *model.Facts, tool model.ToolInfo) ([]byte, error) {
	sysRef := "system:" + f.Repo.Name
	bom := cdxBOM{BOMFormat: "CycloneDX", SpecVersion: "1.6", Version: 1,
		SerialNumber: "urn:uuid:" + stableUUID(f.Repo.Name+"|"+f.Repo.Revision+"|"+tool.Version),
		Metadata: cdxMetadata{Timestamp: f.Repo.ScannedAt,
			Tools:      cdxTools{Components: []cdxComponent{{Type: "application", Name: "Traverse", Version: tool.Version}}},
			Component:  cdxComponent{Type: "application", BOMRef: sysRef, Name: f.Repo.Name},
			Lifecycles: []cdxLifecycle{{Phase: "pre-build"}}}}
	if f.Repo.Revision != "" {
		bom.Metadata.Properties = append(bom.Metadata.Properties, cdxProperty{"traverse:revision", f.Repo.Revision})
	}
	refs := map[string]int{}
	deps := map[string]map[string]bool{}
	addDep := func(from, to string) {
		if deps[from] == nil {
			deps[from] = map[string]bool{}
		}
		deps[from][to] = true
	}
	moduleRef := map[string]string{}
	depRef := map[string]string{} // id|exact version -> bom-ref
	for _, m := range f.Modules {
		ref := "module:" + m.ID
		moduleRef[m.ID] = ref
		purl, group, name := purlOf(m.ID, m.Version)
		c := cdxComponent{Type: "application", BOMRef: ref, Group: group, Name: firstNonEmpty(name, m.Name), Version: m.Version, Purl: purl,
			Properties: []cdxProperty{{"traverse:build", m.Build}, {"traverse:descriptor", m.Descriptor}}}
		if m.Packaging == "jar" || m.Packaging == "library" || m.Packaging == "" {
			c.Type = "library"
		}
		if m.Runtime != "" {
			c.Properties = append(c.Properties, cdxProperty{"traverse:runtime", m.Runtime})
		}
		refs[ref] = len(bom.Components)
		bom.Components = append(bom.Components, c)
		addDep(sysRef, ref)
	}
	for _, m := range f.Modules {
		for _, d := range append(append([]model.Dependency(nil), m.Deps...), m.Transitive...) {
			if d.Managed {
				continue
			}
			if r, ok := moduleRef[d.ID]; ok {
				addDep(moduleRef[m.ID], r)
				continue
			}
			ver := d.Version
			if d.Locked != "" {
				ver = d.Locked
			}
			purl, group, name := purlOf(d.ID, ver)
			ref := purl
			if ref == "" {
				ref = "traverse:" + d.ID + "@" + ver
			}
			i, ok := refs[ref]
			if !ok {
				c := cdxComponent{Type: "library", BOMRef: ref, Group: group, Name: name, Purl: purl, Scope: "required"}
				if ver != "" && reExactVersion.MatchString(ver) {
					c.Version = ver
				}
				if d.Version != "" && d.Version != ver || ver != "" && !reExactVersion.MatchString(ver) {
					c.Properties = append(c.Properties, cdxProperty{"traverse:declaredVersion", d.Version})
				}
				if d.LockFile != "" && d.Kind != "transitive" {
					c.Properties = append(c.Properties, cdxProperty{"traverse:lockFile", d.LockFile})
				}
				if !d.Resolved {
					c.Properties = append(c.Properties, cdxProperty{"traverse:versionResolved", "false"})
				}
				c.Properties = append(c.Properties, cdxProperty{"traverse:kind", d.Kind})
				switch strings.ToLower(d.Scope) {
				case "test", "dev", "testimplementation", "testcompile":
					c.Scope = "excluded"
				case "provided", "compileonly", "providedcompile", "providedruntime":
					c.Properties = append(c.Properties, cdxProperty{"traverse:scope", "provided by the runtime"})
				}
				c.Evidence = &cdxEvidence{}
				i = len(bom.Components)
				refs[ref] = i
				bom.Components = append(bom.Components, c)
			}
			if v, why := vuln.Exact(ver); why == "" {
				depRef[d.ID+"|"+v] = ref
			}
			if ev := bom.Components[i].Evidence; ev != nil && len(ev.Occurrences) < 20 {
				ev.Occurrences = append(ev.Occurrences, cdxOccurrence{Location: d.File, Line: d.Line})
			}
			if d.Kind != "transitive" {
				addDep(moduleRef[m.ID], ref)
			}
		}
	}
	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	known := map[string]bool{sysRef: true}
	for _, c := range bom.Components {
		known[c.BOMRef] = true
	}
	for _, k := range keys {
		var on []string
		for t := range deps[k] {
			if known[t] {
				on = append(on, t)
			}
		}
		sort.Strings(on)
		bom.Dependencies = append(bom.Dependencies, cdxDependency{Ref: k, DependsOn: on})
	}
	if bom.Components == nil {
		bom.Components = []cdxComponent{}
	}
	if bom.Dependencies == nil {
		bom.Dependencies = []cdxDependency{}
	}
	if vs := f.VulnScan; vs != nil {
		bom.Metadata.Properties = append(bom.Metadata.Properties, cdxProperty{"traverse:vulnDatabase", "OSV.dev (" + vs.Database + ")"},
			cdxProperty{"traverse:vulnDataAsOf", vs.AsOf})
	}
	for _, v := range f.Vulnerabilities {
		ref, ok := depRef[v.Package+"|"+v.Version]
		if !ok {
			continue
		}
		x := cdxVuln{BOMRef: "vuln:" + v.ID + ":" + ref, ID: v.ID, Source: cdxSource{"OSV", OSVURL(v.ID)},
			Description: v.Summary, Published: v.Published,
			Affects:    []cdxAffect{{Ref: ref, Versions: []cdxAffectVersion{{Version: v.Version, Status: "affected"}}}},
			Properties: []cdxProperty{{"traverse:confidence", v.Confidence}}}
		for _, a := range v.Aliases {
			x.References = append(x.References, cdxVulnRef{ID: a, Source: aliasSource(a)})
		}
		r := cdxRating{Source: &cdxSource{Name: "OSV"}, Severity: v.Severity}
		if v.Vector != "" {
			r.Score, r.Vector, r.Method = v.Score, v.Vector, "CVSSv3"
			if strings.HasPrefix(v.Vector, "CVSS:3.1/") {
				r.Method = "CVSSv31"
			}
		} else {
			r.Method = "other"
		}
		x.Ratings = []cdxRating{r}
		if len(v.Fixed) > 0 {
			x.Recommendation = "Upgrade to " + strings.Join(v.Fixed, " or ")
		}
		if v.Malicious {
			x.Properties = append(x.Properties, cdxProperty{"traverse:malicious", "true"})
		}
		bom.Vulnerabilities = append(bom.Vulnerabilities, x)
	}
	return json.MarshalIndent(bom, "", " ")
}

// ---------------------------------------------------------------------------
// ArchiMate 3.1 Model Exchange File Format (The Open Group) — portfolio model
// importable into Archi, Sparx EA, BiZZdesign, ADOIT, etc.
// ---------------------------------------------------------------------------

type amText struct {
	Lang string `xml:"xml:lang,attr,omitempty"`
	Text string `xml:",chardata"`
}

type amProperty struct {
	Ref   string `xml:"propertyDefinitionRef,attr"`
	Value amText `xml:"value"`
}

type amElement struct {
	ID         string   `xml:"identifier,attr"`
	Type       string   `xml:"xsi:type,attr"`
	Name       amText   `xml:"name"`
	Doc        *amText  `xml:"documentation,omitempty"`
	Properties *amProps `xml:"properties,omitempty"`
}

type amProps struct {
	Items []amProperty `xml:"property"`
}

func props(ps []amProperty) *amProps {
	if len(ps) == 0 {
		return nil
	}
	return &amProps{ps}
}

type amRelationship struct {
	ID         string   `xml:"identifier,attr"`
	Source     string   `xml:"source,attr"`
	Target     string   `xml:"target,attr"`
	Type       string   `xml:"xsi:type,attr"`
	AccessType string   `xml:"accessType,attr,omitempty"`
	Name       *amText  `xml:"name,omitempty"`
	Doc        *amText  `xml:"documentation,omitempty"`
	Properties *amProps `xml:"properties,omitempty"`
}

type amPropDef struct {
	ID   string `xml:"identifier,attr"`
	Type string `xml:"type,attr"`
	Name amText `xml:"name"`
}

type amNode struct {
	ID    string   `xml:"identifier,attr"`
	Ref   string   `xml:"elementRef,attr"`
	Type  string   `xml:"xsi:type,attr"`
	X     int      `xml:"x,attr"`
	Y     int      `xml:"y,attr"`
	W     int      `xml:"w,attr"`
	H     int      `xml:"h,attr"`
	Nodes []amNode `xml:"node,omitempty"`
}

type amConn struct {
	ID     string `xml:"identifier,attr"`
	Ref    string `xml:"relationshipRef,attr"`
	Type   string `xml:"xsi:type,attr"`
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
}

type amView struct {
	ID    string   `xml:"identifier,attr"`
	Type  string   `xml:"xsi:type,attr"`
	Name  amText   `xml:"name"`
	Doc   *amText  `xml:"documentation,omitempty"`
	Nodes []amNode `xml:"node"`
	Conns []amConn `xml:"connection"`
}

type amModel struct {
	XMLName       xml.Name         `xml:"model"`
	NS            string           `xml:"xmlns,attr"`
	XSI           string           `xml:"xmlns:xsi,attr"`
	SchemaLoc     string           `xml:"xsi:schemaLocation,attr"`
	ID            string           `xml:"identifier,attr"`
	Name          amText           `xml:"name"`
	Doc           amText           `xml:"documentation"`
	Elements      []amElement      `xml:"elements>element"`
	Relationships []amRelationship `xml:"relationships>relationship,omitempty"`
	Organizations *amOrgs          `xml:"organizations,omitempty"`
	PropDefs      []amPropDef      `xml:"propertyDefinitions>propertyDefinition"`
	Views         []amView         `xml:"views>diagrams>view"`
}

type amOrgs struct {
	Items []amItem `xml:"item"`
}

type amItem struct {
	Ref   string   `xml:"identifierRef,attr,omitempty"`
	Label *amText  `xml:"label,omitempty"`
	Items []amItem `xml:"item,omitempty"`
}

func amID(kind, key string) string {
	h := sha1.Sum([]byte(kind + "|" + key))
	return "id-" + hex.EncodeToString(h[:10])
}

// ArchiMate builds the portfolio model: systems (ApplicationComponent),
// deployment artifacts, interfaces, data objects, platforms and their
// relationships, plus a landscape view.
func ArchiMate(all []*model.Facts, p *model.Portfolio, lang string) ([]byte, error) {
	m := amModel{NS: "http://www.opengroup.org/xsd/archimate/3.0/", XSI: "http://www.w3.org/2001/XMLSchema-instance",
		SchemaLoc: "http://www.opengroup.org/xsd/archimate/3.0/ http://www.opengroup.org/xsd/archimate/3.1/archimate3_Diagram.xsd",
		ID:        amID("model", "portfolio"), Name: amText{Lang: lang, Text: tr(lang, "portfolio")},
		Doc: amText{Lang: lang, Text: "Generated by Traverse " + p.Tool.Version + " (rules " + p.Tool.RulesVersion + "). " + tr(lang, "scopeNote")}}
	propDefs := []struct{ id, name string }{{"pd-lang", "Languages"}, {"pd-loc", "Lines of code"}, {"pd-risk-high", "High risks"},
		{"pd-risk-med", "Medium risks"}, {"pd-group", "Group"}, {"pd-conf", "Confidence"}, {"pd-evidence", "Evidence count"},
		{"pd-keys", "Keys"}, {"pd-runtime", "Runtime"}, {"pd-vcs", "Revision"}}
	for _, d := range propDefs {
		m.PropDefs = append(m.PropDefs, amPropDef{ID: d.id, Type: "string", Name: amText{Lang: lang, Text: d.name}})
	}
	prop := func(ref, v string) amProperty { return amProperty{Ref: ref, Value: amText{Lang: lang, Text: v}} }
	seen := map[string]bool{}
	addEl := func(id, typ, name, doc string, ps ...amProperty) string {
		if seen[id] {
			return id
		}
		seen[id] = true
		var keep []amProperty
		for _, p := range ps {
			if p.Value.Text != "" {
				keep = append(keep, p)
			}
		}
		e := amElement{ID: id, Type: typ, Name: amText{Lang: lang, Text: name}, Properties: props(keep)}
		if doc != "" {
			e.Doc = &amText{Lang: lang, Text: doc}
		}
		m.Elements = append(m.Elements, e)
		return id
	}
	relSeen := map[string]bool{}
	addRel := func(typ, src, dst, name, access string, ps ...amProperty) string {
		id := amID("rel", typ+src+dst+name)
		if relSeen[id] || src == dst {
			return id
		}
		relSeen[id] = true
		r := amRelationship{ID: id, Type: typ, Source: src, Target: dst, AccessType: access, Properties: props(ps)}
		if name != "" {
			r.Name = &amText{Lang: lang, Text: name}
		}
		m.Relationships = append(m.Relationships, r)
		return id
	}
	var groupItems = map[string][]amItem{}
	sysID := map[string]string{}
	for _, s := range p.Systems {
		id := addEl(amID("system", s.Name), "ApplicationComponent", s.Name, "",
			prop("pd-lang", strings.Join(s.Languages, ", ")), prop("pd-loc", fmt.Sprint(s.Code)),
			prop("pd-risk-high", fmt.Sprint(s.Risks["high"])), prop("pd-risk-med", fmt.Sprint(s.Risks["medium"])),
			prop("pd-group", s.Group), prop("pd-runtime", strings.Join(s.Runtimes, ", ")), prop("pd-vcs", s.Revision))
		sysID[s.Name] = id
		groupItems[s.Group] = append(groupItems[s.Group], amItem{Ref: id})
	}
	groups := make([]string, 0, len(groupItems))
	for g := range groupItems {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for _, g := range groups {
		items := groupItems[g]
		if g == "" {
			continue
		}
		gid := addEl(amID("group", g), "Grouping", g, "")
		for _, it := range items {
			addRel("Aggregation", gid, it.Ref, "", "")
		}
	}
	for _, f := range all {
		sid := sysID[f.Repo.Name]
		for _, d := range f.Deployables {
			name := strings.ToUpper(d.Kind) + " " + shortID(d.Module)
			aid := addEl(amID("artifact", f.Repo.Name+d.Module), "Artifact", name, "context root: "+d.ContextRoot)
			addRel("Realization", aid, sid, "", "")
		}
		// one interface per kind with its exposed keys as documentation
		byKind := map[string][]string{}
		for _, c := range f.Contracts {
			if c.Direction == "exposes" && c.Kind != "program" {
				byKind[c.Kind] = append(byKind[c.Kind], c.Key)
			}
		}
		kinds := make([]string, 0, len(byKind))
		for k := range byKind {
			kinds = append(kinds, k)
		}
		sort.Strings(kinds)
		for _, k := range kinds {
			keys := byKind[k]
			sort.Strings(keys)
			doc := strings.Join(firstStrs(keys, 200), "\n")
			iid := addEl(amID("iface", f.Repo.Name+k), "ApplicationInterface", f.Repo.Name+" · "+strings.ToUpper(k), doc,
				prop("pd-evidence", fmt.Sprint(len(keys))))
			addRel("Composition", sid, iid, "", "")
		}
		// data objects (tables) with access type
		ops := map[string]map[string]bool{}
		for _, c := range f.Contracts {
			if c.Kind == "db-table" || c.Kind == "dataset" {
				k := c.Kind + ":" + c.Match
				if ops[k] == nil {
					ops[k] = map[string]bool{}
				}
				ops[k][c.Direction] = true
			}
		}
		tkeys := make([]string, 0, len(ops))
		for k := range ops {
			tkeys = append(tkeys, k)
		}
		sort.Strings(tkeys)
		for _, k := range tkeys {
			name := strings.SplitN(k, ":", 2)[1]
			did := addEl(amID("data", k), "DataObject", name, "")
			r, w := ops[k]["read"], ops[k]["write"] || ops[k]["defines"]
			at := "Access"
			switch {
			case r && w:
				at = "ReadWrite"
			case w:
				at = "Write"
			case r:
				at = "Read"
			}
			addRel("Access", sid, did, "", at)
		}
		// deployment: workloads/services as Node, images as Artifact,
		// k8s Service/Ingress/Route as Technology Interface
		depID := map[string]string{}
		for _, n := range f.Deployment.Nodes {
			col := deployColumn(n)
			name := n.Name + " (" + n.Platform + " " + n.Kind + ")"
			switch col {
			case 1, 4:
				nid := addEl(amID("dnode", f.Repo.Name+n.ID), "Node", name, n.File)
				depID[n.ID] = nid
				for _, img := range n.Images {
					aid := addEl(amID("image", img), "Artifact", "image "+img, "")
					addRel("Assignment", nid, aid, "", "")
					addRel("Realization", aid, sid, "", "")
				}
				if len(n.Images) == 0 {
					addRel("Realization", nid, sid, "", "") // Compose build from this repository
				}
			case 0:
				aid := addEl(amID("dockerfile", f.Repo.Name+n.ID), "Artifact", "image "+n.Name+" ("+n.File+")", "FROM "+strings.Join(n.Base, ", "))
				depID[n.ID] = aid
				addRel("Realization", aid, sid, "", "")
			case 2, 3:
				iid := addEl(amID("dif", f.Repo.Name+n.ID), "TechnologyInterface", name, strings.Join(n.Paths, ", "))
				depID[n.ID] = iid
			}
		}
		for _, l := range f.Deployment.Links {
			a, b := depID[l.From], depID[l.To]
			if a != "" && b != "" {
				addRel("Association", a, b, l.Kind, "")
			}
		}
		// platforms: runtimes, application servers, databases
		plat := map[string]bool{}
		for _, m2 := range f.Modules {
			if m2.Runtime != "" {
				plat[m2.Runtime] = true
			}
		}
		for _, fd := range f.Findings {
			if fd.SourceSet == "main" && (fd.Category == "Platform" || fd.Category == "Database") && fd.Technology != "" {
				plat[fd.Technology] = true
			}
		}
		pk := make([]string, 0, len(plat))
		for k := range plat {
			pk = append(pk, k)
		}
		sort.Strings(pk)
		for _, k := range pk {
			pid := addEl(amID("platform", k), "SystemSoftware", k, "")
			addRel("Serving", pid, sid, "", "")
		}
	}
	// cross-system relationships
	var conns []struct{ rel, from, to string }
	for _, l := range p.Links {
		a, b := sysID[l.From], sysID[l.To]
		if a == "" || b == "" {
			continue
		}
		typ, src, dst := "Serving", b, a // provider serves consumer
		switch l.Kind {
		case "jms", "kafka", "shared-dataset":
			typ, src, dst = "Flow", a, b
		case "shared-db", "shared-datasource":
			typ, src, dst = "Association", a, b
		}
		rid := addRel(typ, src, dst, l.Kind, "", prop("pd-conf", l.Confidence), prop("pd-evidence", fmt.Sprint(l.Count)),
			prop("pd-keys", strings.Join(firstStrs(l.Keys, 10), " | ")))
		conns = append(conns, struct{ rel, from, to string }{rid, src, dst})
	}
	// organizations: folders by layer, as ArchiMate tools expect
	folders := map[string][]amItem{}
	for _, e := range m.Elements {
		folder := "Application"
		switch e.Type {
		case "Artifact", "SystemSoftware", "Node", "TechnologyInterface":
			folder = "Technology & Physical"
		case "Grouping":
			folder = "Other"
		}
		folders[folder] = append(folders[folder], amItem{Ref: e.ID})
	}
	orgs := &amOrgs{}
	for _, fn := range []string{"Application", "Technology & Physical", "Other"} {
		if len(folders[fn]) > 0 {
			orgs.Items = append(orgs.Items, amItem{Label: &amText{Lang: lang, Text: fn}, Items: folders[fn]})
		}
	}
	var relItems []amItem
	for _, r := range m.Relationships {
		relItems = append(relItems, amItem{Ref: r.ID})
	}
	if len(relItems) > 0 {
		orgs.Items = append(orgs.Items, amItem{Label: &amText{Lang: lang, Text: "Relations"}, Items: relItems})
	}
	m.Organizations = orgs
	// landscape view: systems on a circle
	v := amView{ID: amID("view", "landscape"), Type: "Diagram", Name: amText{Lang: lang, Text: tr(lang, "landscape")}}
	n := len(p.Systems)
	r := math.Max(220, float64(n)*45)
	nodeOf := map[string]string{}
	for i, s := range p.Systems {
		a := 2*math.Pi*float64(i)/math.Max(1, float64(n)) - math.Pi/2
		x, y := int(r+r*math.Cos(a))+40, int(r+r*math.Sin(a))+40
		if n == 1 {
			x, y = 40, 40
		}
		nid := amID("node", s.Name)
		nodeOf[sysID[s.Name]] = nid
		v.Nodes = append(v.Nodes, amNode{ID: nid, Ref: sysID[s.Name], Type: "Element", X: x, Y: y, W: 160, H: 60})
	}
	for _, c := range conns {
		if nodeOf[c.from] != "" && nodeOf[c.to] != "" {
			v.Conns = append(v.Conns, amConn{ID: amID("conn", c.rel), Ref: c.rel, Type: "Relationship", Source: nodeOf[c.from], Target: nodeOf[c.to]})
		}
	}
	m.Views = []amView{v}
	out, err := xml.MarshalIndent(m, "", " ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}

func shortID(id string) string {
	if i := strings.LastIndexByte(id, ':'); i >= 0 {
		return id[i+1:]
	}
	return id
}

// save returns a writer usable as save(path)(generator()).
func save(path string) func([]byte, error) error {
	return func(b []byte, err error) error {
		if err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o644)
	}
}
