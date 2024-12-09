package verify

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

const (
	timeout    = 45 * time.Second
	maxWorkers = 5
	hashAlgo   = crypto.SHA256
	obsKey     = "https://ftp.suse.com/pub/projects/security/keys/container-key.pem"
)

var archSuffixes = []string{
	"-linux-amd64",
	"-linux-arm64",
	"-windows-amd64",
	"-windows-arm64",
	"-amd64",
	"-arm64",
	"-s390x",
}

// Verify checks whether a given Rancher Prime image is signed based on the Cosign Signature spec.
// The same extents to CNCF images within the Rancher ecosystem.
//
// Upstream documentation:
// https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md
func Verify(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if obsSigned(image) {
		return verifyObs(ctx, image)
	}

	return verifyKeyless(ctx, image)
}

func verifyObs(ctx context.Context, image string) error {
	slog.Debug("OBS verification")
	v := &verify.VerifyCommand{
		KeyRef:        obsKey,
		RekorURL:      options.DefaultRekorURL,
		CertRef:       obsKey,
		CheckClaims:   true,
		HashAlgorithm: hashAlgo,
		MaxWorkers:    maxWorkers,
	}

	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		logs.Debug.SetOutput(os.Stderr)
	}

	return v.Exec(ctx, []string{image})
}

func verifyKeyless(ctx context.Context, image string) error {
	slog.Info("GHA keyless verification")
	certIdentity, err := certIdentity(image)
	if err != nil {
		return err
	}

	fmt.Println("identity:", certIdentity)

	v := &verify.VerifyCommand{
		CertVerifyOptions: options.CertVerifyOptions{
			CertIdentity:   certIdentity,
			CertOidcIssuer: "https://token.actions.githubusercontent.com",
		},
		CheckClaims:   true,
		HashAlgorithm: hashAlgo,
		MaxWorkers:    maxWorkers,
	}

	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		logs.Debug.SetOutput(os.Stderr)
	}

	return v.Exec(ctx, []string{image})
}

func certIdentity(imageName string) (string, error) {
	if len(imageName) < 5 {
		return "", fmt.Errorf("invalid image name: %q", imageName)
	}

	if strings.Contains(imageName, "@") {
		fmt.Println("warn: image name with digest is not supported, use tags only.")
		imageName = strings.Split(imageName, "@")[0]
	}

	d := strings.Split(imageName, ":")
	if len(d) < 2 || len(d[1]) == 0 {
		return "", fmt.Errorf("missing image tag: %q", imageName)
	}

	names := strings.Split(d[0], "/")
	if len(names) < 2 {
		return "", fmt.Errorf("unsupported image name: %q", imageName)
	}

	repo := strings.Join(names[len(names)-2:], "/")
	ref := d[1]

	// RKE2 images have container image tags <VERSION>-rke2r1 which are
	// generated from Git tags <VERSION>+rke2r1.
	if strings.Contains(imageName, "rke2") {
		ref = strings.Replace(d[1], "-rke2", "&#43;rke2", 1)
	}
	for _, suffix := range archSuffixes {
		if strings.HasSuffix(ref, suffix) {
			ref = strings.TrimSuffix(ref, suffix)
		}
	}

	repo = overrideRepo(repo)

	// Check if the image is an upstream image and has a different cert identity.
	if identity, isUpstreamRepo := upstreamImageRepo[repo]; isUpstreamRepo {
		return identity, nil
	}

	return fmt.Sprintf("https://github.com/%s/.github/workflows/release.yml@refs/tags/%s", repo, ref), nil
}

func overrideRepo(repo string) string {
	if v, ok := imageRepo[repo]; ok {
		return v
	}

	return repo
}
