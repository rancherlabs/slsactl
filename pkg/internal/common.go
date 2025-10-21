package internal

import (
	"context"
	"errors"

	cosign "github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
)

var ErrInvalidImage = errors.New("invalid image")

type Verifier interface {
	Matches(image string) bool
	Verify(ctx context.Context, image string) error
}

type UpstreamVerifier interface {
	Verify(ctx context.Context, vc cosign.VerifyCommand, image string) error
}
