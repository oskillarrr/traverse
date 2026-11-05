package scan

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// Key containers committed to a repository are only a serious finding when
// they hold private (or secret) keys. The entry tags of JKS/JCEKS stores and
// the bag types of PKCS#12 files are not encrypted, so this can be told
// without any password. Nothing is decrypted and no key material is read.

type keyContent int

const (
	keyUnknown keyContent = iota
	keyPrivate
	keyPublicOnly
)

// pkcs8ShroudedKeyBag and keyBag OIDs (1.2.840.113549.1.12.10.1.{2,1})
var (
	oidShroudedKeyBag = []byte{0x06, 0x0B, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x0C, 0x0A, 0x01, 0x02}
	oidKeyBag         = []byte{0x06, 0x0B, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x0C, 0x0A, 0x01, 0x01}
)

func inspectKeyContainer(abs string) keyContent {
	fh, err := os.Open(abs)
	if err != nil {
		return keyUnknown
	}
	defer fh.Close()
	b, err := io.ReadAll(io.LimitReader(fh, 4<<20))
	if err != nil || len(b) < 8 {
		return keyUnknown
	}
	switch {
	case bytes.HasPrefix(b, []byte{0xFE, 0xED, 0xFE, 0xED}) || bytes.HasPrefix(b, []byte{0xCE, 0xCE, 0xCE, 0xCE}):
		return jksContent(b)
	case bytes.Contains(b, []byte("-----BEGIN")):
		if bytes.Contains(b, []byte("PRIVATE KEY-----")) {
			return keyPrivate
		}
		return keyPublicOnly
	case b[0] == 0x30: // DER: PKCS#12 or a bare certificate
		if bytes.Contains(b, oidShroudedKeyBag) || bytes.Contains(b, oidKeyBag) {
			return keyPrivate
		}
	}
	return keyUnknown
}

// jksContent walks the entries of a JKS/JCEKS store reading only their tags.
func jksContent(b []byte) keyContent {
	r := bytes.NewReader(b[8:])
	var count uint32
	if binary.Read(r, binary.BigEndian, &count) != nil || count > 10000 {
		return keyUnknown
	}
	skip := func(n int64) bool {
		if n < 0 || n > int64(r.Len()) {
			return false
		}
		_, err := r.Seek(n, io.SeekCurrent)
		return err == nil
	}
	readUTF := func() bool {
		var l uint16
		return binary.Read(r, binary.BigEndian, &l) == nil && skip(int64(l))
	}
	readBlob := func() bool {
		var l uint32
		return binary.Read(r, binary.BigEndian, &l) == nil && skip(int64(l))
	}
	readCert := func() bool { return readUTF() && readBlob() }
	for i := uint32(0); i < count; i++ {
		var tag uint32
		if binary.Read(r, binary.BigEndian, &tag) != nil {
			return keyUnknown
		}
		switch tag {
		case 1, 3: // private key / secret key entry
			return keyPrivate
		case 2: // trusted certificate
			if !readUTF() || !skip(8) || !readCert() {
				return keyUnknown
			}
		default:
			return keyUnknown
		}
	}
	return keyPublicOnly
}

// refineKeystoreFinding adjusts the generic key-container finding to what
// the file actually holds.
func refineKeystoreFinding(abs, rel string) (severity, title, message string, ok bool) {
	low := strings.ToLower(rel)
	switch inspectKeyContainer(abs) {
	case keyPrivate:
		return "high", "Private key committed (keystore / PKCS#12 / PEM)", "The file holds at least one private or secret key: rotate it and move it to a secret store.", true
	case keyPublicOnly:
		return "info", "Truststore / public certificates committed", "Only public certificates: not a secret, but it documents which CAs or peers are trusted.", true
	}
	if strings.Contains(low, ".p12") || strings.Contains(low, ".pfx") {
		return "high", "PKCS#12 key store committed", "PKCS#12 files normally hold a private key; its content could not be confirmed without the password.", true
	}
	return "", "", "", false
}
