//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteServicePortFileMode0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "port.txt")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	const port = 12345
	const secret uint64 = 0xabc
	if err := writeServicePortFile(path, port, secret); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("mode = %04o, want 0600", perm)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%d:%x", port, secret)
	if string(body) != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}
