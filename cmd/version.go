package cmd

import (
	"runtime/debug"
)

var Version string

func init() {
	// Pull version data from ldflags or from Git tags, default to (devel) locally
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if Version != "" {
		rootCmd.Version = Version
	} else {
		rootCmd.Version = info.Main.Version
	}
}
