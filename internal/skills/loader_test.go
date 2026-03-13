package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
)

func TestSubconscious_MergeFileAndDB(t *testing.T) {
	dir := t.TempDir()
	trusted := []string{dir}

	t.Run("MD only", func(t *testing.T) {
		skillDir := filepath.Join(dir, "md-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		md := `---
name: md-skill
description: From markdown
version: 0.1.0
---

## Instructions

Do X when user asks.
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(md), 0644); err != nil {
			t.Fatal(err)
		}
		loader := &SkillLoader{TrustedRoots: trusted}
		row := state.SkillRow{Name: "md-skill", Path: skillDir, Enabled: true}
		s := loader.Load(row)
		if s == nil {
			t.Fatal("expected skill")
		}
		if s.Description != "From markdown" {
			t.Errorf("desc: got %q", s.Description)
		}
		if s.Instructions != "## Instructions\n\nDo X when user asks." {
			t.Errorf("instructions: got %q", s.Instructions)
		}
	})

	t.Run("TOML + instructions.md", func(t *testing.T) {
		skillDir := filepath.Join(dir, "toml-inst-md")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		toml := `[skill]
name = "toml-inst-md"
description = "From TOML"
version = "0.1.0"
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.toml"), []byte(toml), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "instructions.md"), []byte("Use instructions.md content."), 0644); err != nil {
			t.Fatal(err)
		}
		loader := &SkillLoader{TrustedRoots: trusted}
		row := state.SkillRow{Name: "toml-inst-md", Path: skillDir, Enabled: true}
		s := loader.Load(row)
		if s == nil {
			t.Fatal("expected skill")
		}
		if s.Description != "From TOML" {
			t.Errorf("desc: got %q", s.Description)
		}
		if s.Instructions != "Use instructions.md content." {
			t.Errorf("instructions: got %q", s.Instructions)
		}
	})

	t.Run("TOML + [skill.instructions]", func(t *testing.T) {
		skillDir := filepath.Join(dir, "toml-inline")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		toml := `[skill]
name = "toml-inline"
description = "Inline instructions"

[skill.instructions]
text = """
When the user asks to DCA, use the following procedure:
1. Check USDC balance
2. Execute
"""
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.toml"), []byte(toml), 0644); err != nil {
			t.Fatal(err)
		}
		loader := &SkillLoader{TrustedRoots: trusted}
		row := state.SkillRow{Name: "toml-inline", Path: skillDir, Enabled: true}
		s := loader.Load(row)
		if s == nil {
			t.Fatal("expected skill")
		}
		if s.Description != "Inline instructions" {
			t.Errorf("desc: got %q", s.Description)
		}
		if s.Instructions != "When the user asks to DCA, use the following procedure:\n1. Check USDC balance\n2. Execute" {
			t.Errorf("instructions: got %q", s.Instructions)
		}
	})

	t.Run("bad path - not under trusted root", func(t *testing.T) {
		otherDir := t.TempDir() // outside trusted
		skillDir := filepath.Join(otherDir, "bad")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: bad\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
		loader := &SkillLoader{TrustedRoots: trusted}
		row := state.SkillRow{Name: "bad", Path: skillDir, Enabled: true, Instructions: "DB fallback"}
		s := loader.Load(row)
		if s == nil {
			t.Fatal("expected skill (graceful: use DB)")
		}
		// Should use DB instructions, not file (path rejected)
		if s.Instructions != "DB fallback" {
			t.Errorf("should use DB instructions: got %q", s.Instructions)
		}
	})

	t.Run("DB-only no path", func(t *testing.T) {
		loader := &SkillLoader{TrustedRoots: trusted}
		row := state.SkillRow{Name: "builtin", Path: "", Instructions: "Builtin instructions", Enabled: true}
		s := loader.Load(row)
		if s == nil {
			t.Fatal("expected skill")
		}
		if s.Instructions != "Builtin instructions" {
			t.Errorf("got %q", s.Instructions)
		}
	})
}
