package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/docker/buildx/builder"
	"github.com/docker/buildx/util/imagetools"
	dockercmd "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/rancher/slsactl/internal/format"
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

	img := f.Arg(1)
	if f.Arg(0) == "provenance" {
		var format string
		f.StringVar(&format, "format", "", "Format used to return the information.")

		err := f.Parse(args[1:])
		if err != nil {
			return err
		}

		if format == "slsav1" {
			return provenanceSlsaV1(f.Arg(0))
		}
		return provenance(img)
	}

	if f.Arg(0) == "sbom" && len(f.Args()) == 2 {
		return sbom(img)
	}

	showDownloadUsage()
	return nil
}

func showDownloadUsage() {
	fmt.Printf(downloadf, exeName())
	os.Exit(1)
}

func provenance(img string) error {
	return writeContent(img, "{{json .Provenance}}", os.Stdout)
}

func provenanceSlsaV1(img string) error {
	var buf bytes.Buffer
	err := writeContent(img, "{{json .Provenance}}", &buf)
	if err != nil {
		return err
	}

	convert(buf.Bytes(), os.Stdout)
	return nil
}

func sbom(img string) error {
	return writeContent(img, "{{json .SBOM}}", os.Stdout)
}

func writeContent(img, format string, w io.Writer) error {
	cmd, err := dockercmd.NewDockerCli()
	if err != nil {
		return err
	}
	if err := cmd.Initialize(&flags.ClientOptions{}); err != nil {
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

func convert(data []byte, w io.Writer) {
	var buildKit format.BuildKitProvenance02
	err := json.Unmarshal(data, &buildKit)
	if err != nil {
		fmt.Printf("Error parsing v0.2 provenance: %v\n", err)
		os.Exit(1)
	}

	if buildKit.LinuxAmd64 == nil {
		fmt.Println("Error: image does not contain provenance information")
		os.Exit(5)
	}

	provV1 := format.ConvertV02ToV1(buildKit.LinuxAmd64.SLSA)

	outData, err := json.MarshalIndent(provV1, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling v1 provenance: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(w, string(outData))
}
