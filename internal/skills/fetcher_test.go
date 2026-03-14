package skills

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipToDir(t *testing.T) {
	// Create a minimal zip with SKILL.md
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fw, _ := w.Create("SKILL.md")
	_, _ = fw.Write([]byte("---\nname: test\n---\nTest skill."))
	_ = w.Close()
	zipData := buf.Bytes()

	dir := t.TempDir()
	root, err := ExtractZipToDir(zipData, dir)
	if err != nil {
		t.Fatalf("ExtractZipToDir: %v", err)
	}
	if root != dir {
		t.Errorf("expected root %q, got %q", dir, root)
	}
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("SKILL.md not found at %s", skillPath)
	}
}

func TestExtractZipToDir_WithPrefix(t *testing.T) {
	// Zip with single root folder
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fw, _ := w.Create("my-skill/SKILL.md")
	_, _ = fw.Write([]byte("---\nname: my-skill\n---\nContent."))
	_ = w.Close()
	zipData := buf.Bytes()

	dir := t.TempDir()
	root, err := ExtractZipToDir(zipData, dir)
	if err != nil {
		t.Fatalf("ExtractZipToDir: %v", err)
	}
	skillPath := filepath.Join(dir, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("SKILL.md not found at %s", skillPath)
	}
	if root != dir {
		t.Errorf("expected root %q, got %q", dir, root)
	}
}

func TestExtractZipToDir_NoSkill(t *testing.T) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fw, _ := w.Create("README.md")
	_, _ = fw.Write([]byte("No skill here"))
	_ = w.Close()

	_, err := ExtractZipToDir(buf.Bytes(), t.TempDir())
	if err == nil {
		t.Error("expected error for zip without SKILL.md/SKILL.toml")
	}
}
