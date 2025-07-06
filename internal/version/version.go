package version

import (
	"fmt"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Full() string {
	if Version != "dev" {
		return fmt.Sprintf("Gart version %s (commit: %s, built at: %s)", Version, Commit, Date)
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return fmt.Sprintf("Gart version %s", info.Main.Version)
	}
	return "Gart version unknown"
}
