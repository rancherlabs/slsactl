package appco

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
	appCoKey = "https://apps.rancher.io/ap-pubkey.pem"

	appCoRegistries = map[string]struct{}{
		"dp.apps.rancher.io": {},
	}

	appCoPrefixes = map[string]struct{}{
		"rancher/appco-": {},
	}
)

type Verifier struct {
	internal.UpstreamVerifier

	HashAlgorithm crypto.Hash
}

func (v *Verifier) Matches(image string) bool {
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		slog.Debug("failed to parse image", "image", image, "error", err)
		return false
	}

	c := ref.Context()
	registry := c.RegistryStr()
	if _, ok := appCoRegistries[registry]; ok {
		return true
	}

	repo := c.RepositoryStr()
	for prefix := range appCoPrefixes {
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
		KeyRef:        appCoKey,
		CertRef:       appCoKey,
		RekorURL:      options.DefaultRekorURL,
		CheckClaims:   true,
		HashAlgorithm: v.HashAlgorithm,
	}

	return v.UpstreamVerifier.Verify(ctx, vc, image)
}
