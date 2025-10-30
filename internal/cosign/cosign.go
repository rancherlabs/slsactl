package cosign

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/rancherlabs/slsactl/internal/provenance"
	"github.com/sigstore/cosign/v3/pkg/cosign"
	"github.com/sigstore/fulcio/pkg/certificate"
)

const (
	// builderID defines the builder ID when the provenance has been modified.
	builderID = "https://github.com/rancherlabs/slsactl/tree/main/buildtypes/buildkit-gha/v1"
)

func GetCosignCertData(ctx context.Context, img string) (*v1.ProvenancePredicate, error) {
	ref, err := name.ParseReference(img, name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("failed strict validation (image name should be fully qualified): %w", err)
	}

	payloads, err := cosign.FetchSignaturesForReference(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image signatures: %w", err)
	}

	if len(payloads) == 0 {
		return nil, errors.New("no payloads found for image")
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
		deps[0].Annotations = map[string]any{
			"ref": commitRef,
		}
	}

	override.BuildDefinition.ResolvedDependencies = deps
	override.RunDetails.Builder.ID = builderID

	return override, nil
}
