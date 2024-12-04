package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/rancherlabs/slsactl/internal/verify"
)

const verifyf = `usage:
    %[1]s verify <IMAGE>
`

func verifyCmd(args []string) error {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	err := f.Parse(args)
	if err != nil {
		return err
	}

	if len(f.Args()) != 1 {
		showVerifyUsage()
	}

	err = verify.Verify(f.Arg(0))
	if err != nil {
		fmt.Printf("cannot validate image %s: ensure you are using an image from the Prime registry\n", f.Arg(0))
	}
	return err
}

func showVerifyUsage() {
	fmt.Printf(verifyf, exeName())
	os.Exit(1)
}

// Verify is a public function to be imported and execute Verify function from internal/verify/verify.go
func Verify(imageName string) error {
	return verify.Verify(imageName)
}
