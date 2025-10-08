package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rancherlabs/slsactl/internal/product"
)

const productf = `usage:
    %[1]s product verify rancher-prime:v2.12.2
    %[1]s product artifacts rancher-prime:v2.12.2
`

func productCmd(args []string) error {
	var registry string
	f := flag.NewFlagSet("", flag.ContinueOnError)
	f.StringVar(&registry, "registry", "", "The registry used to fetch images and artefacts.")
	err := f.Parse(args[1:])
	if err != nil {
		return err
	}

	if len(f.Args()) < 1 {
		showProductUsage()
	}

	arg := f.Arg(0)
	nameVer := strings.Split(arg, ":")
	if len(nameVer) != 2 {
		return fmt.Errorf("invalid name version %q: format expected <name>:<version>", arg)
	}

	return product.Verify(registry, nameVer[0], nameVer[1], true, true)
}

func showProductUsage() {
	fmt.Printf(productf, exeName())
	os.Exit(1)
}
