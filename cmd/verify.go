package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/rancherlabs/slsactl/pkg/verify"
)

const verifyf = `usage:
    %[1]s verify <IMAGE>
`

var (
	certIdentityWorkflow string
	upstreamImageType    string
)

func verifyCmd(args []string) error {
	f := flag.NewFlagSet("", flag.ContinueOnError)

	if err := f.Parse(args); err != nil {
		return err
	}

	if len(f.Args()) == 0 || len(f.Args()) > 3 {
		showVerifyUsage()
	}

	switch f.Arg(0) {
	case "upstream":
		f.StringVar(&certIdentityWorkflow, "certIdentityWorkflow", "",
			"The workflow used to generate the certificate identity, see relevant Rancher documentation for more information.")

		f.StringVar(&upstreamImageType, "upstreamImageType", "cluster-api",
			"The type of upstream image to verify, currently only 'cluster-api' is supported.")

		if err := f.Parse(args[1:]); err != nil {
			return err
		}

		if certIdentityWorkflow == "" {
			fmt.Println("certIdentityWorkflow is required")
			showVerifyUsage()
		}
	}

	err := verify.Verify(f.Arg(0), upstreamImageType, certIdentityWorkflow)
	if err != nil {
		fmt.Printf("cannot validate image %s: ensure you are using an image from the Prime registry\n", f.Arg(0))
	}
	return err
}

func showVerifyUsage() {
	fmt.Printf(verifyf, exeName())
	os.Exit(1)
}
