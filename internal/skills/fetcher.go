package skills

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZipToDir extracts a zip archive to targetDir. The zip may have a single
// root folder (e.g. slug-1.0.0/) or flat files. Strips one level if all entries
// share a common prefix. Returns the actual skill root (dir containing SKILL.md or SKILL.toml).
func ExtractZipToDir(zipData []byte, targetDir string) (skillRoot string, err error) {
	if len(zipData) == 0 {
		return "", fmt.Errorf("empty zip")
	}
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", fmt.Errorf("invalid zip: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	// Check if all entries share a common prefix (single root folder)
	var prefix string
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.HasSuffix(name, "/") {
			continue
		}
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			if prefix == "" {
				prefix = parts[0] + "/"
			} else if !strings.HasPrefix(name, prefix) {
				prefix = ""
				break
			}
		}
	}
	hasSkill := false
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.HasSuffix(name, "/") {
			continue
		}
		rel := name
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			rel = strings.TrimPrefix(name, prefix)
		}
		// Reject path traversal
		if strings.Contains(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(rel))
		if err := extractFile(f, dest); err != nil {
			return "", err
		}
		base := filepath.Base(rel)
		if base == skillMd || base == skillToml {
			hasSkill = true
			skillRoot = filepath.Dir(dest)
		}
	}
	if !hasSkill {
		return "", fmt.Errorf("zip does not contain SKILL.md or SKILL.toml")
	}
	if skillRoot == "" {
		skillRoot = targetDir
	}
	return skillRoot, nil
}

func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("open zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}
