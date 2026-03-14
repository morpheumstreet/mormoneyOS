// Package skills implements the Subconscious — runtime merge of file + DB into unified Skill.
package skills

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/morpheumlabs/mormoneyos-go/internal/state"
	"github.com/morpheumlabs/mormoneyos-go/internal/types"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const (
	skillToml = "SKILL.toml"
	skillMd   = "SKILL.md"
	instMd   = "instructions.md"
)

// Skill is the unified in-memory representation (file + DB merged).
type Skill struct {
	Name         string
	Description  string
	Instructions string
	Source       string
	Path         string
	Enabled      bool
}

// Subconscious loads and merges skills from DB + filesystem (runtime merge layer).
type Subconscious struct {
	TrustedRoots []string
	DescUpdater  func(name, description string) error // optional; for one-time DB sync
	Log          *slog.Logger
}

// Load reads a row and merges with file content when path is set.
// Graceful: on error, logs and returns nil (never fail prompt build).
func (sub *Subconscious) Load(row state.SkillRow) *Skill {
	if !row.Enabled {
		return nil
	}
	s := &Skill{
		Name:         row.Name,
		Description:  row.Description,
		Instructions: row.Instructions,
		Source:       row.Source,
		Path:         row.Path,
		Enabled:      row.Enabled,
	}
	if row.Path == "" {
		return s
	}
	dir := filepath.Clean(row.Path)
	if strings.HasSuffix(dir, skillMd) || strings.HasSuffix(dir, skillToml) {
		dir = filepath.Dir(dir)
	}
	// Resolve symlinks and validate under trusted root
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		if sub.Log != nil {
			sub.Log.Warn("skill load: path resolve failed", "name", row.Name, "path", dir, "err", err)
		}
		return s
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		if sub.Log != nil {
			sub.Log.Warn("skill load: abs failed", "name", row.Name, "err", err)
		}
		return s
	}
	allowed := false
	for _, root := range sub.TrustedRoots {
		r := filepath.Clean(root)
		if r == "" {
			continue
		}
		if strings.HasPrefix(r, "~") {
			home, _ := os.UserHomeDir()
			r = home + strings.TrimPrefix(r, "~")
		}
		absRoot, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		// Resolve symlinks so /var/... and /private/var/... match
		if res, err := filepath.EvalSymlinks(absRoot); err == nil {
			absRoot = res
		}
		if abs == absRoot || strings.HasPrefix(abs, absRoot+string(filepath.Separator)) {
			allowed = true
			break
		}
	}
	if !allowed {
		if sub.Log != nil {
			sub.Log.Warn("skill load: path not under trusted root", "name", row.Name, "path", abs)
		}
		return s
	}
	// Load from file: SKILL.toml first, then SKILL.md
	fileDesc, fileInst, err := sub.loadFromDir(abs)
	if err != nil {
		if sub.Log != nil {
			sub.Log.Warn("skill load failed", "name", row.Name, "path", abs, "err", err)
		}
		return s
	}
	if fileDesc != "" {
		s.Description = fileDesc
		if sub.DescUpdater != nil {
			_ = sub.DescUpdater(row.Name, fileDesc)
		}
	}
	if fileInst != "" {
		s.Instructions = fileInst
	}
	return s
}

// loadFromDir reads SKILL.toml or SKILL.md from dir.
func (sub *Subconscious) loadFromDir(dir string) (description, instructions string, err error) {
	if _, err := os.Stat(dir); err != nil {
		return "", "", err
	}
	// Precedence: SKILL.toml first, then SKILL.md
	tomlPath := filepath.Join(dir, skillToml)
	mdPath := filepath.Join(dir, skillMd)
	if _, err := os.Stat(tomlPath); err == nil {
		return sub.loadFromTOML(tomlPath, dir)
	}
	if _, err := os.Stat(mdPath); err == nil {
		return sub.loadFromMD(mdPath)
	}
	return "", "", nil
}

func (sub *Subconscious) loadFromTOML(tomlPath, dir string) (description, instructions string, err error) {
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return "", "", err
	}
	var meta struct {
		Skill struct {
			Name        string `toml:"name"`
			Description string `toml:"description"`
			// [skill.instructions] sub-table with text
			Instructions struct {
				Text string `toml:"text"`
			} `toml:"instructions"`
		} `toml:"skill"`
	}
	if err := toml.Unmarshal(data, &meta); err != nil {
		return "", "", err
	}
	description = meta.Skill.Description
	// Instructions: 1) instructions.md sibling, 2) [skill.instructions].text
	instPath := filepath.Join(dir, instMd)
	if data, err := os.ReadFile(instPath); err == nil {
		instructions = strings.TrimSpace(string(data))
	}
	if instructions == "" && meta.Skill.Instructions.Text != "" {
		instructions = strings.TrimSpace(meta.Skill.Instructions.Text)
	}
	return description, instructions, nil
}

func (sub *Subconscious) loadFromMD(mdPath string) (description, instructions string, err error) {
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return "", "", err
	}
	content := string(data)
	// Parse YAML frontmatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) >= 3 {
		var fm struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Version     string `yaml:"version"`
		}
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err == nil {
			description = fm.Description
		}
		instructions = strings.TrimSpace(parts[2])
	}
	return description, instructions, nil
}

// LoadAll loads all rows and returns merged skills (graceful degradation).
func (sub *Subconscious) LoadAll(rows []state.SkillRow) []*Skill {
	var out []*Skill
	for _, row := range rows {
		s := sub.Load(row)
		if s != nil {
			out = append(out, s)
		}
	}
	return out
}

// SkillRowStore provides GetSkillRows.
type SkillRowStore interface {
	GetSkillRows() ([]state.SkillRow, error)
}

// LoadAllFromStore loads all enabled skills from store (for prompt builder).
func LoadAllFromStore(store SkillRowStore, cfg *types.SkillsConfig) []*Skill {
	if store == nil {
		return nil
	}
	rows, err := store.GetSkillRows()
	if err != nil || len(rows) == 0 {
		return nil
	}
	trusted := []string{"~/.automaton/skills"}
	if cfg != nil && len(cfg.TrustedRoots) > 0 {
		trusted = cfg.TrustedRoots
	}
	sub := &Subconscious{TrustedRoots: trusted, Log: slog.Default()}
	return sub.LoadAll(rows)
}
