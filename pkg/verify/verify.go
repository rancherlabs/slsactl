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
	cosignCmd "github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

var (
	// ErrNoVerifierFound will be returned by Verify when no verifiers match the provided image.
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

	var matched []internal.Verifier

	for _, v := range verifiers {
		if v.Matches(image) {
			matched = append(matched, v)
		}
	}

	if len(matched) == 0 {
		return fmt.Errorf("%w: %q", ErrNoVerifierFound, image)
	}

	var lastErr error
	for _, v := range matched {
		err := v.Verify(ctx, image)
		if err == nil {
			return nil
		}
		lastErr = errors.Join(lastErr, err) // Aggregate errors from all verifiers
	}

	return lastErr
}

type cosignImplementation struct{}

func (*cosignImplementation) Verify(ctx context.Context, vc cosignCmd.VerifyCommand, image string) error {
	return vc.Exec(ctx, []string{image})
}
