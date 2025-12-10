package imagecopy

import (
	"context"
	"errors"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

// ErrSignatureNotFound indicates no signature was found for the image
var ErrSignatureNotFound = imagelist.ErrSignatureNotFound

// ImageAndSignature copies a single container image with its cosign signature from source
// to target registry. The source and target must be fully qualified image references
// including registry, repository, and tag.
//
// Example:
//
//	err := copy.ImageAndSignature(
//	    "stgregistry.suse.com/rancher/fleet-agent:v0.13.0",
//	    "localhost:5000/rancher/fleet-agent:v0.13.0",
//	)
func ImageAndSignature(sourceImage, targetImage string) error {
	return imagelist.CopySignature(context.Background(), sourceImage, targetImage, true)
}
