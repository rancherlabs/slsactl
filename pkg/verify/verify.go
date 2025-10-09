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
	"github.com/google/go-containerregistry/pkg/name"
	cosign "github.com/rancherlabs/slsactl/internal/cosign"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
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

	repo, ref, err := getImageRepoRef(image)
	if err != nil {
		return fmt.Errorf("failed to parse image name: %w", err)
	}

	if outsideGitHub(repo) {
		return verifyOutsideGitHub(ctx, repo, image)
	}

	if obsSigned(image) {
		return verifyObs(ctx, image)
	}

	return verifyKeyless(ctx, repo, ref, image)
}

func outsideGitHub(repo string) bool {
	_, ok := nonGitHub[repo]
	return ok
}

func verifyOutsideGitHub(ctx context.Context, repo, image string) error {
	slog.InfoContext(ctx, "Non-GHA keyless verification")

	data, ok := nonGitHub[repo]
	if !ok {
		return fmt.Errorf("unknown non-GHA image %q", image)
	}

	return genericVerify(ctx, image, data.identity, data.issuer)
}

func verifyObs(ctx context.Context, image string) error {
	slog.DebugContext(ctx, "OBS verification")
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

func verifyKeyless(ctx context.Context, repo, ref, image string) error {
	slog.InfoContext(ctx, "GHA keyless verification")
	var certIdentity string
	var err error

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

	return genericVerify(ctx, image, certIdentity, "https://token.actions.githubusercontent.com")
}

func genericVerify(ctx context.Context, image, certIdentity, issuer string) error {
	fmt.Println("identity:", certIdentity)

	v := &verify.VerifyCommand{
		CertVerifyOptions: options.CertVerifyOptions{
			CertIdentityRegexp: certIdentity,
			CertOidcIssuer:     issuer,
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

func getImageRepoRef(imageName string) (string, string, error) {
	if len(imageName) < 5 {
		return "", "", fmt.Errorf("invalid image name: %q", imageName)
	}

	if strings.Contains(imageName, "@") {
		fmt.Println("warn: image name with digest is not supported, use tags only.")
		imageName = strings.Split(imageName, "@")[0]
	}

	d := strings.Split(imageName, ":")
	if len(d) < 2 || len(d[1]) == 0 {
		return "", "", fmt.Errorf("missing image tag: %q", imageName)
	}

	names := strings.Split(d[0], "/")
	if len(names) < 2 {
		return "", "", fmt.Errorf("unsupported image name: %q", imageName)
	}

	ref, err := name.ParseReference(imageName, name.WeakValidation)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse image name: %w", err)
	}
	repo := ref.Context().RepositoryStr()

	// For multi-leveled, assumes the last two components represents org/repo.
	r := strings.Split(repo, "/")
	if len(r) > 2 {
		repo = strings.Join(r[len(r)-2:], "/")
	}

	return repo, ref.Identifier(), nil
}

func getMutableCertIdentity(ctx context.Context, imageName string) (string, error) {
	var ref string
	var realref interface{}
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

	// Check if the image is an upstream image and has a different cert identity.
	if identity, isUpstreamRepo := upstreamImageRepo[repo]; isUpstreamRepo {
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
