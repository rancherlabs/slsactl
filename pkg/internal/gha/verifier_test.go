package gha

import (
	"context"
	"crypto"
	"errors"
	"testing"

	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMatches(t *testing.T) {
	t.Parallel()

	registries := []string{"", "docker.io/"}
	tests := []struct {
		image       string
		want        bool
		wantKey     string
		hasRegistry bool
	}{
		{image: "foo/bar", want: true},
		{image: "bar/foo", want: true},
		{image: "rancher/foo", want: true},
	}

	for _, registry := range registries {
		for _, tc := range tests {
			image := tc.image
			if !tc.hasRegistry {
				image = registry + tc.image
			} else if registry != "" {
				continue
			}

			t.Run(image, func(t *testing.T) {
				t.Parallel()

				v := Verifier{}
				got := v.Matches(image)
				assert.Equal(t, tc.want, got)
			})
		}
	}
}

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
			image: "rancher/rke2-upgrade:v1.31.14-rke2r1",
			want:  "^https://github.com/rancher/rke2-upgrade/.github/workflows/release.(yml|yaml)@refs/tags/v1.31.14(\\+|&#43;)rke2r1$",
		},
		{
			image: "rancher/system-agent-installer-k3s:v1.31.14-k3s1",
			want:  "^https://github.com/rancher/system-agent-installer-k3s/.github/workflows/release.(yml|yaml)@refs/tags/v1.31.14(\\+|&#43;)k3s1$",
		},
		{
			image: "rancher/system-agent-installer-k3s:v1.31.14-k3s1-linux-amd64",
			want:  "^https://github.com/rancher/system-agent-installer-k3s/.github/workflows/release.(yml|yaml)@refs/tags/v1.31.14(\\+|&#43;)k3s1$",
		},
		{
			image: "rancher/system-agent-installer-rke2:v1.31.14-rke2r1",
			want:  "^https://github.com/rancher/system-agent-installer-rke2/.github/workflows/release.(yml|yaml)@refs/tags/v1.31.14(\\+|&#43;)rke2r1$",
		},
		{
			image: "rancher/system-agent-installer-rke2:v1.31.14-rke2r1-linux-amd64",
			want:  "^https://github.com/rancher/system-agent-installer-rke2/.github/workflows/release.(yml|yaml)@refs/tags/v1.31.14(\\+|&#43;)rke2r1$",
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
			image: "rancher/hardened-etcd:v3.5.16-k3s1-build20241106",
			want:  "^https://github.com/rancher/image-build-etcd/.github/workflows/(image-push|release).yml@refs/tags/v",
		},
		{
			image: "rancher/hardened-multus-dynamic-networks-controller:v0.3.7-build20250711",
			want:  "^https://github.com/rancher/image-build-multus-dynamic-networks-controller/.github/workflows/release.(yml|yaml)@refs/tags/v0.3.7-build20250711$",
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
		{
			image: "rancher/mirrored-cilium-cilium:v1.17.0",
			want:  "^https://github.com/cilium/cilium/.github/workflows/build-images-releases.yaml@refs/tags/v",
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

func TestVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		context context.Context //nolint
		image   string
		setup   func(*upstreamMock)
		wantErr error
	}{
		{
			name:    "Missing tag",
			context: context.TODO(),
			image:   "rancher/foo",
			wantErr: errors.New("missing image tag"),
		},
		{
			name:    "Success",
			context: context.TODO(),
			image:   "rancher/foo:v2.7.1",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					CertVerifyOptions: options.CertVerifyOptions{
						CertIdentityRegexp: "^https://github.com/rancher/foo/.github/workflows/release.(yml|yaml)@refs/tags/v2.7.1$",
						CertOidcIssuer:     "https://token.actions.githubusercontent.com",
					},
					RekorURL:        options.DefaultRekorURL,
					CheckClaims:     true,
					HashAlgorithm:   crypto.SHA256,
					NewBundleFormat: true,
				}
				m.On("Verify", context.TODO(), vc, "rancher/foo:v2.7.1").Return(nil)
			},
		},
		{
			name:    "Return upstream error",
			context: context.TODO(),
			image:   "rancher/foo:v2.7.1",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					CertVerifyOptions: options.CertVerifyOptions{
						CertIdentityRegexp: "^https://github.com/rancher/foo/.github/workflows/release.(yml|yaml)@refs/tags/v2.7.1$",
						CertOidcIssuer:     "https://token.actions.githubusercontent.com",
					},
					RekorURL:        options.DefaultRekorURL,
					CheckClaims:     true,
					HashAlgorithm:   crypto.SHA256,
					NewBundleFormat: true,
				}
				m.On("Verify", context.TODO(), vc, "rancher/foo:v2.7.1").Return(
					errors.New(`upstream failure`))
			},
			wantErr: errors.New(`upstream failure`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := new(upstreamMock)
			sut := &Verifier{
				HashAlgorithm:    crypto.SHA256,
				UpstreamVerifier: m,
			}

			if tc.setup != nil {
				tc.setup(m)
			}

			err := sut.Verify(tc.context, tc.image)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr.Error())
			}

			m.AssertExpectations(t)
		})
	}
}

type upstreamMock struct {
	mock.Mock
}

func (m *upstreamMock) Verify(ctx context.Context, vc verify.VerifyCommand, image string) error {
	args := m.Called(ctx, vc, image)

	return args.Error(0)
}
