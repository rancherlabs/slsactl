package verify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCertificateIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		image   string
		want    string
		wantErr string
	}{
		{
			image: "rancher/rke2:v0.0.7",
			want:  "^https://github.com/rancher/rke2/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rancher/rke2:v0.0.7-rke2foo2",
			want:  "^https://github.com/rancher/rke2/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7(\\+|&#43;)rke2foo2$",
		},
		{
			image: "rancher/hardened-kubernetes:v1.32.3-rke2r1-build20250312",
			want:  "^https://github.com/rancher/image-build-kubernetes/.github/workflows/release.(yml|yaml)@refs/tags/v1.32.3-rke2r1-build20250312$",
		},
		{
			image: "rancher/hardened-multus-cni:v1.32.3-arch",
			want:  "^https://github.com/rancher/image-build-multus/.github/workflows/release.(yml|yaml)@refs/tags/v1.32.3$",
		},
		{
			image:   "",
			wantErr: "invalid image name",
		},
		{
			image:   "foo/bar",
			wantErr: "missing image tag",
		},
		{
			image:   "foo/bar:",
			wantErr: "missing image tag",
		},
		{
			image:   "foo/bar@sha256:a32d91ba265e6fcb1963c28bb688d0b799a1966f30f6ea17d8eca1d436bbc267",
			wantErr: "missing image tag",
		},
		{
			image: "bar/foo:v3.14@sha256:a32d91ba265e6fcb1963c28bb688d0b799a1966f30f6ea17d8eca1d436bbc267",
			want:  "^https://github.com/bar/foo/.github/workflows/release.(yml|yaml)@refs/tags/v3.14$",
		},
		{
			image:   "foo:bar",
			wantErr: "unsupported image name",
		},
		{
			image: "localhost:5000/foo/bar:v0.0.7",
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7",
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/bar/foo/bar:v3.14",
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v3.14$",
		},
		{
			image: "rancher/rke2-runtime:v0.0.7",
			want:  "^https://github.com/rancher/rke2/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-amd64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-arm64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-s390x", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-windows-amd64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-windows-arm64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-linux-amd64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-linux-arm64", // single tag may yield arch-specific images
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7$",
		},
		{
			image: "rocker.local/foo/bar:v0.0.7-build12345",
			want:  "^https://github.com/foo/bar/.github/workflows/release.(yml|yaml)@refs/tags/v0.0.7-build12345$",
		},
		{
			image: "rancher/neuvector-controller:5.4.2",
			want:  "^https://github.com/neuvector/neuvector/.github/workflows/release.(yml|yaml)@refs/tags/v5.4.2$",
		},
		{
			image: "rancher/neuvector-scanner:3.685",
			want:  "^https://github.com/neuvector/scanner/.github/workflows/release.(yml|yaml)@refs/tags/v3.685$",
		},
	}

	for _, tc := range tests {
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()

			got, err := getCertIdentity(tc.image)

			assert.Equal(t, tc.want, got)

			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestObsSigned(t *testing.T) {
	t.Parallel()

	tests := []struct {
		image   string
		want    bool
		wantKey string
	}{
		{image: "rancher/elemental-operator", want: true, wantKey: obsKey},
		{image: "rancher/seedimage-builder", want: true, wantKey: obsKey},
		{image: "rancher/elemental-channel/sl-micro", want: true, wantKey: obsKey},
		{image: "rancher/elemental-operator-crds-chart", want: true, wantKey: obsKey},
		{image: "rancher/elemental-operator-chart", want: true, wantKey: obsKey},
		{image: "suse/sles/15.7/foo", want: true, wantKey: obsKey},
		{image: "bci/foo-bar", want: true, wantKey: obsKey},
		{image: "rancher/mirrored-bci-busybox", want: true, wantKey: obsKey},
		{image: "rancher/appco-something", want: true, wantKey: appCoKey},
		{image: "rancher/rancher"},
		{image: "ghcr.io/kubewarden/policy-server"},
		{image: "fuzz/bar"},
	}

	for _, tc := range tests {
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()
			gotKey, got := obsSigned(tc.image)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantKey, gotKey)
		})
	}
}
