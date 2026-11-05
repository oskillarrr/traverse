// Package secure protects infrastructure names (host names) found in source
// code. Reports never contain them in clear text: each host is replaced by a
// stable pseudonym (keyed HMAC), and the real value is stored encrypted
// (AES-256-GCM) in a separate vault that only key holders can open.
//
// Only the Go standard library is used (crypto/hkdf, crypto/pbkdf2, AES-GCM).
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

const (
	keySize          = 32
	pbkdf2Iterations = 600_000 // OWASP 2023 recommendation for PBKDF2-HMAC-SHA256
	vaultFormat      = "traverse-host-vault"
)

// Sealer turns host names into pseudonyms and keeps them encrypted.
type Sealer struct {
	mac, enc []byte
	gcm      cipher.AEAD
	mu       sync.Mutex
	entries  map[string]entry
}

type entry struct {
	Token      string `json:"token"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// Vault is the encrypted host table written next to the reports.
type Vault struct {
	Format   string  `json:"format"`
	Version  int     `json:"version"`
	Cipher   string  `json:"cipher"`
	KeyCheck string  `json:"keyCheck"`
	Entries  []entry `json:"entries"`
}

// GenerateKey returns a new random key, base64-encoded, for a key file.
func GenerateKey() (string, error) {
	k := make([]byte, keySize)
	if _, err := rand.Read(k); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(k), nil
}

// FromKeyFile loads a key created with GenerateKey (recommended).
func FromKeyFile(path string) (*Sealer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(k) != keySize {
		return nil, errors.New("invalid key file: expected 32 random bytes in base64 (use `traverse keygen`)")
	}
	return newSealer(k)
}

// FromPassphrase derives the key with PBKDF2-HMAC-SHA256. The salt is fixed
// so that pseudonyms stay comparable across runs and teams sharing the
// passphrase; prefer a key file for stronger protection.
func FromPassphrase(p string) (*Sealer, error) {
	if len(p) < 12 {
		return nil, errors.New("passphrase must have at least 12 characters")
	}
	k, err := pbkdf2.Key(sha256.New, p, []byte("traverse/hosts/passphrase/v1"), pbkdf2Iterations, keySize)
	if err != nil {
		return nil, err
	}
	return newSealer(k)
}

func newSealer(master []byte) (*Sealer, error) {
	mac, err := hkdf.Key(sha256.New, master, nil, "traverse hosts pseudonym v1", keySize)
	if err != nil {
		return nil, err
	}
	enc, err := hkdf.Key(sha256.New, master, nil, "traverse hosts encryption v1", keySize)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(enc)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{mac: mac, enc: enc, gcm: gcm, entries: map[string]entry{}}, nil
}

func (s *Sealer) sum(key []byte, v string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(v))
	return h.Sum(nil)
}

// Token returns the pseudonym of a host and records it, encrypted, in the vault.
// The same host always yields the same token for the same key.
func (s *Sealer) Token(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	tok := "host-" + hex.EncodeToString(s.sum(s.mac, host))[:12]
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[tok]; !ok {
		// Deterministic nonce (SIV-like): equal hosts give equal ciphertexts,
		// which reveals nothing beyond what the token already shows, and keeps
		// outputs reproducible.
		nonce := s.sum(s.enc, "nonce|"+host)[:s.gcm.NonceSize()]
		ct := s.gcm.Seal(nil, nonce, []byte(host), []byte(tok))
		s.entries[tok] = entry{Token: tok, Nonce: base64.StdEncoding.EncodeToString(nonce), Ciphertext: base64.StdEncoding.EncodeToString(ct)}
	}
	return tok
}

func (s *Sealer) check() string { return hex.EncodeToString(s.sum(s.mac, "traverse key check"))[:16] }

// Vault returns the encrypted host table.
func (s *Sealer) Vault() Vault {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := Vault{Format: vaultFormat, Version: 1, Cipher: "AES-256-GCM (HKDF-SHA256 keys, HMAC-SHA256 pseudonyms)", KeyCheck: s.check()}
	for _, e := range s.entries {
		v.Entries = append(v.Entries, e)
	}
	sort.Slice(v.Entries, func(i, j int) bool { return v.Entries[i].Token < v.Entries[j].Token })
	if v.Entries == nil {
		v.Entries = []entry{}
	}
	return v
}

// Merge adds the entries of another sealer created with the same key.
func (s *Sealer) Merge(o *Sealer) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range o.entries {
		s.entries[k] = e
	}
}

// Reveal decrypts a vault file. It fails if the key does not match.
func (s *Sealer) Reveal(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v Vault
	if err := json.Unmarshal(b, &v); err != nil || v.Format != vaultFormat {
		return nil, fmt.Errorf("%s is not an Traverse host vault", path)
	}
	if !hmac.Equal([]byte(v.KeyCheck), []byte(s.check())) {
		return nil, errors.New("wrong key for this vault")
	}
	out := map[string]string{}
	for _, e := range v.Entries {
		nonce, err1 := base64.StdEncoding.DecodeString(e.Nonce)
		ct, err2 := base64.StdEncoding.DecodeString(e.Ciphertext)
		if err1 != nil || err2 != nil {
			return nil, errors.New("corrupted vault entry " + e.Token)
		}
		pt, err := s.gcm.Open(nil, nonce, ct, []byte(e.Token))
		if err != nil {
			return nil, errors.New("cannot decrypt entry " + e.Token + " (tampered or wrong key)")
		}
		out[e.Token] = string(pt)
	}
	return out, nil
}
