package main

import (
	"os"
	"path/filepath"
)

// FindCompilerLanuncher probes compiler launcher if exists.
// Currently only probes SN-DBS launcher.
func FindCompilerLauncher() string {
	dir, ok := os.LookupEnv("SCE_ROOT_DIR")
	if ok {
		lancherPath := filepath.Join(dir, "Common", "SN-DBS", "bin", "dbsbuild.exe")
		if _, err := os.Stat(lancherPath); err == nil {
			return filepath.ToSlash(filepath.Clean(lancherPath))
		}
	}
	return ""
}
