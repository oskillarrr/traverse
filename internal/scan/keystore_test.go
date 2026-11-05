package scan

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func jks(tags ...uint32) []byte {
	var b bytes.Buffer
	w := func(v any) { binary.Write(&b, binary.BigEndian, v) }
	b.Write([]byte{0xFE, 0xED, 0xFE, 0xED})
	w(uint32(2))
	w(uint32(len(tags)))
	for _, t := range tags {
		w(t)
		w(uint16(2))
		b.WriteString("ca")
		w(uint64(0))
		if t == 2 {
			w(uint16(5))
			b.WriteString("X.509")
			w(uint32(3))
			b.Write([]byte{1, 2, 3})
		} else {
			w(uint32(4))
			b.Write([]byte{9, 9, 9, 9})
		}
	}
	b.Write(make([]byte, 20))
	return b.Bytes()
}

func TestKeyContainerContent(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		data []byte
		want keyContent
	}{
		"trust.jks":  {jks(2, 2), keyPublicOnly},
		"ks.jks":     {jks(2, 1), keyPrivate},
		"key.pem":    {[]byte("-----BEGIN RSA PRIVATE KEY-----\nAAA\n-----END RSA PRIVATE KEY-----"), keyPrivate},
		"cert.pem":   {[]byte("-----BEGIN CERTIFICATE-----\nAAA\n-----END CERTIFICATE-----"), keyPublicOnly},
		"sello.p12":  {append([]byte{0x30, 0x82, 0x01, 0x00}, oidShroudedKeyBag...), keyPrivate},
		"random.key": {[]byte("not a key at all"), keyUnknown},
	}
	for name, c := range cases {
		p := filepath.Join(dir, name)
		os.WriteFile(p, c.data, 0o600)
		if got := inspectKeyContainer(p); got != c.want {
			t.Errorf("%s: got %v want %v", name, got, c.want)
		}
	}
	if sev, _, _, _ := refineKeystoreFinding(filepath.Join(dir, "trust.jks"), "x/trust.jks"); sev != "info" {
		t.Errorf("truststore severity %s", sev)
	}
}
