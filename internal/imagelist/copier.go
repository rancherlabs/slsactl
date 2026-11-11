package imagelist

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// ErrNoSignaturesFound indicates no signature was found for the image.
var ErrNoSignaturesFound = errors.New("no signatures found")

var errNoReferrers = errors.New("no signature referrers found")

// Artifact types of OCI 1.1 signature referrers (cosign v3 bundles and cosign OCI 1.1 signatures).
const (
	sigstoreBundleTypePrefix = "application/vnd.dev.sigstore.bundle"
	cosignSignatureType      = "application/vnd.dev.cosign.artifact.sig.v1+json"

	sigstoreBundleContentAnnotation   = "dev.sigstore.bundle.content"
	sigstoreBundlePredicateAnnotation = "dev.sigstore.bundle.predicateType"

	// cosignSignPredicateType is the in-toto predicate type of the DSSE envelope that
	// cosign v3 writes when signing an image: it is a signature, not an attestation.
	cosignSignPredicateType = "https://sigstore.dev/cosign/sign/v1"
	ociEmptyMediaType       = "application/vnd.oci.empty.v1+json"
)

var externalImages = map[string]string{
	"sig-storage/snapshot-controller":                                 "registry.k8s.io/sig-storage/snapshot-controller",
	"sig-storage/snapshot-validation-webhook":                         "registry.k8s.io/sig-storage/snapshot-validation-webhook",
	"rancher/mirrored-sig-storage-csi-node-driver-registrar":          "registry.k8s.io/sig-storage/csi-node-driver-registrar",
	"rancher/mirrored-sig-storage-csi-attacher":                       "registry.k8s.io/sig-storage/csi-attacher",
	"rancher/mirrored-longhornio-csi-attacher":                        "registry.k8s.io/sig-storage/csi-attacher",
	"rancher/mirrored-sig-storage-csi-provisioner":                    "registry.k8s.io/sig-storage/csi-provisioner",
	"rancher/mirrored-sig-storage-csi-resizer":                        "registry.k8s.io/sig-storage/csi-resizer",
	"rancher/mirrored-sig-storage-csi-snapshotter":                    "registry.k8s.io/sig-storage/csi-snapshotter",
	"rancher/mirrored-sig-storage-livenessprobe":                      "registry.k8s.io/sig-storage/livenessprobe",
	"rancher/mirrored-sig-storage-snapshot-controller":                "registry.k8s.io/sig-storage/snapshot-controller",
	"rancher/appco-alertmanager":                                      "dp.apps.rancher.io/containers/alertmanager",
	"rancher/appco-grafana":                                           "dp.apps.rancher.io/containers/grafana",
	"rancher/appco-k8s-sidecar":                                       "dp.apps.rancher.io/containers/k8s-sidecar",
	"rancher/appco-kubernetes-cluster-autoscaler":                     "dp.apps.rancher.io/containers/kubernetes-cluster-autoscaler",
	"rancher/appco-kube-rbac-proxy":                                   "dp.apps.rancher.io/containers/kube-rbac-proxy",
	"rancher/appco-kube-state-metrics":                                "dp.apps.rancher.io/containers/kube-state-metrics",
	"rancher/appco-redis":                                             "dp.apps.rancher.io/containers/redis",
	"rancher/appco-node-exporter":                                     "dp.apps.rancher.io/containers/node-exporter",
	"rancher/appco-thanos":                                            "dp.apps.rancher.io/containers/thanos",
	"rancher/charts/appco-kubernetes-cluster-autoscaler":              "dp.apps.rancher.io/charts/kubernetes-cluster-autoscaler",
	"rancher/mirrored-cilium-cilium":                                  "quay.io/cilium/cilium",
	"rancher/mirrored-cilium-cilium-envoy":                            "quay.io/cilium/cilium-envoy",
	"rancher/mirrored-cilium-clustermesh-apiserver":                   "quay.io/cilium/clustermesh-apiserver",
	"rancher/mirrored-cilium-hubble-relay":                            "quay.io/cilium/hubble-relay",
	"rancher/mirrored-cilium-operator-aws":                            "quay.io/cilium/operator-aws",
	"rancher/mirrored-cilium-operator-azure":                          "quay.io/cilium/operator-azure",
	"rancher/mirrored-cilium-operator-generic":                        "quay.io/cilium/operator-generic",
	"rancher/mirrored-prometheus-operator-admission-webhook":          "quay.io/prometheus-operator/admission-webhook",
	"rancher/mirrored-prometheus-operator-prometheus-config-reloader": "quay.io/prometheus-operator/prometheus-config-reloader",
	"rancher/mirrored-prometheus-operator-prometheus-operator":        "quay.io/prometheus-operator/prometheus-operator",
	"rancher/mirrored-kube-logging-config-reloader":                   "ghcr.io/kube-logging/config-reloader",
	"rancher/mirrored-kube-logging-logging-operator":                  "ghcr.io/kube-logging/logging-operator",
	"rancher/mirrored-jkroepke-kube-webhook-certgen":                  "ghcr.io/jkroepke/kube-webhook-certgen",
	"rancher/mirrored-kube-state-metrics-kube-state-metrics":          "registry.k8s.io/kube-state-metrics/kube-state-metrics",
	"rancher/mirrored-elemental-operator":                             "registry.suse.com/rancher/elemental-operator",
	"rancher/mirrored-elemental-seedimage-builder":                    "registry.suse.com/rancher/seedimage-builder",
	"rancher/mirrored-cluster-api-controller":                         "registry.k8s.io/cluster-api/cluster-api-controller",
	"rancher/mirrored-curlimages-curl":                                "curlimages/curl",
	"rancher/mirrored-fluent-fluent-bit":                              "fluent/fluent-bit",
	"rancher/mirrored-bci-micro":                                      "registry.suse.com/bci/bci-micro",
	"rancher/mirrored-cloud-provider-vsphere-csi-release-driver":      "registry.k8s.io/csi-vsphere/driver",
	"rancher/mirrored-cloud-provider-vsphere-csi-release-syncer":      "registry.k8s.io/csi-vsphere/syncer",
	"rancher/mirrored-cloud-provider-vsphere":                         "registry.k8s.io/cloud-pv-vsphere/cloud-provider-vsphere",
	"rancher/mirrored-metrics-server":                                 "registry.k8s.io/metrics-server/metrics-server",
	"rancher/mirrored-pause":                                          "registry.k8s.io/pause",
}

type imageCopier struct {
	m          sync.Mutex
	copyImages bool
}

func (i *imageCopier) Copy(srcImg, dstRegistry string) Entry {
	entry := Entry{
		Image: srcImg,
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

// signatureRepo returns the fully qualified repository where the signatures
// for srcRef are published: the upstream repository for external images,
// or the source repository itself.
func signatureRepo(srcRef name.Reference) string {
	repo := srcRef.Context().RepositoryStr()
	if upstream, found := externalImages[repo]; found {
		return upstream
	}
	return srcRef.Context().Name()
}

func signatureSource(srcRef name.Reference, tag string) string {
	// Fully qualified reference: <registry>/<repository>:<signature_tag>
	return fmt.Sprintf("%s:%s", signatureRepo(srcRef), tag)
}

// CopySignature copies a container image with its cosign signature from source to target registry.
// Supports both the legacy format (sha256-<digest>.sig tag) and OCI 1.1 referrers (cosign v3
// bundles), discovered via the Referrers API or the referrers tag schema fallback. Source and
// target tags must match since signatures are bound to content digests. Uses NoClobber to
// prevent overwriting existing tags.
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

	// copy image only after all safety checks but before checking signatures
	if copyImage {
		err := copyArtifact(ctx, srcImgRef, dstImgRef)
		if err != nil {
			return err
		}
	}

	hex := strings.TrimPrefix(digest, "sha256:")
	signatureTag := fmt.Sprintf("sha256-%s.sig", hex)
	sourceSigRef := signatureSource(sourceRef, signatureTag)

	var errList []error

	// Legacy format: signature stored under the sha256-<digest>.sig tag.
	_, err = crane.Manifest(sourceSigRef, crane.WithContext(ctx))
	if err == nil {
		dstSigRef := fmt.Sprintf("%s:%s", targetRef.Context().Name(), signatureTag)
		return copyArtifact(ctx, sourceSigRef, dstSigRef)
	}
	errList = append(errList, err)

	// Try the no-suffix format. The same tag is also used for the OCI referrers
	// fallback index, which must be handled by copyReferrers instead.
	signatureTag = strings.TrimSuffix(signatureTag, ".sig")
	sourceSigRef = strings.TrimSuffix(sourceSigRef, ".sig")
	plainSigRef, err := name.ParseReference(sourceSigRef)
	if err == nil {
		desc, getErr := remote.Get(plainSigRef,
			remote.WithContext(ctx),
			remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if getErr != nil {
			err = getErr
		} else if desc.MediaType.IsIndex() {
			err = fmt.Errorf("%q is an OCI image index, not a standalone signature", sourceSigRef)
		} else {
			dstSigRef := fmt.Sprintf("%s:%s", targetRef.Context().Name(), signatureTag)
			return copyArtifact(ctx, sourceSigRef, dstSigRef)
		}
	}
	errList = append(errList, err)

	// OCI 1.1 format: signatures attached to the digest as referrers.
	err = copyReferrers(ctx, signatureRepo(sourceRef), targetRef.Context().Name(), digest)
	if err == nil {
		return nil
	}

	if !errors.Is(err, errNoReferrers) {
		// discovery or transfer failure: not a confirmed absence of signatures
		return err
	}
	errList = append(errList, err)

	return fmt.Errorf("%w: %w", ErrNoSignaturesFound, errors.Join(errList...))
}

// copyReferrers copies the signature manifests referring to digest in srcRepo into dstRepo,
// by digest using the Referrers API. Other referrers (SBOMs, attestations) are ignored.
// Returns errNoReferrers when no signature referrer exists.
func copyReferrers(ctx context.Context, srcRepo, dstRepo, digest string) error {
	subject, err := name.NewDigest(fmt.Sprintf("%s@%s", srcRepo, digest))
	if err != nil {
		return fmt.Errorf("failed to parse subject digest: %w", err)
	}

	idx, err := remote.Referrers(subject,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("failed to list referrers for %q: %w", subject, err)
	}

	m, err := idx.IndexManifest()
	if err != nil {
		return fmt.Errorf("failed to read referrers for %q: %w", subject, err)
	}

	found := false

	for _, desc := range m.Manifests {
		isSignature, err := isSignatureArtifact(ctx, srcRepo, desc)
		if err != nil {
			return err
		}
		if !isSignature {
			continue
		}

		found = true
		src := fmt.Sprintf("%s@%s", srcRepo, desc.Digest)
		dst := fmt.Sprintf("%s@%s", dstRepo, desc.Digest)

		err = copyArtifact(ctx, src, dst)
		if err != nil {
			return err
		}
	}

	if !found {
		return fmt.Errorf("%w for %q", errNoReferrers, subject)
	}

	return nil
}

func isSignatureArtifact(ctx context.Context, srcRepo string, desc v1.Descriptor) (bool, error) {
	desc, err := resolveReferrerDescriptor(ctx, srcRepo, desc)
	if err != nil {
		return false, err
	}

	if desc.ArtifactType == cosignSignatureType {
		return true, nil
	}
	if !strings.HasPrefix(desc.ArtifactType, sigstoreBundleTypePrefix) {
		return false, nil
	}

	if isSignature, decided := isSignatureBundleAnnotations(desc.Annotations); decided {
		return isSignature, nil
	}

	ref, err := name.NewDigest(fmt.Sprintf("%s@%s", srcRepo, desc.Digest))
	if err != nil {
		return false, fmt.Errorf("failed to parse bundle digest: %w", err)
	}
	bundle, err := remote.Get(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return false, fmt.Errorf("failed to fetch signature bundle %q: %w", ref, err)
	}

	image, err := bundle.Image()
	if err != nil {
		return false, fmt.Errorf("failed to read signature bundle image %q: %w", ref, err)
	}
	manifest, err := image.Manifest()
	if err != nil {
		return false, fmt.Errorf("failed to read signature bundle manifest %q: %w", ref, err)
	}
	if isSignature, decided := isSignatureBundleAnnotations(manifest.Annotations); decided {
		return isSignature, nil
	}

	layers, err := image.Layers()
	if err != nil {
		return false, fmt.Errorf("failed to read signature bundle layers %q: %w", ref, err)
	}

	for _, layer := range layers {
		reader, err := layer.Uncompressed()
		if err != nil {
			continue
		}
		var content struct {
			MessageSignature json.RawMessage `json:"messageSignature"`
			DSSEEnvelope     json.RawMessage `json:"dsseEnvelope"`
		}
		decodeErr := json.NewDecoder(reader).Decode(&content)
		_ = reader.Close()
		if decodeErr == nil {
			if len(content.DSSEEnvelope) > 0 {
				// a DSSE envelope is a signature or an attestation depending on its predicate
				return dssePredicateType(content.DSSEEnvelope) == cosignSignPredicateType, nil
			}
			if len(content.MessageSignature) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

// ociEmptyMediaType is the media type of the empty config of artifact manifests.

// resolveReferrerDescriptor returns desc with the real artifactType and annotations of the
// referrer. Registries that serve referrers through the tag schema fallback (e.g. ghcr.io,
// registry.rancher.com) store an index whose descriptors carry the config media type
// (application/vnd.oci.empty.v1+json) instead of the artifactType, and no annotations, so
// in that case the referrer manifest is read.
func resolveReferrerDescriptor(ctx context.Context, srcRepo string, desc v1.Descriptor) (v1.Descriptor, error) {
	if desc.ArtifactType != "" && desc.ArtifactType != ociEmptyMediaType {
		return desc, nil
	}

	ref, err := name.NewDigest(fmt.Sprintf("%s@%s", srcRepo, desc.Digest))
	if err != nil {
		return desc, fmt.Errorf("failed to parse referrer digest: %w", err)
	}

	raw, err := remote.Get(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return desc, fmt.Errorf("failed to fetch referrer manifest %q: %w", ref, err)
	}

	manifest, err := v1.ParseManifest(bytes.NewReader(raw.Manifest))
	if err != nil {
		return desc, fmt.Errorf("failed to parse referrer manifest %q: %w", ref, err)
	}

	desc.ArtifactType = manifest.ArtifactType
	if desc.ArtifactType == "" || desc.ArtifactType == ociEmptyMediaType {
		// no artifactType: the artifact is identified by its (single) layer media type
		if len(manifest.Layers) == 1 {
			desc.ArtifactType = string(manifest.Layers[0].MediaType)
		} else {
			desc.ArtifactType = string(manifest.Config.MediaType)
		}
	}
	if len(desc.Annotations) == 0 {
		desc.Annotations = manifest.Annotations
	}

	return desc, nil
}

// isSignatureBundleAnnotations classifies a sigstore bundle from its annotations.
// decided is false when the annotations are not conclusive and the bundle must be inspected.
func isSignatureBundleAnnotations(annotations map[string]string) (isSignature, decided bool) {
	switch predicate := annotations[sigstoreBundlePredicateAnnotation]; predicate {
	case "":
	case cosignSignPredicateType:
		return true, true
	default:
		return false, true
	}

	if annotations[sigstoreBundleContentAnnotation] == "message-signature" {
		return true, true
	}

	// "dsse-envelope" alone says nothing: cosign v3 signatures are DSSE envelopes too.
	return false, false
}

// dssePredicateType returns the in-toto predicateType carried by a DSSE envelope payload,
// or an empty string when it cannot be read.
func dssePredicateType(envelope json.RawMessage) string {
	var env struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(envelope, &env); err != nil {
		return ""
	}

	payload, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		payload, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(env.Payload, "="))
		if err != nil {
			return ""
		}
	}

	var statement struct {
		PredicateType string `json:"predicateType"`
	}
	if err := json.Unmarshal(payload, &statement); err != nil {
		return ""
	}

	return statement.PredicateType
}

func copyArtifact(ctx context.Context, src, dst string) error {
	err := crane.Copy(src, dst,
		crane.WithContext(ctx),
		crane.WithNoClobber(true)) // ensures won't be overwritten.

	if err != nil && !strings.Contains(err.Error(), "refusing to clobber existing tag") {
		return fmt.Errorf("failed to copy from %q to %q: %w",
			src, dst, err)
	}

	return nil
}
