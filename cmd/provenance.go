package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/rancherlabs/slsactl/internal/provenance"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	certificate "github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
)

var (
	// builderId defines the builder ID when the provenance has been modified.
	builderId = "https://github.com/rancherlabs/slsactl/tree/main/buildtypes/buildkit-gha/v1"
	// buildKitV1 holds the buildType supported for provenance enrichment.
	buildKitV1 = "https://mobyproject.org/buildkit@v1"
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
	if strings.EqualFold(platform, "linux/amd64") && buildKit.LinuxAmd64 != nil {
		predicate = &buildKit.LinuxAmd64.SLSA
	} else if strings.EqualFold(platform, "linux/arm64") && buildKit.LinuxArm64 != nil {
		predicate = &buildKit.LinuxArm64.SLSA
	} else if buildKit.SLSA != nil {
		predicate = buildKit.SLSA
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
		if predicate.BuildType != buildKitV1 {
			return fmt.Errorf("image builtType not supported: %q", predicate.BuildType)
		}

		override, err := cosignCertData(img)
		if err != nil {
			return err
		}

		provV1 := provenance.ConvertV02ToV1(*predicate, override)
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

func cosignCertData(img string) (*v1.ProvenancePredicate, error) {
	ref, err := name.ParseReference(img, name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("failed strict validation (image name should be fully qualified): %w", err)
	}

	payloads, err := cosign.FetchSignaturesForReference(context.Background(), ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image signatures: %w", err)
	}

	if len(payloads) == 0 {
		return nil, fmt.Errorf("no payloads found for image")
	}

	var inparams provenance.InternalParameters
	var commitID, commitRef, repoURL string

	for _, ext := range payloads[0].Cert.Extensions {
		switch {
		case ext.Id.Equal(certificate.OIDSourceRepositoryDigest):
			certificate.ParseDERString(ext.Value, &commitID)
		case ext.Id.Equal(certificate.OIDSourceRepositoryURI):
			certificate.ParseDERString(ext.Value, &repoURL)
		case ext.Id.Equal(certificate.OIDSourceRepositoryRef):
			certificate.ParseDERString(ext.Value, &commitRef)
		case ext.Id.Equal(certificate.OIDBuildTrigger):
			certificate.ParseDERString(ext.Value, &inparams.Trigger)
		case ext.Id.Equal(certificate.OIDRunInvocationURI):
			certificate.ParseDERString(ext.Value, &inparams.InvocationUri)
		}
	}

	override := &v1.ProvenancePredicate{}
	override.BuildDefinition.InternalParameters = inparams
	deps := []v1.ResourceDescriptor{
		{
			URI:    repoURL,
			Digest: common.DigestSet{"gitCommit": commitID},
		},
	}

	if commitRef != "" {
		deps[0].Annotations = map[string]interface{}{
			"ref": commitRef,
		}
	}

	override.BuildDefinition.ResolvedDependencies = deps
	override.RunDetails.Builder.ID = builderId

	return override, nil
}
