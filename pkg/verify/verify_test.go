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
			image: "foo/bar:v0.0.7",
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "foo/bar:v0.0.7-rke2foo2",
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7&#43;rke2foo2",
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
			want:  "https://github.com/bar/foo/.github/workflows/release.yml@refs/tags/v3.14",
		},
		{
			image:   "foo:bar",
			wantErr: "unsupported image name",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7",
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/bar/foo/bar:v3.14",
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v3.14",
		},
		{
			image: "rancher/rke2-runtime:v0.0.7",
			want:  "https://github.com/rancher/rke2/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7-amd64", // single tag may yield arch-specific images
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7-arm64", // single tag may yield arch-specific images
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7-windows-amd64", // single tag may yield arch-specific images
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7-windows-arm64", // single tag may yield arch-specific images
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7",
		},
		{
			image: "tocker.local/foo/bar:v0.0.7-build12345",
			want:  "https://github.com/foo/bar/.github/workflows/release.yml@refs/tags/v0.0.7-build12345",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.image, func(t *testing.T) {
			t.Parallel()

			got, err := certIdentity(tc.image)

			assert.Equal(t, tc.want, got)

			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
