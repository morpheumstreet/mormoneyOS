package tools

import (
	"log/slog"
	"path/filepath"
	"runtime"
)

// PluginLoader loads tools from plugin paths (.so on Linux, future: .wasm).
// Go plugins require Linux and same Go version; other platforms no-op.
type PluginLoader struct {
	Paths []string
}

// LoadPlugins attempts to load .so plugins from Paths and register tools.
// Each path can be a directory (scans for *.so) or a single .so file.
// Plugins must export a symbol "RegisterTools" with signature:
//
//	func RegisterTools(register func(Tool))
func (p *PluginLoader) LoadPlugins(reg *Registry) {
	if runtime.GOOS != "linux" {
		slog.Debug("plugin loader: Go plugins only supported on Linux, skipping")
		return
	}
	for _, path := range p.Paths {
		p.loadPath(reg, path)
	}
}

func (p *PluginLoader) loadPath(reg *Registry, path string) {
	// Try as directory first
	entries, err := readDirNoStat(path)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".so" {
				p.loadOne(reg, filepath.Join(path, e.Name()))
			}
		}
		return
	}
	// Try as file
	if filepath.Ext(path) == ".so" {
		p.loadOne(reg, path)
	}
}

func (p *PluginLoader) loadOne(reg *Registry, soPath string) {
	loaded, err := loadPluginAndRegister(soPath, reg)
	if err != nil {
		slog.Warn("plugin load failed", "path", soPath, "err", err)
		return
	}
	if loaded {
		slog.Info("plugin loaded", "path", soPath)
	}
}
