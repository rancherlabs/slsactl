package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rancherlabs/slsactl/internal/provenance"
)

func provenanceCmd(img, format string) error {
	switch format {
	case "slsav0.2":
		return writeContent(img, "{{json .Provenance}}", os.Stdout)
	case "slsav1":
		var buf bytes.Buffer
		err := writeContent(img, "{{json .Provenance}}", &buf)
		if err != nil {
			return err
		}

		convert(buf.Bytes(), os.Stdout)
		return nil
	}

	return fmt.Errorf("invalid format %q: supported values are slsav0.2 or slsav1", format)
}

func convert(data []byte, w io.Writer) {
	var buildKit provenance.BuildKitProvenance02
	err := json.Unmarshal(data, &buildKit)
	if err != nil {
		fmt.Printf("Error parsing v0.2 provenance: %v\n", err)
		os.Exit(1)
	}

	if buildKit.LinuxAmd64 == nil {
		fmt.Println("Error: image does not contain provenance information")
		os.Exit(5)
	}

	provV1 := provenance.ConvertV02ToV1(buildKit.LinuxAmd64.SLSA)

	outData, err := json.MarshalIndent(provV1, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling v1 provenance: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(w, string(outData))
}
