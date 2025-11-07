package imagelist

import (
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
			want:  "registry.k8s.io/sig-storage/rancher/mirrored-sig-storage-livenessprobe",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-sig-storage-snapshot-controller",
			want:  "registry.k8s.io/sig-storage/snapshot-controller",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-longhornio-csi-attacher",
			want:  "registry.k8s.io/sig-storage/csi-attacher",
		},
		{
			image: "127.0.0.1:5000/rancher/appco-redis",
			want:  "dp.apps.rancher.io/containers/redis",
		},
		{
			image: "127.0.0.1:5000/rancher/mirrored-cilium-cilium",
			want:  "quay.io/cilium/cilium",
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
