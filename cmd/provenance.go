package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	cosign "github.com/rancherlabs/slsactl/internal/cosign"
	"github.com/rancherlabs/slsactl/internal/provenance"
	"github.com/rancherlabs/slsactl/pkg/verify"
)

var (
	// buildKitV1 holds the buildType supported for provenance enrichment.
	buildKitV1 = "https://mobyproject.org/buildkit@v1"
	// buildkit default to SLSA v1 - https://github.com/moby/buildkit/pull/6526
	buildKitV1Full = "https://github.com/moby/buildkit/blob/master/docs/attestations/slsa-definitions.md"
)

func checkBuildKitV1Spec(img, platform string) (error, *v1.ProvenancePredicate, *v02.ProvenancePredicate) { //nolint
	var data bytes.Buffer
	err := writeContent(img, "provenance", &data)
	if err != nil {
		return err, nil, nil
	}

	if strings.Contains(data.String(), buildKitV1Full) {
		var buildKit provenance.SLSAV1Provenance
		err = json.Unmarshal(data.Bytes(), &buildKit)
		if err != nil {
			return fmt.Errorf("cannot parse v1.0 provenance: %w", err), nil, nil
		}

		var predicate *v1.ProvenancePredicate //nolint

		if strings.EqualFold(platform, "linux/amd64") && buildKit.LinuxAmd64 != nil {
			predicate = &buildKit.LinuxAmd64.SLSA
		} else if strings.EqualFold(platform, "linux/arm64") && buildKit.LinuxArm64 != nil {
			predicate = &buildKit.LinuxArm64.SLSA
		} else if buildKit.SLSA != nil {
			predicate = buildKit.SLSA
		} else {
			return fmt.Errorf("platform not supported: %q", platform), nil, nil
		}

		predicate.RunDetails.Builder.ID = cosign.BuilderID
		return nil, predicate, nil
	}

	var buildKit provenance.BuildKitProvenance02
	err = json.Unmarshal(data.Bytes(), &buildKit)
	if err != nil {
		return fmt.Errorf("cannot parse v0.2 provenance: %w", err), nil, nil
	}

	var predicate *v02.ProvenancePredicate

	if strings.EqualFold(platform, "linux/amd64") && buildKit.LinuxAmd64 != nil {
		predicate = &buildKit.LinuxAmd64.SLSA
	} else if strings.EqualFold(platform, "linux/arm64") && buildKit.LinuxArm64 != nil {
		predicate = &buildKit.LinuxArm64.SLSA
	} else if buildKit.SLSA != nil {
		predicate = buildKit.SLSA
	} else {
		return fmt.Errorf("platform not supported: %q", platform), nil, nil
	}

	return nil, nil, predicate
}

func provenanceCmd(img, format, platform string) error {
	err, predicateV1Full, predicate := checkBuildKitV1Spec(img, platform)
	if err != nil {
		return err
	}

	switch format {
	case "slsav0.2":
		if predicate != nil && predicate.BuildType == buildKitV1 {
			err = printOutput(os.Stdout, predicate)
		} else {
			err = printOutput(os.Stdout, predicateV1Full)
		}
	case "slsav1":
		if predicate != nil && predicate.BuildType != buildKitV1 {
			return fmt.Errorf("image builtType not supported: %q", predicate.BuildType)
		} else if predicateV1Full != nil && predicateV1Full.BuildDefinition.BuildType != buildKitV1Full {
			return fmt.Errorf("image builtType not supported: %q", predicateV1Full.BuildDefinition.BuildType)
		}

		// Avoid polluting the output in successful verifications.
		sout := os.Stdout
		serr := os.Stderr
		os.Stdout = nil
		os.Stderr = nil
		err := verify.Verify(img)
		os.Stdout = sout
		os.Stderr = serr

		if err != nil {
			return fmt.Errorf("failed to verify %q: %w", img, err)
		}

		if predicate != nil {
			override, err := cosign.GetCosignCertData(context.Background(), img)
			if err != nil {
				return err
			}

			provV1 := provenance.ConvertV02ToV1(*predicate, override)
			err = printOutput(os.Stdout, provV1) //nolint
		} else {
			err = printOutput(os.Stdout, predicateV1Full) //nolint
		}

	default:
		return fmt.Errorf("invalid format %q: supported values are slsav0.2 or slsav1", format)
	}

	return err
}

func printOutput(w io.Writer, v any) error {
	outData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal v1 provenance: %w", err)
	}

	_, err = fmt.Fprintln(w, string(outData))
	return err
}
