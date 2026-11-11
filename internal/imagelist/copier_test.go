package imagelist

import (
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideSignatureSource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		image string
		want  string
	}{
		{
			image: "rancher/rancher",
			want:  "index.docker.io/rancher/rancher",
		},
		{
			image: "127.0.0.1:5000/sig-storage/snapshot-controller",
			want:  "registry.k8s.io/sig-storage/snapshot-controller",
		},
		{
			image: "127.0.0.1:5000/sig-storage/snapshot-validation-webhook",
			want:  "registry.k8s.io/sig-storage/snapshot-validation-webhook",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-csi-node-driver-registrar",
			want:  "registry.k8s.io/sig-storage/csi-node-driver-registrar",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-csi-attacher",
			want:  "registry.k8s.io/sig-storage/csi-attacher",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-csi-provisioner",
			want:  "registry.k8s.io/sig-storage/csi-provisioner",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-csi-resizer",
			want:  "registry.k8s.io/sig-storage/csi-resizer",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-csi-snapshotter",
			want:  "registry.k8s.io/sig-storage/csi-snapshotter",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-livenessprobe",
			want:  "registry.k8s.io/sig-storage/livenessprobe",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-snapshot-controller",
			want:  "registry.k8s.io/sig-storage/snapshot-controller",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-longhornio-csi-attacher",
			want:  "docker.io/longhornio/csi-attacher",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-longhornio-longhorn-manager",
			want:  "docker.io/longhornio/longhorn-manager",
		},
		{
			image: "127.0.0.1:5000/rancher/appco-alertmanager",
			want:  "dp.apps.rancher.io/containers/alertmanager",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-cilium-cilium",
			want:  "quay.io/cilium/cilium",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-prometheus-operator-prometheus-operator",
			want:  "quay.io/prometheus-operator/prometheus-operator",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-jkroepke-kube-webhook-certgen",
			want:  "ghcr.io/jkroepke/kube-webhook-certgen",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-curlimages-curl",
			want:  "curlimages/curl",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-fluent-fluent-bit",
			want:  "fluent/fluent-bit",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-bci-micro",
			want:  "registry.suse.com/bci/bci-micro",
		},
	}

	for _, tc := range tests {
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()

			tag := "sha256-aabbeedd.sig"
			srcRef, err := name.ParseReference(tc.image + ":" + tag)
			require.NoError(t, err)

			got := signatureSource(srcRef, tag)
			assert.Equal(t, tc.want+":"+tag, got)
		})
	}
}

// TestExternalImagesEntries guards against malformed values in externalImages
// (stray whitespace, embedded tags or digests) that only show up when
// signatures are looked up against a real registry.
func TestExternalImagesEntries(t *testing.T) {
	t.Parallel()

	for key, value := range externalImages {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, strings.TrimSpace(key), key, "key has surrounding whitespace")
			assert.Equal(t, strings.TrimSpace(value), value, "value has surrounding whitespace")
			assert.NotContains(t, value, "@", "value must not contain a digest")

			// NewRepository rejects references carrying a tag or digest.
			_, err := name.NewRepository(value)
			assert.NoError(t, err, "value must be a plain repository")
		})
	}
}
