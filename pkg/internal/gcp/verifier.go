package gcp

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

var (
	registries = map[string]struct{}{
		"registry.k8s.io": {},
	}

	repos = map[string]struct{}{
		"sig-storage/snapshot-controller":                        {},
		"sig-storage/snapshot-validation-webhook":                {},
		"rancher/mirrored-sig-storage-csi-node-driver-registrar": {},
		"rancher/mirrored-sig-storage-csi-attacher":              {},
		"rancher/mirrored-sig-storage-csi-provisioner":           {},
		"rancher/mirrored-sig-storage-csi-resizer":               {},
		"rancher/mirrored-sig-storage-csi-snapshotter":           {},
		"rancher/mirrored-sig-storage-livenessprobe":             {},
		"rancher/mirrored-sig-storage-snapshot-controller":       {},
		"rancher/mirrored-kube-state-metrics-kube-state-metrics": {},
		"rancher/mirrored-cluster-api-controller":                {},
	}
)

type Verifier struct {
	HashAlgorithm crypto.Hash
	internal.UpstreamVerifier
}

func (v *Verifier) Matches(image string) bool {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		slog.Debug("failed to parse image", "image", image, "error", err)
		return false
	}

	c := ref.Context()
	registry := c.RegistryStr()
	if _, ok := registries[registry]; ok {
		return true
	}

	repo := c.RepositoryStr()
	if _, ok := repos[repo]; ok {
		return true
	}
	return false
}

func (v *Verifier) Verify(ctx context.Context, image string) error {
	if !v.Matches(image) {
		return fmt.Errorf("%w %q", internal.ErrInvalidImage, image)
	}

	slog.DebugContext(ctx, "GCP OIDC verification")
	vc := verify.VerifyCommand{
		CertVerifyOptions: options.CertVerifyOptions{
			CertIdentityRegexp: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
			CertOidcIssuer:     "https://accounts.google.com",
		},
		RekorURL:      options.DefaultRekorURL,
		CheckClaims:   true,
		HashAlgorithm: v.HashAlgorithm,
	}

	return v.UpstreamVerifier.Verify(ctx, vc, image)
}
