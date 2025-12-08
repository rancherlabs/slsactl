package copy

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

// CopyImage copies a single container image with its cosign signature from source
// to target registry. The source and target must be fully qualified image references
// including registry, repository, and tag.
//
// Example:
//
//	err := copy.CopyImage(
//	    "stgregistry.suse.com/rancher/fleet-agent:v0.13.0",
//	    "localhost:5000/rancher/fleet-agent:v0.13.0",
//	)
func CopyImage(sourceImage, targetImage string) error {
	ctx := context.Background()

	// Copy the container image itself with NoClobber to prevent overwriting existing tags
	err := crane.Copy(sourceImage, targetImage,
		crane.WithContext(ctx),
		crane.WithNoClobber(true))
	if err != nil {
		return fmt.Errorf("failed to copy image from %q to %q: %w", sourceImage, targetImage, err)
	}

	// Get the digest of the source image to locate its signature
	digest, err := crane.Digest(sourceImage)
	if err != nil {
		return fmt.Errorf("failed to get source image digest for %q: %w", sourceImage, err)
	}

	// Parse source and target references for signature copying
	sourceRef, err := name.ParseReference(sourceImage)
	if err != nil {
		return fmt.Errorf("failed to parse source image reference: %w", err)
	}

	targetRef, err := name.ParseReference(targetImage)
	if err != nil {
		return fmt.Errorf("failed to parse target image reference: %w", err)
	}

	if sourceRef.Identifier() != targetRef.Identifier() {
		return fmt.Errorf("source tag can't be different from target tag (signatures are bound to content, not tags)")
	}

	// Construct signature tag using the digest
	// Cosign stores signatures as: <repository>:sha256-<hex>.sig
	hex := digest[7:] // Remove "sha256:" prefix
	signatureTag := fmt.Sprintf("sha256-%s.sig", hex)

	// Build source and target signature references
	sourceSigRef := fmt.Sprintf("%s:%s", sourceRef.Context().Name(), signatureTag)

	// check old format first
	if _, err := crane.Manifest(sourceSigRef, crane.WithContext(ctx)); err != nil {
		// try one last time for the new format (no .sig suffix)
		signatureTag = strings.TrimSuffix(signatureTag, ".sig")
		sourceSigRef = strings.TrimSuffix(sourceSigRef, ".sig")
		if _, err := crane.Manifest(sourceSigRef, crane.WithContext(ctx)); err != nil {
			return fmt.Errorf("tried old/new formats, no manifest found: %w", err)
		}
	}

	targetSigRef := fmt.Sprintf("%s:%s", targetRef.Context().Name(), signatureTag)

	// Copy the signature with NoClobber to prevent overwriting existing signatures
	err = crane.Copy(sourceSigRef, targetSigRef,
		crane.WithContext(ctx),
		crane.WithNoClobber(true))
	if err != nil {
		return fmt.Errorf("failed to copy signature from %q to %q: %w", sourceSigRef, targetSigRef, err)
	}

	return nil
}
