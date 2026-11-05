package scan

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"encoding/pem"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/oskillarrr/traverse/internal/model"
)

// Public certificates say who a system trusts and who it is: the CAs of its
// truststores, the peers registered for mTLS (a signature gateway keeps the certificate of
// every client application) and its own client/server identity. Only public
// data is read. A natural person's certificate keeps no subject at all.

var certExt = map[string]bool{".cer": true, ".crt": true, ".der": true, ".pem": true, ".jks": true, ".jceks": true,
	".keystore": true, ".truststore": true, ".ts": false}

// certCandidate tells files worth inspecting: certificate and keystore
// extensions (also with backup suffixes) and small files without extension.
func certCandidate(rel string, size int64) bool {
	base := strings.ToLower(path.Base(rel))
	ext := path.Ext(base)
	for e := range certExt {
		if certExt[e] && (ext == e || strings.Contains(base, e+".") || strings.HasSuffix(base, e+"old") || strings.HasSuffix(base, e+"_old") || strings.HasSuffix(base, e+".bak")) {
			return size < 4<<20
		}
	}
	return ext == "" && size > 200 && size < 64<<10
}

type rawCert struct {
	der   []byte
	alias string
	role  string
}

// extractCertificates returns the certificates a file holds.
func extractCertificates(b []byte) []rawCert {
	switch {
	case bytes.HasPrefix(b, []byte{0xFE, 0xED, 0xFE, 0xED}) || bytes.HasPrefix(b, []byte{0xCE, 0xCE, 0xCE, 0xCE}):
		return jksCertificates(b)
	case bytes.Contains(b, []byte("-----BEGIN CERTIFICATE-----")):
		var out []rawCert
		rest := b
		for {
			var blk *pem.Block
			blk, rest = pem.Decode(rest)
			if blk == nil {
				break
			}
			if blk.Type == "CERTIFICATE" {
				out = append(out, rawCert{der: blk.Bytes, role: "file"})
			}
		}
		return out
	case len(b) > 4 && b[0] == 0x30 && b[1] == 0x82:
		if _, err := x509.ParseCertificate(b); err == nil {
			return []rawCert{{der: b, role: "file"}}
		}
	}
	return nil
}

// jksCertificates walks a JKS/JCEKS store: trusted entries hold one
// certificate; key entries keep their certificate chain in clear.
func jksCertificates(b []byte) []rawCert {
	r := bytes.NewReader(b[8:])
	var count uint32
	if binary.Read(r, binary.BigEndian, &count) != nil || count > 10000 {
		return nil
	}
	readUTF := func() (string, bool) {
		var l uint16
		if binary.Read(r, binary.BigEndian, &l) != nil || int(l) > r.Len() {
			return "", false
		}
		s := make([]byte, l)
		_, err := io.ReadFull(r, s)
		return string(s), err == nil
	}
	readBlob := func() ([]byte, bool) {
		var l uint32
		if binary.Read(r, binary.BigEndian, &l) != nil || int64(l) > int64(r.Len()) {
			return nil, false
		}
		s := make([]byte, l)
		_, err := io.ReadFull(r, s)
		return s, err == nil
	}
	var out []rawCert
	for i := uint32(0); i < count; i++ {
		var tag uint32
		if binary.Read(r, binary.BigEndian, &tag) != nil {
			return out
		}
		alias, ok := readUTF()
		if !ok {
			return out
		}
		if _, err := r.Seek(8, io.SeekCurrent); err != nil { // timestamp
			return out
		}
		switch tag {
		case 1: // private key entry: encrypted key, then the clear certificate chain
			if _, ok := readBlob(); !ok {
				return out
			}
			var n uint32
			if binary.Read(r, binary.BigEndian, &n) != nil || n > 20 {
				return out
			}
			for j := uint32(0); j < n; j++ {
				if _, ok := readUTF(); !ok {
					return out
				}
				der, ok := readBlob()
				if !ok {
					return out
				}
				if j == 0 { // the leaf is the identity; the rest are its CAs
					out = append(out, rawCert{der: der, alias: alias, role: "identity"})
				}
			}
		case 2:
			if _, ok := readUTF(); !ok {
				return out
			}
			der, ok := readBlob()
			if !ok {
				return out
			}
			out = append(out, rawCert{der: der, alias: alias, role: "trusted"})
		default: // secret keys (JCEKS) have nothing public to read
			return out
		}
	}
	return out
}

var (
	oidGivenName    = asn1.ObjectIdentifier{2, 5, 4, 42}
	oidSurname      = asn1.ObjectIdentifier{2, 5, 4, 4}
	oidSerialNumber = asn1.ObjectIdentifier{2, 5, 4, 5}
	oidPseudonym    = asn1.ObjectIdentifier{2, 5, 4, 65}
	oidEmail        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}
	// ETSI EN 319 412-1 natural-person semantics identifiers: passport (PAS),
	// identity card (IDC), national personal number (PNO), tax reference
	// (TAX/TIN), followed by an ISO 3166 country code: "IDCXX-…", "PNOXX:…".
	rePersonSemanticsID = regexp.MustCompile(`(?i)^(PAS|IDC|PNO|TAX|TIN)[A-Z]{2}[-:]`)
	// a word followed by a digit-heavy token: most personal ID numbers in the
	// world are, and hiding a subject too many is the safe mistake
	rePersonIDInCN = regexp.MustCompile(`(?i)\p{L}{2,}.*(^|[^0-9])[0-9]{6,}[A-Z]?([^0-9]|$)`)
	reHostCN       = regexp.MustCompile(`^(\*\.)?[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+$`)
	// a host name inside another subject attribute ("OU=srv01.corp.local, …")
	reHostInValue = regexp.MustCompile(`(?i)\b(\*\.)?[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}\b`)
)

// isPersonal recognizes a natural person's certificate (citizen, employee,
// representative) from standard X.509 attributes, never from one country's
// conventions: given name, surname or pseudonym (RFC 5280 / X.520), a
// natural-person semantics identifier (ETSI EN 319 412-1), an e-mail address
// in a subject that is not a host, or a common name carrying an ID number.
func isPersonal(n pkix.Name) bool {
	host := reHostCN.MatchString(n.CommonName)
	for _, a := range n.Names {
		switch {
		case a.Type.Equal(oidGivenName), a.Type.Equal(oidSurname), a.Type.Equal(oidPseudonym):
			return true
		case a.Type.Equal(oidSerialNumber):
			if s, ok := a.Value.(string); ok && rePersonSemanticsID.MatchString(s) {
				return true
			}
		case a.Type.Equal(oidEmail):
			if !host {
				return true
			}
		}
	}
	return !host && (rePersonSemanticsID.MatchString(n.CommonName) || rePersonIDInCN.MatchString(n.CommonName))
}

func describe(n pkix.Name) string {
	var parts []string
	if n.CommonName != "" {
		parts = append(parts, "CN="+n.CommonName)
	}
	for _, ou := range n.OrganizationalUnit {
		parts = append(parts, "OU="+ou)
	}
	for _, o := range n.Organization {
		parts = append(parts, "O="+o)
	}
	return strings.Join(parts, ", ")
}

// hideHosts replaces the host names found inside a subject or alias by a
// placeholder and returns the first one, to be sealed like any other host.
// Certificate authorities are public and keep their subject as is.
func hideHosts(v string) (string, string) {
	first := ""
	out := reHostInValue.ReplaceAllStringFunc(v, func(h string) string {
		if first == "" {
			first = strings.TrimPrefix(h, "*.")
		}
		return "‹host›"
	})
	return out, first
}

// certificates turns the raw certificates of a file into facts, relative to
// the scan date for the expiry status.
func certificates(rel string, raws []rawCert, now time.Time) []model.Certificate {
	var out []model.Certificate
	for _, rc := range raws {
		c, err := x509.ParseCertificate(rc.der)
		if err != nil {
			continue
		}
		mc := model.Certificate{File: rel, Alias: rc.alias, Role: rc.role, CA: c.IsCA,
			NotAfter: c.NotAfter.UTC().Format("2006-01-02"), Issuer: describeIssuer(c.Issuer)}
		switch {
		case !c.IsCA && isPersonal(c.Subject): // an authority is never a natural person
			mc.Personal = true
			mc.Alias = "" // aliases of personal certificates often carry the name
		case reHostCN.MatchString(c.Subject.CommonName):
			mc.Host = strings.TrimPrefix(c.Subject.CommonName, "*.")
			n := c.Subject
			n.CommonName = ""
			mc.Subject = describe(n)
		default:
			mc.Subject = describe(c.Subject)
		}
		if !c.IsCA {
			var h1, h2 string
			mc.Subject, h1 = hideHosts(mc.Subject)
			mc.Alias, h2 = hideHosts(mc.Alias)
			if mc.Host == "" {
				mc.Host = h1
				if mc.Host == "" {
					mc.Host = h2
				}
			}
		}
		if !now.IsZero() {
			switch {
			case c.NotAfter.Before(now):
				mc.Status = "expired"
			case c.NotAfter.Before(now.AddDate(0, 0, 90)):
				mc.Status = "expiring"
			}
		}
		out = append(out, mc)
	}
	return out
}

func describeIssuer(n pkix.Name) string {
	if n.CommonName != "" {
		return n.CommonName
	}
	if len(n.Organization) > 0 {
		return n.Organization[0]
	}
	return ""
}
