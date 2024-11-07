package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/rancherlabs/slsactl/internal/provenance"
)

func provenanceCmd(img, format, platform string) error {
	var data bytes.Buffer
	err := writeContent(img, "{{json .Provenance}}", &data)
	if err != nil {
		return err
	}

	var buildKit provenance.BuildKitProvenance02
	err = json.Unmarshal(data.Bytes(), &buildKit)
	if err != nil {
		return fmt.Errorf("cannot parse v0.2 provenance: %w", err)
	}

	var predicate *v02.ProvenancePredicate
	if strings.EqualFold(platform, "linux/amd64") {
		if buildKit.LinuxAmd64 != nil {
			predicate = &buildKit.LinuxAmd64.SLSA
		}
	} else if strings.EqualFold(platform, "linux/arm64") {
		if buildKit.LinuxArm64 != nil {
			predicate = &buildKit.LinuxArm64.SLSA
		}
	} else {
		return fmt.Errorf("platform not supported: %q", platform)
	}

	if predicate == nil {
		return fmt.Errorf("provenance information not found for platform %q", platform)
	}

	switch format {
	case "slsav0.2":
		err = print(os.Stdout, predicate)
	case "slsav1":
		provV1 := provenance.ConvertV02ToV1(*predicate)
		err = print(os.Stdout, provV1)

	default:
		return fmt.Errorf("invalid format %q: supported values are slsav0.2 or slsav1", format)
	}

	return err
}

func print(w io.Writer, v interface{}) error {
	outData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal v1 provenance: %w", err)
	}

	_, err = fmt.Fprintln(w, string(outData))
	return err
}
