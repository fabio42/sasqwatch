package cmd

import (
	"runtime/debug"
)

func init() {
	// Pull version data from Git if available
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
