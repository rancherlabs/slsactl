package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rancher/slsactl/internal/provenance"
)

func provenanceCmd(img string) error {
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
