package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/util/imagetools"
	dockercmd "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
)

const downloadf = `usage:
    %[1]s download provenance <IMAGE>
    %[1]s download sbom <IMAGE>
`

func downloadCmd(args []string) error {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	err := f.Parse(args)
	if err != nil {
		return err
	}

	if len(f.Args()) < 2 {
		showDownloadUsage()
	}

	var format string
	var platform string
	img := f.Arg(f.NArg() - 1)
	if f.Arg(0) == "provenance" {
		f.StringVar(&format, "format", "slsav0.2", "The format for the Provenance output. Supported values are slsav0.2 (default) and slsav1.")
		f.StringVar(&platform, "platform", "linux/amd64", "The target platform for the container image. Most supported platforms are linux/amd64 and linux/arm64.")

		err := f.Parse(args[1:])
		if err != nil {
			return err
		}

		return provenanceCmd(img, format, platform)
	}

	if f.Arg(0) == "sbom" {
		f.StringVar(&format, "format", "spdxjson", "The format for the SBOM output. Supported values are spdxjson (default) and cyclonedxjson.")
		f.StringVar(&platform, "platform", "linux/amd64", "The target platform for the container image. Most supported platforms are linux/amd64 and linux/arm64.")

		err := f.Parse(args[1:])
		if err != nil {
			return err
		}

		return sbomCmd(img, format, platform)
	}

	showDownloadUsage()
	return nil
}

func showDownloadUsage() {
	fmt.Printf(downloadf, exeName())
	os.Exit(1)
}

func writeContent(img, format string, w io.Writer) error {
	cmd, err := dockercmd.NewDockerCli()
	if err != nil {
		return err
	}

	err = cmd.Initialize(&flags.ClientOptions{})
	if err != nil {
		return err
	}

	b, err := builder.New(cmd)
	if err != nil {
		return err
	}
	opts, err := b.ImageOpt()
	if err != nil {
		return err
	}
	printer, err := imagetools.NewPrinter(context.TODO(), opts, img, format)
	if err != nil {
		return err
	}

	err = printer.Print(false, w)
	if err != nil {
		return err
	}

	// End with a line break.
	_, err = fmt.Fprintln(w)
	return err
}
