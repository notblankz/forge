package engine

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// HashBytes returns the hex-encoded SHA-256 of b.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

// HashFile returns the hex-encoded SHA-256 of the file at path.
// A missing file returns an error wrapping fs.ErrNotExist, which Run reads
// as a deletion
func HashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return HashBytes(b), nil
}

// HashDir fingerprints a directory from its files' path, size, and mtime
// (metadata only - it never reads file contents). A missing directory returns
// an error wrapping fs.ErrNotExist.
func HashDir(dir string) (string, error) {
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "%s|%d|%d\n", path, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", err
	}
	return HashBytes([]byte(b.String())), nil
}
