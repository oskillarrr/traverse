package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The getting-started guide creates the team key in a folder that may not exist yet.
func TestKeygenCreatesFolderAndRefusesOverwrite(t *testing.T) {
	out := filepath.Join(t.TempDir(), "keys", "team.key")
	if err := cmdKeygen([]string{"--out", out}); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(out); err != nil || st.Size() == 0 {
		t.Fatalf("key not written: %v", err)
	}
	if err := cmdKeygen([]string{"--out", out}); err == nil {
		t.Error("an existing key must never be overwritten")
	}
}
