package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rancherlabs/slsactl/internal/sbom"
)

func sbomCmd(img, outformat, platform string) error {
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

	var data sbom.BuildKitSBOM
	err = json.Unmarshal(buf.Bytes(), &data)
	if err != nil {
		fmt.Println("Error cannot unmarshal SBOM: %w\n", err)
		os.Exit(6)
	}

	var spdx json.RawMessage
	if strings.EqualFold(platform, "linux/amd64") {
		if data.LinuxAmd64 != nil {
			spdx = data.LinuxAmd64.SPDX
		}

	} else if strings.EqualFold(platform, "linux/arm64") {
		if data.LinuxArm64 != nil {
			spdx = data.LinuxArm64.SPDX
		}
	} else {
		return fmt.Errorf("platform not supported: %q", platform)
	}

	if len(spdx) != 0 {
		m, err := spdx.MarshalJSON()
		if len(m) > 4 && err == nil {
			buf.Reset()
			buf.ReadFrom(bytes.NewBuffer(m))
		}
	}

	if buf.Len() < 10 {
		buf.Reset()
		// The image does not contain a SBOM layer, generates SBOM on demand.
		err = sbom.Generate(img, outformat, &buf)
		if err != nil {
			fmt.Println("Error generating SBOM: %w\n", err)
			os.Exit(7)
		}
	} else {
		// The SBOM layer exists, we can now convert it to a different format
		// if asked by the user.
		if outformat == "cyclonedxjson" {
			err = sbom.ConvertToCyclonedxJson(&buf, os.Stdout)
			if err != nil {
				fmt.Printf("Error converting SBOM: %v\n", err)
				os.Exit(8)
			}
			return nil
		}
	}

	_, err = io.Copy(os.Stdout, &buf)
	if err != nil {
		fmt.Printf("Error exporting SBOM to stdout: %v\n", err)
		os.Exit(8)
	}
	fmt.Fprintln(os.Stdout)

	return nil
}
