//go:build linux

package tools

import (
	"os"
	"plugin"
)

func loadPluginAndRegister(soPath string, reg *Registry) (bool, error) {
	p, err := plugin.Open(soPath)
	if err != nil {
		return false, err
	}
	sym, err := p.Lookup("RegisterTools")
	if err != nil {
		return false, err
	}
	// RegisterTools must have signature: func(func(Tool))
	fn, ok := sym.(*func(func(Tool)))
	if !ok {
		return false, nil
	}
	(*fn)(reg.Register)
	return true, nil
}

func readDirNoStat(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
