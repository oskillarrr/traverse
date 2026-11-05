package vuln

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		eco, a, b string
		want      int
	}{
		// Maven ComparableVersion
		{"Maven", "2.14.1", "2.15.0", -1},
		{"Maven", "2.0-beta9", "2.0", -1},
		{"Maven", "2.0-rc1", "2.0", -1},
		{"Maven", "2.0-beta9", "2.0-rc1", -1},
		{"Maven", "2.0-alpha1", "2.0-beta1", -1},
		{"Maven", "1.0-SNAPSHOT", "1.0", -1},
		{"Maven", "1.0", "1.0.0", 0},
		{"Maven", "3.2.2.RELEASE", "3.2.2", 0},
		{"Maven", "4.2.3.GA", "4.2.3", 0},
		{"Maven", "5.3.18", "5.3.20", -1},
		{"Maven", "1.0-sp1", "1.0", 1},
		{"Maven", "1.10", "1.9", 1},
		{"Maven", "2.0.0.M1", "2.0.0", -1},
		{"Maven", "1.0a1", "1.0-alpha-1", 0},
		{"Maven", "2.17.1", "2.15.0", 1},
		{"Maven", "1.2.17", "1.2.9", 1},
		// SemVer
		{"npm", "4.17.20", "4.17.21", -1},
		{"npm", "1.0.0-alpha", "1.0.0", -1},
		{"npm", "1.0.0-alpha.1", "1.0.0-alpha.beta", -1},
		{"npm", "1.0.0-beta.2", "1.0.0-beta.11", -1},
		{"npm", "1.0.0+build", "1.0.0", 0},
		{"Go", "0.0.0-20210220033148-5ea612d1eb83", "0.0.0", -1},
		{"NuGet", "13.0.1", "13.0.1.0", 0},
		{"NuGet", "1.0.0-RC1", "1.0.0-rc1", 0},
		// PEP 440
		{"PyPI", "2.25.1", "2.31.0", -1},
		{"PyPI", "1.0rc1", "1.0", -1},
		{"PyPI", "1.0.dev1", "1.0a1", -1},
		{"PyPI", "1.0", "1.0.post1", -1},
		{"PyPI", "1.0", "1.0.0", 0},
		{"PyPI", "1!0.1", "2.0", 1},
	}
	for _, c := range cases {
		if got := Compare(c.eco, c.a, c.b); got != c.want {
			t.Errorf("Compare(%s, %q, %q) = %d, want %d", c.eco, c.a, c.b, got, c.want)
		}
		if got := Compare(c.eco, c.b, c.a); got != -c.want {
			t.Errorf("Compare(%s, %q, %q) = %d, want %d (antisymmetry)", c.eco, c.b, c.a, got, -c.want)
		}
	}
}

func TestExact(t *testing.T) {
	cases := map[string]string{
		"1.2.3": "", "v1.2.3": "", "==1.2.3": "", "[1.2.3]": "", "=1.2.3": "",
		"^4.17.0": "range", "~1.2": "range", ">=1.0": "range", "[1.0,2.0)": "range", "1.+": "range", "*": "range",
		"latest": "range", "${log4j.version}": "unresolved", "": "missing", "1.x": "range",
	}
	for in, want := range cases {
		if _, r := Exact(in); r != want {
			t.Errorf("Exact(%q) reason = %q, want %q", in, r, want)
		}
	}
	if v, _ := Exact("[13.0.1]"); v != "13.0.1" {
		t.Errorf("Exact([13.0.1]) = %q", v)
	}
}

func TestCVSS3(t *testing.T) {
	cases := map[string]float64{
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H": 10.0,
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H": 9.8,
		"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H": 8.1,
		"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N": 6.5,
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N": 6.1,
		"CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N": 3.3,
		"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:N": 0,
	}
	for v, want := range cases {
		got, ok := CVSS3(v)
		if !ok || got != want {
			t.Errorf("CVSS3(%s) = %v,%v want %v", v, got, ok, want)
		}
	}
	if _, ok := CVSS3("CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:N/SI:N/SA:N"); ok {
		t.Error("CVSS v4 must not be scored by the v3 formula")
	}
	if SeverityOf(10) != "critical" || SeverityOf(7) != "high" || SeverityOf(6.9) != "medium" || SeverityOf(0.1) != "low" {
		t.Error("qualitative scale")
	}
}

// Test records follow the OSV schema; identifiers and ranges are invented.
var testRecords = map[string]string{
	"TEST-0001.json": `{"id":"GHSA-test-0001","modified":"2026-10-01T00:00:00Z","published":"2021-12-10T00:00:00Z",
	 "aliases":["CVE-2099-0001"],"summary":"Remote code execution in lookup feature",
	 "severity":[{"type":"CVSS_V3","score":"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"}],
	 "affected":[{"package":{"ecosystem":"Maven","name":"org.apache.logging.log4j:log4j-core"},
	   "ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"2.0-beta9"},{"fixed":"2.15.0"}]}]}]}`,
	"TEST-0002.json": `{"id":"CVE-2099-0001","modified":"2026-10-02T00:00:00Z","aliases":["GHSA-test-0001"],
	 "affected":[{"package":{"ecosystem":"Maven","name":"org.apache.logging.log4j:log4j-core"},
	   "ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"fixed":"2.15.0"}]}]}]}`,
	"TEST-0003.json": `{"id":"GHSA-test-0003","modified":"2026-09-01T00:00:00Z","summary":"Prototype pollution",
	 "database_specific":{"severity":"MODERATE"},
	 "affected":[{"package":{"ecosystem":"npm","name":"lodash"},
	   "ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"4.17.21"}]}]}]}`,
	"TEST-0004.json": `{"id":"GHSA-test-0004","modified":"2026-09-01T00:00:00Z","withdrawn":"2026-09-02T00:00:00Z",
	 "affected":[{"package":{"ecosystem":"npm","name":"lodash"},"versions":["4.17.20"]}]}`,
	"TEST-0005.json": `{"id":"MAL-2099-0005","modified":"2026-09-01T00:00:00Z","summary":"Malicious code in evil-pkg",
	 "affected":[{"package":{"ecosystem":"npm","name":"evil-pkg"},"versions":["1.0.0"]}]}`,
	"TEST-0006.json": `{"id":"PYSEC-2099-6","modified":"2026-08-01T00:00:00Z","details":"Header injection.\nMore text.",
	 "affected":[{"package":{"ecosystem":"PyPI","name":"Requests"},
	   "ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"last_affected":"2.30.0"}]}]}]}`,
	"TEST-0007.json": `{"id":"GHSA-test-0007","modified":"2026-08-01T00:00:00Z",
	 "affected":[{"package":{"ecosystem":"Maven","name":"commons-collections:commons-collections"},
	   "ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"3.0"},{"fixed":"3.2.2"}]}]}]}`,
	"TEST-0008.json": `{"id":"GHSA-test-0008","modified":"2026-08-01T00:00:00Z",
	 "affected":[{"package":{"ecosystem":"Maven","name":"org.springframework:spring-core"},
	   "ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"fixed":"3.2.9"}]}]}]}`,
}

// writeZip writes one <Ecosystem>-all.zip per ecosystem, like the OSV dumps,
// and returns the folder.
func writeZip(t *testing.T, dir string) string {
	byEco := map[string]map[string]string{}
	for name, body := range testRecords {
		var h osvHead
		if err := json.Unmarshal([]byte(body), &h); err != nil {
			t.Fatal(name, err)
		}
		e := h.Affected[0].Package.Ecosystem
		if byEco[e] == nil {
			byEco[e] = map[string]string{}
		}
		byEco[e][name] = body
	}
	for e, recs := range byEco {
		f, err := os.Create(filepath.Join(dir, e+"-all.zip"))
		if err != nil {
			t.Fatal(err)
		}
		zw := zip.NewWriter(f)
		for name, body := range recs {
			w, _ := zw.Create(name)
			w.Write([]byte(body))
		}
		zw.Close()
		f.Close()
	}
	return dir
}

func TestPartialEcosystem(t *testing.T) {
	dir := t.TempDir()
	f, _ := os.Create(filepath.Join(dir, "Maven-all.zip"))
	zw := zip.NewWriter(f)
	for i, body := range []string{
		`{"id":"GHSA-m1","modified":"2026-11-01T00:00:00Z","affected":[{"package":{"ecosystem":"Maven","name":"org.webjars.npm:lodash"},"versions":["4.17.20"]},{"package":{"ecosystem":"npm","name":"lodash"},"versions":["4.17.20"]}]}`,
		`{"id":"GHSA-m2","modified":"2026-11-01T00:00:00Z","affected":[{"package":{"ecosystem":"Maven","name":"g:a"},"versions":["1"]}]}`,
		`{"id":"GHSA-m3","modified":"2026-11-01T00:00:00Z","affected":[{"package":{"ecosystem":"Maven","name":"g:b"},"versions":["1"]}]}`,
	} {
		w, _ := zw.Create(fmt.Sprintf("r%d.json", i))
		w.Write([]byte(body))
	}
	zw.Close()
	f.Close()
	fx := &model.Facts{Modules: []model.Module{{ID: "npm:web", Deps: []model.Dependency{{ID: "npm:lodash", Version: "4.17.20", Kind: "declared", Resolved: true}}}}}
	db, err := Load([]string{dir}, Wanted([]*model.Facts{fx}))
	if err != nil {
		t.Fatal(err)
	}
	if !db.Complete["Maven"] || db.Complete["npm"] || db.Advisories["npm"] != 1 {
		t.Fatalf("complete=%v advisories=%v", db.Complete, db.Advisories)
	}
	db.Match(fx, "2026-11-02T00:00:00Z")
	if len(fx.Vulnerabilities) != 0 || len(fx.VulnScan.Unevaluated) != 1 || fx.VulnScan.Unevaluated[0].Reason != "ecosystem-not-loaded" {
		t.Errorf("an ecosystem only mentioned by another dump must not be evaluated: %+v %+v", fx.Vulnerabilities, fx.VulnScan.Unevaluated)
	}
	if _, ok := fx.VulnScan.Advisories["npm"]; ok {
		t.Error("partial ecosystems must not be reported as loaded")
	}
}

func facts() *model.Facts {
	return &model.Facts{Modules: []model.Module{
		{ID: "maven:g:app", Deps: []model.Dependency{
			{ID: "maven:org.apache.logging.log4j:log4j-core", Version: "2.14.1", Kind: "declared", Resolved: true, File: "pom.xml", Line: 12},
			{ID: "maven:org.springframework:spring-core", Version: "3.2.2.RELEASE", Kind: "declared", Resolved: true, File: "pom.xml", Line: 20},
			{ID: "maven:x:y", Version: "${y.version}", Kind: "declared", File: "pom.xml", Line: 30},
			{ID: "jar:commons-collections", Version: "3.2.1", Kind: "vendored-jar", Resolved: true, File: "WEB-INF/lib/commons-collections-3.2.1.jar"},
			{ID: "maven:g:lib", Version: "1.0", Kind: "declared"},
		}},
		{ID: "maven:g:lib", Deps: []model.Dependency{
			{ID: "maven:org.apache.logging.log4j:log4j-core", Version: "2.17.1", Kind: "declared", Resolved: true, File: "lib/pom.xml"},
		}},
		{ID: "npm:web", Deps: []model.Dependency{
			{ID: "npm:lodash", Version: "4.17.20", Kind: "declared", Resolved: true, File: "package.json"},
			{ID: "npm:evil-pkg", Version: "1.0.0", Kind: "declared", Resolved: true, File: "package.json"},
			{ID: "npm:react", Version: "^18.2.0", Kind: "declared", Resolved: true, File: "package.json"},
		}},
		{ID: "pypi:tool", Deps: []model.Dependency{
			{ID: "pypi:requests", Version: "2.30.0", Kind: "declared", Resolved: true, File: "requirements.txt"},
			{ID: "pypi:urllib3", Version: "2.0.0", Kind: "declared", Resolved: true, File: "requirements.txt"},
		}},
		{ID: "nuget:Api", Deps: []model.Dependency{
			{ID: "nuget:Newtonsoft.Json", Version: "13.0.1", Kind: "declared", Resolved: true, File: "Api.csproj"},
		}},
	}}
}

func TestMatch(t *testing.T) {
	dir := t.TempDir()
	p := writeZip(t, dir)
	f := facts()
	db, err := Load([]string{p}, Wanted([]*model.Facts{f}))
	if err != nil {
		t.Fatal(err)
	}
	if db.AsOf != "2026-10-02T00:00:00Z" || db.Advisories["Maven"] != 4 || db.Advisories["npm"] != 3 {
		t.Fatalf("db summary: %s %v", db.AsOf, db.Advisories)
	}
	db.Match(f, "2026-10-20T00:00:00Z")
	got := map[string]model.Vulnerability{}
	for _, v := range f.Vulnerabilities {
		got[v.Package+"@"+v.Version] = v
	}
	log4j, ok := got["maven:org.apache.logging.log4j:log4j-core@2.14.1"]
	if !ok {
		t.Fatalf("log4j 2.14.1 not reported: %+v", f.Vulnerabilities)
	}
	if log4j.ID != "GHSA-test-0001" || len(log4j.Aliases) != 1 || log4j.Aliases[0] != "CVE-2099-0001" {
		t.Errorf("aliases not unified: %+v", log4j)
	}
	if log4j.Score != 10 || log4j.Severity != "critical" || log4j.Confidence != "high" || log4j.Line != 12 || len(log4j.Fixed) != 1 || log4j.Fixed[0] != "2.15.0" {
		t.Errorf("log4j: %+v", log4j)
	}
	if _, ok := got["maven:org.apache.logging.log4j:log4j-core@2.17.1"]; ok {
		t.Error("fixed version reported as vulnerable")
	}
	if v, ok := got["maven:org.springframework:spring-core@3.2.2.RELEASE"]; !ok || v.Severity != "unknown" {
		t.Errorf("spring-core qualifier: %+v", v)
	}
	if v, ok := got["jar:commons-collections@3.2.1"]; !ok || v.Confidence != "medium" || v.Kind != "vendored-jar" {
		t.Errorf("vendored jar: %+v", v)
	}
	if v, ok := got["npm:lodash@4.17.20"]; !ok || v.Severity != "medium" || v.ID != "GHSA-test-0003" {
		t.Errorf("lodash (withdrawn advisory must be ignored): %+v", v)
	}
	if v, ok := got["npm:evil-pkg@1.0.0"]; !ok || !v.Malicious || v.Severity != "critical" {
		t.Errorf("malicious: %+v", v)
	}
	if v, ok := got["pypi:requests@2.30.0"]; !ok || v.Summary != "Header injection." {
		t.Errorf("PyPI normalization / last_affected / details summary: %+v", v)
	}
	if _, ok := got["pypi:urllib3@2.0.0"]; ok {
		t.Error("unexpected urllib3")
	}
	if len(got) != 6 {
		t.Errorf("want 6 vulnerabilities, got %d: %v", len(got), got)
	}
	if f.Vulnerabilities[0].Severity != "critical" || f.Vulnerabilities[len(f.Vulnerabilities)-1].Severity != "unknown" {
		t.Error("ordering by severity")
	}
	reasons := map[string]string{}
	for _, u := range f.VulnScan.Unevaluated {
		reasons[u.Package] = u.Reason
	}
	if reasons["maven:x:y"] != "unresolved" || reasons["npm:react"] != "range" || reasons["nuget:Newtonsoft.Json"] != "ecosystem-not-loaded" {
		t.Errorf("unevaluated: %v", reasons)
	}
	if _, ok := reasons["maven:g:lib"]; ok {
		t.Error("internal module must not be evaluated")
	}
	if f.VulnScan.Checked != 8 {
		t.Errorf("checked = %d", f.VulnScan.Checked)
	}
	for _, d := range f.Diagnostics {
		if d.Kind == "vulndb-stale" {
			t.Error("an 18-day-old database must not be reported as stale")
		}
	}
}

func TestStaleAndAmbiguous(t *testing.T) {
	dir := t.TempDir()
	writeZip(t, dir)
	extra := `{"id":"GHSA-test-0009","modified":"2026-01-01T00:00:00Z","affected":[{"package":{"ecosystem":"Maven","name":"org.other:commons-collections"},"ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"}]}]}]}`
	os.WriteFile(filepath.Join(dir, "extra.json"), []byte(extra), 0o644)
	f := &model.Facts{Modules: []model.Module{{ID: "maven:g:a", Deps: []model.Dependency{
		{ID: "jar:commons-collections", Version: "3.2.1", Kind: "vendored-jar", Resolved: true, File: "lib/x.jar"}}}}}
	db, err := Load([]string{dir}, Wanted([]*model.Facts{f}))
	if err != nil {
		t.Fatal(err)
	}
	db.Match(f, "2027-01-01T00:00:00Z")
	if len(f.Vulnerabilities) != 2 {
		t.Fatalf("want 2, got %+v", f.Vulnerabilities)
	}
	for _, v := range f.Vulnerabilities {
		if v.Confidence != "low" {
			t.Errorf("ambiguous artifactId must be low confidence: %+v", v)
		}
	}
	stale := false
	for _, d := range f.Diagnostics {
		stale = stale || d.Kind == "vulndb-stale"
	}
	if !stale {
		t.Error("stale database not reported")
	}
}

func TestLimits(t *testing.T) {
	dir := t.TempDir()
	old := MaxEntryBytes
	MaxEntryBytes = 100
	defer func() { MaxEntryBytes = old }()
	p := writeZip(t, dir)
	db, err := Load([]string{p}, &Want{})
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Diag) == 0 || !strings.Contains(db.Diag[0], "limit") {
		t.Errorf("oversized entries must be skipped with a diagnostic: %v", db.Diag)
	}
}

func TestInfoKeepsNothing(t *testing.T) {
	p := writeZip(t, t.TempDir())
	db, err := Info([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if db.Loaded != 0 || db.Advisories["PyPI"] != 1 {
		t.Errorf("info: loaded=%d %v", db.Loaded, db.Advisories)
	}
}
