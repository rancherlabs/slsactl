package obs

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

var (
	obsKey = "https://ftp.suse.com/pub/projects/security/keys/container-key.pem"

	obsRegistries = map[string]struct{}{
		"registry.suse.com": {},
	}

	obsPrefixes = map[string]struct{}{
		"bci/":                       {},
		"suse/":                      {},
		"rancher/mirrored-bci":       {},
		"rancher/mirrored-elemental": {},
	}

	obs = map[string]struct{}{
		"rancher/elemental-operator":            {},
		"rancher/seedimage-builder":             {},
		"rancher/elemental-channel/sl-micro":    {},
		"rancher/elemental-operator-crds-chart": {},
		"rancher/elemental-operator-chart":      {},
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
	if _, ok := obsRegistries[registry]; ok {
		return true
	}

	repo := c.RepositoryStr()
	if _, ok := obs[repo]; ok {
		return ok
	}

	for prefix := range obsPrefixes {
		if strings.HasPrefix(repo, prefix) {
			return true
		}
	}
	return false
}

func (v *Verifier) Verify(ctx context.Context, image string) error {
	if !v.Matches(image) {
		return fmt.Errorf("%w %q", internal.ErrInvalidImage, image)
	}

	slog.DebugContext(ctx, "OBS verification")
	vc := verify.VerifyCommand{
		KeyRef:        obsKey,
		CertRef:       obsKey,
		RekorURL:      options.DefaultRekorURL,
		CheckClaims:   true,
		HashAlgorithm: v.HashAlgorithm,
	}

	return v.UpstreamVerifier.Verify(ctx, vc, image)
}
