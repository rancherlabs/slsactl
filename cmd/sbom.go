package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/rancher/slsactl/internal/sbom"
)

func sbomCmd(img, outformat string) error {
	var buf bytes.Buffer
	err := writeContent(img, "{{json .SBOM}}", &buf)
	if err != nil {
		return fmt.Errorf("cannot write SBOM content: %w", err)
	}

	switch outformat {
	case "cyclonedxjson", "spdxjson":
	default:
		fmt.Printf(
			"invalid format %q for SBOM: supported values are cyclonedxjson or spdxjson\n",
			outformat)
		os.Exit(6)
	}

	if buf.Len() <= 10 {
		fmt.Println("The image does not contain a SBOM layer, generating SBOM based on the image...")
		err = sbom.Generate(img, outformat, &buf)
		if err != nil {
			fmt.Println("Error generating SBOM: %w\n", err)
			os.Exit(7)
		}
	}

	_, err = io.Copy(os.Stdout, &buf)
	if err != nil {
		fmt.Println("Error exporting SBOM to stdout: %w\n", err)
		os.Exit(8)
	}

	return nil
}
