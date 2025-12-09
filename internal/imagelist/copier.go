package imagelist

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
)

var externalImages = map[string]string{
	"sig-storage/snapshot-controller":                        "registry.k8s.io/sig-storage/snapshot-controller",
	"sig-storage/snapshot-validation-webhook":                "registry.k8s.io/sig-storage/snapshot-validation-webhook",
	"rancher/mirrored-sig-storage-csi-node-driver-registrar": "registry.k8s.io/sig-storage/csi-node-driver-registrar",
	"rancher/mirrored-sig-storage-csi-attacher":              "registry.k8s.io/sig-storage/csi-attacher",
	"rancher/mirrored-longhornio-csi-attacher":               "registry.k8s.io/sig-storage/csi-attacher",
	"rancher/mirrored-sig-storage-csi-provisioner":           "registry.k8s.io/sig-storage/csi-provisioner",
	"rancher/mirrored-sig-storage-csi-resizer":               "registry.k8s.io/sig-storage/csi-resizer",
	"rancher/mirrored-sig-storage-csi-snapshotter":           "registry.k8s.io/sig-storage/csi-snapshotter",
	"rancher/mirrored-sig-storage-livenessprobe":             "registry.k8s.io/sig-storage/rancher/mirrored-sig-storage-livenessprobe",
	"rancher/mirrored-sig-storage-snapshot-controller":       "registry.k8s.io/sig-storage/snapshot-controller",
	"rancher/appco-redis":                                    "dp.apps.rancher.io/containers/redis",
	"rancher/mirrored-cilium-cilium":                         "quay.io/cilium/cilium",
	"rancher/mirrored-cilium-cilium-envoy":                   "quay.io/cilium/cilium-envoy",
	"rancher/mirrored-cilium-clustermesh-apiserver":          "quay.io/cilium/clustermesh-apiserver",
	"rancher/mirrored-cilium-hubble-relay":                   "quay.io/cilium/hubble-relay",
	"rancher/mirrored-cilium-operator-aws":                   "quay.io/cilium/operator-aws",
	"rancher/mirrored-cilium-operator-azure":                 "quay.io/cilium/operator-azure",
	"rancher/mirrored-cilium-operator-generic":               "quay.io/cilium/operator-generic",
	"rancher/mirrored-kube-logging-config-reloader":          "ghcr.io/kube-logging/config-reloader",
	"rancher/mirrored-kube-logging-logging-operator":         "ghcr.io/kube-logging/logging-operator",
	"rancher/mirrored-kube-state-metrics-kube-state-metrics": "registry.k8s.io/kube-state-metrics/kube-state-metrics",
	"rancher/mirrored-elemental-operator":                    "registry.suse.com/rancher/elemental-operator",
	"rancher/mirrored-elemental-seedimage-builder":           "registry.suse.com/rancher/seedimage-builder",
	"rancher/mirrored-cluster-api-controller":                "registry.k8s.io/cluster-api/cluster-api-controller",
}

type imageCopier struct {
	m            sync.Mutex
	mirroredOnly bool
	copyImages   bool
}

func (i *imageCopier) Copy(srcImg, dstRegistry string) Entry {
	entry := Entry{
		Image: srcImg,
	}

	if i.mirroredOnly && !strings.Contains(srcImg, "mirrored") {
		entry.Error = errors.New("skipping non-mirrored image: " + srcImg)
		return entry
	}

	ref, err := name.ParseReference(srcImg, name.WeakValidation)
	if err != nil {
		entry.Error = err
		return entry
	}

	reg, err := name.NewRegistry(dstRegistry)
	if err != nil {
		entry.Error = err
		return entry
	}

	repo := reg.Repo(ref.Context().RepositoryStr())
	dst := repo.Tag(ref.Identifier()).String()

	ctx := context.TODO()

	// Reset stdout/stderr to avoid verbose output from cosign.
	i.m.Lock()
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = nil
	os.Stderr = nil

	err = CopySignature(ctx, srcImg, dst, i.copyImages)
	if err != nil {
		entry.Error = err
	}
	entry.Signed = (err == nil)

	os.Stdout = stdout
	os.Stderr = stderr
	i.m.Unlock()

	return entry
}

func signatureSource(srcRef name.Reference, tag string) string {
	repo := srcRef.Context().RepositoryStr()
	if upstream, found := externalImages[repo]; found {
		return fmt.Sprintf("%s:%s", upstream, tag)
	}

	// Fully qualified reference: <registry>/<repository>:<signature_tag>
	return fmt.Sprintf("%s:%s", srcRef.Context().Name(), tag)
}

// CopySignature copies a container image with its cosign signature from source to target registry.
// Supports both legacy (.sig suffix) and new OCI artifact signature formats. Source and target
// tags must match since signatures are bound to content digests. Uses NoClobber to prevent
// overwriting existing tags.
func CopySignature(ctx context.Context, srcImgRef, dstImgRef string, copyImage bool) error {
	digest, err := crane.Digest(srcImgRef)
	if err != nil {
		return fmt.Errorf("failed to get signed image digest for %q: %w", srcImgRef, err)
	}

	sourceRef, err := name.ParseReference(srcImgRef)
	if err != nil {
		return fmt.Errorf("failed to parse source image reference: %w", err)
	}

	targetRef, err := name.ParseReference(dstImgRef)
	if err != nil {
		return fmt.Errorf("failed to parse target image reference: %w", err)
	}

	if sourceRef.Identifier() != targetRef.Identifier() {
		return fmt.Errorf("source tag can't be different from target tag (signatures are bound to content, not tags); source tag: %s | target tag: %s", sourceRef.Identifier(), targetRef.Identifier())
	}

	hex := strings.TrimPrefix(digest, "sha256:")
	signatureTag := fmt.Sprintf("sha256-%s.sig", hex)
	sourceSigRef := signatureSource(sourceRef, signatureTag)

	// check old format first (with .sig suffix)
	_, err = crane.Manifest(sourceSigRef, crane.WithContext(ctx))
	if err != nil {
		// try one last time for the new format (no .sig suffix)
		signatureTag = strings.TrimSuffix(signatureTag, ".sig")
		sourceSigRef = strings.TrimSuffix(sourceSigRef, ".sig")
		_, err = crane.Manifest(sourceSigRef, crane.WithContext(ctx))
		if err != nil {
			return fmt.Errorf("tried old/new formats, no manifest found for %s - %w", srcImgRef, err)
		}
	}

	// copy image only after all safety checks
	if copyImage {
		err := crane.Copy(srcImgRef, dstImgRef,
			crane.WithContext(ctx),
			crane.WithNoClobber(true)) // ensure tags won't be overwritten.

		if err != nil && !strings.Contains(err.Error(), "refusing to clobber existing tag") {
			return fmt.Errorf("failed to copy image from %q to %q: %w",
				srcImgRef, dstImgRef, err)
		}
	}

	dstSigRef := fmt.Sprintf("%s:%s", targetRef.Context().Name(), signatureTag)

	err = crane.Copy(sourceSigRef, dstSigRef,
		crane.WithContext(ctx),
		crane.WithNoClobber(true), // ensure existing signatures won't be overwritten.
	)
	if err != nil && !strings.Contains(err.Error(), "refusing to clobber existing tag") {
		return fmt.Errorf("failed to copy signature from %q to %q: %w", sourceSigRef, dstSigRef, err)
	}

	return nil
}
