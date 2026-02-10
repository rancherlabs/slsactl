package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rancherlabs/slsactl/internal/product"
)

const productf = `usage:
    %[1]s product verify --registry <src_registry> rancher-prime:v2.12.2
    %[1]s product copy --registry <src_registry> rancher-prime:v2.12.2 <target_registry>
    %[1]s product download --registry <src_registry> rancher-prime:v2.12.2
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

	switch args[0] {
	case "verify":
		return product.Verify(registry, nameVer[0], nameVer[1], true, true)
	case "copy":
		if f.NArg() != 2 {
			showProductUsage()
		}

		targetRegistry := f.Arg(1)
		return product.Copy(registry, nameVer[0], nameVer[1], targetRegistry)
	case "download":
		return product.Download(registry, nameVer[0], nameVer[1])
	default:
		showProductUsage()
	}

	return nil
}

func showProductUsage() {
	fmt.Printf(productf, exeName())
	os.Exit(1)
}
