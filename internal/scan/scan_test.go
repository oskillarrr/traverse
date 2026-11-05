package scan

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/oskillarrr/traverse/internal/model"
	"github.com/oskillarrr/traverse/internal/rules"
	"github.com/oskillarrr/traverse/internal/secure"
)

func scanFixture(t *testing.T, dir string) *model.Facts {
	t.Helper()
	cat, err := rules.Load(true, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Repo("testdata/"+dir, dir, "", Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &Config{}, Now: "test", Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func comp(f *model.Facts, fqn string) *model.Component {
	for i := range f.Components {
		if f.Components[i].FQN == fqn {
			return &f.Components[i]
		}
	}
	return nil
}

func hasContract(f *model.Facts, kind, dir, key string) bool {
	for _, c := range f.Contracts {
		if c.Kind == kind && c.Direction == dir && c.Key == key {
			return true
		}
	}
	return false
}

// Regression suite for the defects found in the PowerShell PoC (see docs).
func TestLegacyFixture(t *testing.T) {
	f := scanFixture(t, "legacy-es")
	const p = "org.example.legacy.gestion."

	// #4 #5: layers by whole package segment, Spanish vocabulary
	if c := comp(f, p+"transaction.servicio.ExpedienteServicio"); c == nil || c.Layer != LBusiness {
		t.Errorf("transaction.servicio should be Business, got %+v", c)
	}
	if c := comp(f, p+"negocio.GestorNegocio"); c == nil || c.Layer != LBusiness {
		t.Errorf("negocio should be Business, got %+v", c)
	}
	// #6: nested type FQN
	if c := comp(f, p+"negocio.GestorNegocio.Interna"); c == nil || !strings.HasPrefix(c.Kind, "nested") {
		t.Errorf("nested class not detected correctly: %+v", c)
	}
	// #7: test sources excluded by default
	if c := comp(f, p+"web.ExpedienteRestTest"); c != nil {
		t.Errorf("test class must be excluded by default")
	}
	// #8: wildcard import + field usage creates a dependency
	found := false
	for _, d := range f.Dependencies {
		if d.From == p+"negocio.GestorNegocio" && d.To == p+"web.ExpedienteRest" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing dependency GestorNegocio -> ExpedienteRest: %+v", f.Dependencies)
	}
	// #1 #2 #3: endpoints with correct verbs, class prefix, no phantom endpoint
	var eps []string
	for _, c := range f.Contracts {
		if c.Kind == "http" && c.Direction == "exposes" {
			eps = append(eps, c.Key)
		}
	}
	// no jboss-web.xml: the context root is inferred from the WAR name (artifactId-version)
	want := []string{"GET /modulo-a-1.0/expedientes/{id}", "POST /modulo-a-1.0/expedientes/alta"}
	if strings.Join(eps, "|") != strings.Join(want, "|") {
		t.Errorf("endpoints = %v, want %v", eps, want)
	}
	// #9 #10: nothing detected inside comments
	for _, fd := range f.Findings {
		if strings.HasSuffix(fd.File, "GestorNegocio.java") && (fd.RuleID == "JAVA-JMS" || fd.RuleID == "JAVA-HTTP-CLIENT") {
			t.Errorf("finding inside a comment: %+v", fd)
		}
	}
	// #11 #12: Maven parent inheritance and dependencyManagement
	if len(f.Modules) != 1 {
		t.Fatalf("modules = %+v", f.Modules)
	}
	m := f.Modules[0]
	if m.ID != "maven:org.example.legacy:modulo-a" || m.Version != "1.0" || m.Packaging != "war" {
		t.Errorf("unexpected module identity %+v", m)
	}
	for _, d := range m.Deps {
		if d.ID == "maven:x:gestionado" && !d.Managed {
			t.Errorf("dependencyManagement entry must be flagged managed")
		}
	}
}

func TestPoCExample(t *testing.T) {
	f := scanFixture(t, "poc-example")
	// #13: context root from jboss-web.xml combined with JAX-RS paths
	if !hasContract(f, "http", "exposes", "GET /pedidos/pedidos/estado") {
		t.Errorf("expected full route with context root, got %+v", f.Contracts)
	}
	if !hasContract(f, "jndi", "uses", "java:comp/env/jdbc/pedidos") {
		t.Errorf("datasource JNDI missing")
	}
	// #15: @Entity in a 'modelo' package is Model, not Persistence
	if c := comp(f, "org.example.portal.modelo.Pedido"); c == nil || c.Layer != LModel {
		t.Errorf("Pedido should be Model: %+v", c)
	}
	if !hasContract(f, "db-table", "uses", "PEDIDO") {
		t.Errorf("entity table missing")
	}
	// #14: one aggregated finding per rule and file
	n := 0
	for _, fd := range f.Findings {
		if fd.RuleID == "JAVA-JAVAX-EE" && strings.HasSuffix(fd.File, "PedidosResource.java") {
			n++
			if fd.Count != 3 {
				t.Errorf("expected count 3, got %d", fd.Count)
			}
		}
	}
	if n != 1 {
		t.Errorf("expected one aggregated javax finding, got %d", n)
	}
	// layer dependencies web -> service -> dao/integration
	for _, pair := range [][2]string{{"web.PedidosResource", "service.PedidosService"}, {"service.PedidosService", "dao.PedidosDao"}, {"service.PedidosService", "integracion.ProveedorSoap"}} {
		ok := false
		for _, d := range f.Dependencies {
			if d.From == "org.example.portal."+pair[0] && d.To == "org.example.portal."+pair[1] {
				ok = true
			}
		}
		if !ok {
			t.Errorf("missing dependency %v", pair)
		}
	}
}

func TestDeterministicOutput(t *testing.T) {
	a := scanFixture(t, "poc-example")
	b := scanFixture(t, "poc-example")
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("two scans of the same input differ")
	}
}

func TestSVNLayoutSkipsBranchesAndTags(t *testing.T) {
	f := scanFixture(t, "svn-layout")
	for _, fl := range f.Files {
		if strings.HasPrefix(fl.Path, "branches/") || strings.HasPrefix(fl.Path, "tags/") {
			t.Errorf("svn copy analyzed: %s", fl.Path)
		}
	}
	if f.Repo.Layout == "" {
		t.Errorf("layout note missing")
	}
}

func TestPolyglotFixture(t *testing.T) {
	f := scanFixture(t, "polyglot")
	for _, want := range [][3]string{
		{"http", "exposes", "GET /users"},                       // Express
		{"http", "exposes", "GET /orders/{order_id}"},           // FastAPI with APIRouter prefix
		{"http", "exposes", "GET /api/invoices/{id}"},           // Laravel routes/api.php
		{"http", "exposes", "GET /api/v1/articles/:slug"},       // gin group prefix
		{"http", "exposes", "GET /api/Catalog/{id}"},            // ASP.NET [controller] token
		{"http", "exposes", "GET /health/ready"},                // minimal API
		{"cics-transaction", "exposes", "CC00"},                 // CSD
		{"dataset", "read", "AWS.M2.CARDDEMO.USRSEC.VSAM.KSDS"}, // CICS file -> CSD DSNAME
		{"dataset", "defines", "AWS.M2.CARDDEMO.ACCT.REPORT"},   // JCL DISP=NEW
		{"db-table", "uses", "ORDERS"},                          // SQLAlchemy __tablename__
	} {
		if !hasContract(f, want[0], want[1], want[2]) {
			t.Errorf("missing contract %v", want)
		}
	}
	deps := map[string]bool{}
	for _, d := range f.Dependencies {
		deps[d.From+">"+d.To] = true
	}
	for _, want := range []string{
		"node.src.routes.users>node.src.services.userService",
		"py.shop.api.orders>py.shop.api.repo",
		"go>go.internal.handlers",
		"mainframe.mf.cbl.COSGN00C>mainframe.mf.cbl.COMEN01C", // EXEC CICS XCTL
		"mainframe.mf.cbl.COSGN00C>mainframe.mf.bms.COSGN00",  // SEND/RECEIVE MAP
		"mainframe.mf.jcl.CBACT01J>mainframe.mf.cbl.CBACT01C", // EXEC PGM=
	} {
		if !deps[want] {
			t.Errorf("missing dependency %s", want)
		}
	}
	for _, c := range f.Contracts {
		if c.Key == "COMMENTED" {
			t.Errorf("COBOL comment produced a contract")
		}
	}
	if c := comp(f, "mainframe.mf.cbl.COSGN00C"); c == nil || c.Layer != LPresentation {
		t.Errorf("CICS screen program layer: %+v", c)
	}
}

func TestDeploymentAndHostSealing(t *testing.T) {
	cat, _ := rules.Load(true, nil)
	run := func(s *secure.Sealer) *model.Facts {
		f, err := Repo("testdata/iac", "iac", "", Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &Config{}, Now: "t", Workers: 2, Sealer: s})
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	f := run(nil)
	kinds := map[string]bool{}
	for _, l := range f.Deployment.Links {
		kinds[l.Kind] = true
	}
	for _, k := range []string{"routes", "selects", "config", "depends_on", "env-ref"} {
		if !kinds[k] {
			t.Errorf("missing deployment link kind %s: %+v", k, f.Deployment.Links)
		}
	}
	for _, n := range f.Deployment.Nodes {
		if len(n.Hosts) > 0 {
			t.Errorf("hosts recorded without key: %+v", n)
		}
		for _, img := range n.Images {
			if strings.Contains(img, "registry.interno") {
				t.Errorf("registry host left in image name: %s", img)
			}
		}
	}
	b, _ := json.Marshal(f)
	for _, secret := range []string{"interno.example", "c2VjcmV0", "jdbc:oracle"} {
		if strings.Contains(string(b), secret) {
			t.Errorf("facts contain %q", secret)
		}
	}
	k, _ := secure.GenerateKey()
	kf := t.TempDir() + "/k"
	os.WriteFile(kf, []byte(k), 0o600)
	s, _ := secure.FromKeyFile(kf)
	f = run(s)
	sealed := 0
	for _, n := range f.Deployment.Nodes {
		for _, h := range n.Hosts {
			if !strings.HasPrefix(h, "host-") {
				t.Errorf("clear host %s", h)
			}
			sealed++
		}
	}
	if sealed < 2 || len(s.Vault().Entries) < 2 {
		t.Errorf("expected registry and ingress hosts sealed, got %d", sealed)
	}
	b, _ = json.Marshal(f)
	if strings.Contains(string(b), "interno.example") {
		t.Errorf("clear-text host in facts with key")
	}
}
