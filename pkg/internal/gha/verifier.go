package gha

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancherlabs/slsactl/internal/cosign"
	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

// Verifier implements a verifier for Rancher images that were signed using
// ephemeral keys (a.k.a. "keyless") with GitHub OIDC.
type Verifier struct {
	internal.UpstreamVerifier

	HashAlgorithm crypto.Hash
}

func (v *Verifier) Matches(image string) bool {
	return true
}

func (v *Verifier) Verify(ctx context.Context, image string) error {
	if !v.Matches(image) {
		return fmt.Errorf("%w %q", internal.ErrInvalidImage, image)
	}

	slog.DebugContext(ctx, "GHA keyless verification")
	var certIdentity string
	var err error

	repo, ref, err := getImageRepoRef(image)
	if err != nil {
		return fmt.Errorf("failed to parse image name: %w", err)
	}

	if mutable, ok := mutableRepo[repo+":"+ref]; ok && mutable {
		certIdentity, err = getMutableCertIdentity(ctx, image)
		if err != nil {
			return err
		}
	} else {
		certIdentity, err = getCertIdentity(image)
		if err != nil {
			return err
		}
	}

	vc := verify.VerifyCommand{
		CertVerifyOptions: options.CertVerifyOptions{
			CertIdentityRegexp: certIdentity,
			CertOidcIssuer:     "https://token.actions.githubusercontent.com",
		},
		RekorURL:        options.DefaultRekorURL,
		CheckClaims:     true,
		HashAlgorithm:   v.HashAlgorithm,
		NewBundleFormat: true,
	}

	return v.UpstreamVerifier.Verify(ctx, vc, image)
}

func getImageRepoRef(imageName string) (string, string, error) {
	if len(imageName) < 5 {
		return "", "", fmt.Errorf("invalid image name: %q", imageName)
	}

	if strings.Contains(imageName, "@") {
		imageName = strings.Split(imageName, "@")[0]
	}

	d := strings.Split(imageName, ":")
	if len(d) < 2 || len(d[1]) == 0 {
		return "", "", fmt.Errorf("missing image tag: %q", imageName)
	}

	ref, err := name.ParseReference(imageName, name.WeakValidation)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse image name: %w", err)
	}
	repo := ref.Context().RepositoryStr()

	names := strings.Split(strings.TrimPrefix(repo, "library/"), "/")
	if len(names) < 2 {
		return "", "", fmt.Errorf("unsupported image name: %q", imageName)
	}

	// For multi-leveled, assumes the last two components represents org/repo.
	r := strings.Split(repo, "/")
	if len(r) > 2 {
		repo = strings.Join(r[len(r)-2:], "/")
	}

	return repo, ref.Identifier(), nil
}

func getMutableCertIdentity(ctx context.Context, imageName string) (string, error) {
	var ref string
	var realref any
	var ok bool

	repo, _, err := getImageRepoRef(imageName)
	if err != nil {
		return "", fmt.Errorf("failed to parse image name: %w", err)
	}

	data, err := cosign.GetCosignCertData(ctx, imageName)
	if err != nil {
		return "", err
	}
	if len(data.BuildDefinition.ResolvedDependencies) == 0 {
		return "", fmt.Errorf("no resolved dependencies field in cert data: %q", imageName)
	}
	if realref, ok = data.BuildDefinition.ResolvedDependencies[0].Annotations["ref"]; !ok {
		return "", fmt.Errorf("no ref field in cert data: %q", imageName)
	}
	if ref, ok = realref.(string); !ok {
		return "", fmt.Errorf("ref field is invalid in cert data: %q", imageName)
	}

	// Override repo
	repo = overrideRepo(repo)

	return fmt.Sprintf("https://github.com/%s/.github/workflows/release.yml@%s", repo, ref), nil
}

func getCertIdentity(imageName string) (string, error) {
	repo, ref, err := getImageRepoRef(imageName)
	if err != nil {
		return "", fmt.Errorf("failed to parse image name: %w", err)
	}

	// RKE2 images have container image tags <VERSION>-rke2r1 which are
	// generated from Git tags <VERSION>+rke2r1. Around version v1.33.0,
	// the &#43; was replaced with +.
	if strings.HasPrefix(repo, "rancher/rke2") {
		ref = strings.Replace(ref, "-rke2", "(\\+|&#43;)rke2", 1)
	}

	if strings.HasPrefix(repo, "rancher/system-agent-installer-k3s") {
		ref = strings.Replace(ref, "-k3s", "\\+k3s", 1)
	}

	// neuvector images don't have "v" prefix like its Git tags
	if strings.Contains(imageName, "neuvector") {
		ref = "v" + ref
	}

	suffixes := archSuffixes
	if s, ok := imageSuffixes[repo]; ok {
		suffixes = append(suffixes, s...)
	}

	for _, suffix := range suffixes {
		ref = strings.TrimSuffix(ref, suffix)
	}

	repo = overrideRepo(repo)

	// Check whether there is an identity override for the specific repo.
	if identity, found := identityOverride[repo]; found {
		return identity, nil
	}

	return fmt.Sprintf("^https://github.com/%s/.github/workflows/release.(yml|yaml)@refs/tags/%s$", repo, ref), nil
}

func overrideRepo(repo string) string {
	if v, ok := imageRepo[repo]; ok {
		return v
	}

	return repo
}
