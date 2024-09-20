package cmd

import (
	"fmt"
	"runtime/debug"
)

var version = "v0.0.0-dev"

func versionCmd(args []string) error {
	if info, ok := debug.ReadBuildInfo(); ok {
		rev := ""
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				rev = fmt.Sprintf("Revision: %s\n", s.Value)
				break
			}
		}
		fmt.Printf("%s version: %s\n%sGo: %s\n",
			exeName(), version, rev, info.GoVersion)
	} else {
		fmt.Printf("%s version: %s\n", exeName(), version)
	}

	return nil
}
