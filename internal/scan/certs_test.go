package scan

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oskillarrr/traverse/internal/rules"
	"github.com/oskillarrr/traverse/internal/secure"
)

func mkCert(t *testing.T, subj pkix.Name, ca bool, notAfter time.Time) []byte {
	t.Helper()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: subj, Issuer: pkix.Name{CommonName: "Example Test CA"},
		NotBefore: notAfter.AddDate(-2, 0, 0), NotAfter: notAfter, IsCA: ca, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	if err != nil {
		t.Fatal(err)
	}
	return der
}

func jksWith(certs map[string][]byte, keyAlias string, keyChain []byte) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.BigEndian, v) }
	b.Write([]byte{0xFE, 0xED, 0xFE, 0xED})
	w(uint32(2))
	n := len(certs)
	if keyChain != nil {
		n++
	}
	w(uint32(n))
	if keyChain != nil {
		w(uint32(1))
		w(uint16(len(keyAlias)))
		b.WriteString(keyAlias)
		w(uint64(0))
		w(uint32(3))
		b.Write([]byte{1, 2, 3})
		w(uint32(1))
		w(uint16(5))
		b.WriteString("X.509")
		w(uint32(len(keyChain)))
		b.Write(keyChain)
	}
	for alias, der := range certs {
		w(uint32(2))
		w(uint16(len(alias)))
		b.WriteString(alias)
		w(uint64(0))
		w(uint16(5))
		b.WriteString("X.509")
		w(uint32(len(der)))
		b.Write(der)
	}
	b.Write(make([]byte, 20))
	return b.Bytes()
}

func TestCertificatesAreReadWithoutPersonalData(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	app := mkCert(t, pkix.Name{CommonName: "ORDERS APP", Organization: []string{"EXAMPLE ORG"}}, false, now.AddDate(1, 0, 0))
	person := mkCert(t, pkix.Name{CommonName: "DOE JANE - 12345678Z",
		ExtraNames: []pkix.AttributeTypeAndValue{{Type: asn1.ObjectIdentifier{2, 5, 4, 5}, Value: "IDCXX-12345678Z"}}}, false, now.AddDate(1, 0, 0))
	server := mkCert(t, pkix.Name{CommonName: "signgw.internal.example.org"}, false, now.AddDate(0, 0, -3))
	ca := mkCert(t, pkix.Name{CommonName: "Example Users CA"}, true, now.AddDate(5, 0, 0))
	identity := mkCert(t, pkix.Name{CommonName: "SELLO PRE ACME"}, false, now.AddDate(0, 0, 30))

	os.MkdirAll(filepath.Join(root, "config", "apps"), 0o755)
	os.WriteFile(filepath.Join(root, "config", "apps", "8f2a1c"), app, 0o644) // extensionless DER, as some gateways store them
	os.WriteFile(filepath.Join(root, "config", "person.cer"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: person}), 0o644)
	os.WriteFile(filepath.Join(root, "config", "trust.jks"), jksWith(map[string][]byte{"usersca": ca, "srv": server}, "", nil), 0o644)
	os.WriteFile(filepath.Join(root, "config", "client.jksOLD"), jksWith(nil, "seal", identity), 0o644)

	cat, _ := rules.Load(true, nil)
	sealer, _ := secure.FromPassphrase("a test passphrase that is long enough")
	f, err := Repo(root, "signgw", "", Options{MaxFileSize: 5 << 20, Catalog: cat, Config: &Config{}, Now: "2026-11-04T00:00:00Z", Workers: 1, Sealer: sealer})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range f.Certificates {
		if strings.Contains(c.Subject+c.Alias, "12345678") || strings.Contains(c.Subject, "DOE") {
			t.Errorf("personal data leaked: %+v", c)
		}
		if strings.Contains(c.Subject+c.Host, "example.org") {
			t.Errorf("host not encrypted: %+v", c)
		}
		switch {
		case c.Subject == "CN=ORDERS APP, O=EXAMPLE ORG" && c.Role == "file":
			got["app"] = true
		case c.Personal && c.Subject == "":
			got["person"] = true
		case c.Role == "trusted" && c.CA && c.Subject == "CN=Example Users CA":
			got["ca"] = true
		case c.Role == "trusted" && c.Host != "" && c.Status == "expired":
			got["server"] = true
		case c.Role == "identity" && c.Subject == "CN=SELLO PRE ACME" && c.Status == "expiring":
			got["identity"] = true
		}
	}
	for _, k := range []string{"app", "person", "ca", "server", "identity"} {
		if !got[k] {
			t.Errorf("missing %s in %+v", k, f.Certificates)
		}
	}
	exp := 0
	for _, fd := range f.Findings {
		if fd.RuleID == "SEC-CERT-EXPIRED" || fd.RuleID == "SEC-CERT-EXPIRING" {
			exp++
		}
	}
	if exp != 2 {
		t.Errorf("expiry findings: %d", exp)
	}
}

func TestIsPersonalUsesStandardsOnly(t *testing.T) {
	attr := func(oid asn1.ObjectIdentifier, v string) pkix.AttributeTypeAndValue {
		return pkix.AttributeTypeAndValue{Type: oid, Value: v}
	}
	for _, c := range []struct {
		name pkix.Name
		want bool
	}{
		{pkix.Name{CommonName: "Jane Doe", Names: []pkix.AttributeTypeAndValue{attr(oidGivenName, "Jane"), attr(oidSurname, "Doe")}}, true},
		{pkix.Name{CommonName: "x", Names: []pkix.AttributeTypeAndValue{attr(oidPseudonym, "jd42")}}, true},
		{pkix.Name{CommonName: "Jane Doe", Names: []pkix.AttributeTypeAndValue{attr(oidSerialNumber, "PNOSE-196501011234")}}, true},
		{pkix.Name{CommonName: "Max Mustermann", Names: []pkix.AttributeTypeAndValue{attr(oidSerialNumber, "PASDE-C01X00T47")}}, true},
		{pkix.Name{CommonName: "Jane Doe", Names: []pkix.AttributeTypeAndValue{attr(oidEmail, "jane@example.org")}}, true},
		{pkix.Name{CommonName: "SMITH JOHN 1234567"}, true},
		{pkix.Name{CommonName: "Example Org e-seal", Names: []pkix.AttributeTypeAndValue{attr(oidSerialNumber, "VATXX-B12345678")}}, false},
		{pkix.Name{CommonName: "api.example.org", Names: []pkix.AttributeTypeAndValue{attr(oidEmail, "ops@example.org")}}, false},
		{pkix.Name{CommonName: "ORDERS APP 2024"}, false},
		{pkix.Name{CommonName: "Example Users CA"}, false},
	} {
		if got := isPersonal(c.name); got != c.want {
			t.Errorf("%+v: %v want %v", c.name, got, c.want)
		}
	}
}

func TestCertificateSubjectsHideHostsAndCAsAreNotPersons(t *testing.T) {
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	ca := mkCert(t, pkix.Name{CommonName: "Example Root CA 12345678", Organization: []string{"Example Trust"}}, true, now.AddDate(5, 0, 0))
	srv := mkCert(t, pkix.Name{OrganizationalUnit: []string{"srv01.corp.example.net", "WEB SERVERS"}, Organization: []string{"EXAMPLE ORG"}}, false, now.AddDate(1, 0, 0))
	got := certificates("x", []rawCert{{der: ca, role: "trusted"}, {der: srv, alias: "srv01.corp.example.net", role: "trusted"}}, now)
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Personal || got[0].Subject == "" {
		t.Errorf("CA taken for a person: %+v", got[0])
	}
	if strings.Contains(got[1].Subject+got[1].Alias, "example.net") || got[1].Host != "srv01.corp.example.net" ||
		!strings.Contains(got[1].Subject, "OU=‹host›") || !strings.Contains(got[1].Subject, "WEB SERVERS") {
		t.Errorf("host not hidden: %+v", got[1])
	}
	for _, v := range []string{"S.G. DE TECNOLOGIAS", "AC Camerfirma S.A.", "version 2.3"} {
		if out, h := hideHosts(v); out != v || h != "" {
			t.Errorf("%q → %q (%q)", v, out, h)
		}
	}
}
