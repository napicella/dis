package tools

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteFileAtomic keeps an existing file's mode and gives a new one perm.
func TestWriteFileAtomicMode(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old")
	if err := os.WriteFile(old, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	newFile := filepath.Join(dir, "sub", "new")
	for _, f := range []string{old, newFile} {
		if err := WriteFileAtomic(f, []byte("data"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	for f, want := range map[string]os.FileMode{old: 0o600, newFile: 0o640} {
		if fi, err := os.Stat(f); err != nil || fi.Mode().Perm() != want || readRC(t, f) != "data" {
			t.Errorf("%s: %v, %v, want mode %v", f, fi, err, want)
		}
	}
}

// Every kind of TOML value keep writes reads back as the same value and type.
