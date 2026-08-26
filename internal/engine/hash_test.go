package engine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// AI generated tests
func TestHashers(t *testing.T) {
	if HashBytes([]byte("a")) == HashBytes([]byte("b")) {
		t.Fatal("HashBytes collided")
	}

	dir := t.TempDir()
	f := filepath.Join(dir, "x.txt")
	os.WriteFile(f, []byte("one"), 0644)

	h1, err := HashFile(f)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(f, []byte("two"), 0644)
	h2, _ := HashFile(f)
	if h1 == h2 {
		t.Fatal("HashFile didn't change after an edit")
	}
	t.Logf("HashFile changed on edit:  %s.. -> %s..", h1[:8], h2[:8])

	if _, err := HashFile(filepath.Join(dir, "gone.txt")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing file should wrap fs.ErrNotExist, got %v", err)
	}

	d1, _ := HashDir(dir)
	os.WriteFile(filepath.Join(dir, "y.txt"), []byte("new"), 0644)
	d2, _ := HashDir(dir)
	if d1 == d2 {
		t.Fatal("HashDir didn't change after a new file")
	}
	t.Logf("HashDir changed on add:    %s.. -> %s..", d1[:8], d2[:8])
}
