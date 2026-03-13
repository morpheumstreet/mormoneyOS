//go:build !linux

package tools

import (
	"os"
)

func loadPluginAndRegister(soPath string, reg *Registry) (bool, error) {
	return false, nil
}

func readDirNoStat(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
