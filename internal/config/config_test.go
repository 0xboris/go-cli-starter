package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomic_preservesModeAndSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.yml")
	if err := os.WriteFile(target, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unsupported:", err)
	}

	if err := WriteFileAtomic(link, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was replaced by a regular file")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}

func TestDirPrecedence(t *testing.T) {
	t.Setenv(EnvConfigDir, "/custom")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if d, _ := Dir(); d != "/custom" {
		t.Errorf("TOOL_CONFIG_DIR should win, got %q", d)
	}
	t.Setenv(EnvConfigDir, "")
	if d, _ := Dir(); d != filepath.Join("/xdg", "tool") {
		t.Errorf("XDG_CONFIG_HOME should be used, got %q", d)
	}
}

func TestSetValidatesAllowedValues(t *testing.T) {
	c := NewFromMap(map[string]string{})
	if err := c.Set("prompt", "sometimes"); err == nil || !strings.Contains(err.Error(), "valid values") {
		t.Errorf("expected allowed-values error, got %v", err)
	}
	if err := c.Set("nope", "x"); err == nil {
		t.Error("expected unknown key error")
	}
	if err := c.Set("prompt", "disabled"); err != nil || c.GetOrDefault("prompt") != "disabled" {
		t.Errorf("set failed: %v", err)
	}
}
