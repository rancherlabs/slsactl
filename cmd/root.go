package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rancherlabs/slsactl/internal/landlock"
)

type command func(args []string) error

var (
	cmds = map[string]command{
		"download": downloadCmd,
		"version":  versionCmd,
		"verify":   verifyCmd,
		"product":  productCmd,
	}

	usagef = `usage: %[1]s <command>

Available commands:
  download:   Download artefacts from container image
  verify:     Verifies the container image's signature
  version:    Shows %[1]s version and build information
  product:    Handle product level requests

`
)

func Exec(args []string) {
	landlock.EnforceOrDie()

	if len(args) < 2 {
		showUsage()
	}

	name := os.Args[1]
	cmd, ok := cmds[name]
	if !ok {
		showUsage()
	}

	err := cmd(args[2:])
	if err != nil {
		fmt.Printf("failed to run %s: %v\n", name, err)
		os.Exit(2)
	}
}

func showUsage() {
	fmt.Printf(usagef, exeName())
	os.Exit(1)
}

func exeName() string {
	return filepath.Base(os.Args[0])
}
