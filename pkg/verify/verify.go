package verify

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/rancherlabs/slsactl/pkg/internal/appco"
	"github.com/rancherlabs/slsactl/pkg/internal/gcp"
	"github.com/rancherlabs/slsactl/pkg/internal/gha"
	"github.com/rancherlabs/slsactl/pkg/internal/obs"
	cosign "github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

var (
	ErrNoVerifierFound = errors.New("no verifier found for image")

	cosignVerifier = &cosignImplementation{}

	verifiers = []internal.Verifier{
		&obs.Verifier{
			HashAlgorithm:    hashAlgo,
			UpstreamVerifier: cosignVerifier,
		},
		&appco.Verifier{
			HashAlgorithm:    hashAlgo,
			UpstreamVerifier: cosignVerifier,
		},
		&gcp.Verifier{
			HashAlgorithm:    hashAlgo,
			UpstreamVerifier: cosignVerifier,
		},
		&gha.Verifier{
			HashAlgorithm:    hashAlgo,
			UpstreamVerifier: cosignVerifier,
		},
	}

	timeout  = 45 * time.Second
	hashAlgo = crypto.SHA256
)

// Verify checks whether a given Rancher Prime image is signed based on the Cosign Signature spec.
// The same extents to CNCF images within the Rancher ecosystem.
//
// Upstream documentation:
// https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md
func Verify(image string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if strings.EqualFold(os.Getenv("DEBUG"), "true") {
		logs.Debug.SetOutput(os.Stderr)
	}

	for _, v := range verifiers {
		if v.Matches(image) {
			return v.Verify(ctx, image)
		}
	}

	return fmt.Errorf("%w: %q", ErrNoVerifierFound, image)
}

type cosignImplementation struct{}

func (*cosignImplementation) Verify(ctx context.Context, vc cosign.VerifyCommand, image string) error {
	return vc.Exec(ctx, []string{image})
}
