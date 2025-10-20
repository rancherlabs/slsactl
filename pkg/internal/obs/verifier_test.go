package obs_test

import (
	"context"
	"crypto"
	"errors"
	"testing"

	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/rancherlabs/slsactl/pkg/internal/obs"

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
		{image: "rancher/elemental-operator", want: true},
		{image: "rancher/seedimage-builder", want: true},
		{image: "rancher/elemental-channel/sl-micro", want: true},
		{image: "rancher/elemental-operator-crds-chart", want: true},
		{image: "rancher/elemental-operator-chart", want: true},
		{image: "suse/sles/15.7/foo", want: true},
		{image: "bci/foo-bar", want: true},
		{image: "rancher/mirrored-bci-busybox", want: true},
		{image: "registry.suse.com/bar/fuzz", want: true, hasRegistry: true},
		{image: "rancher/appco-something"},
		{image: "rancher/rancher"},
		{image: "fuzz/bar"},
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

				v := obs.Verifier{}
				got := v.Matches(image)
				assert.Equal(t, tc.want, got)
			})
		}
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
			name:    "Invalid image",
			context: context.TODO(),
			image:   "foo/bar",
			wantErr: internal.ErrInvalidImage,
		},
		{
			name:    "Success",
			context: context.TODO(),
			image:   "suse/sles",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					KeyRef:        "https://ftp.suse.com/pub/projects/security/keys/container-key.pem",
					CertRef:       "https://ftp.suse.com/pub/projects/security/keys/container-key.pem",
					RekorURL:      options.DefaultRekorURL,
					CheckClaims:   true,
					HashAlgorithm: crypto.SHA256,
				}
				m.On("Verify", context.TODO(), vc, "suse/sles").Return(nil)
			},
		},
		{
			name:    "Return upstream error",
			context: context.TODO(),
			image:   "suse/sles",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					KeyRef:        "https://ftp.suse.com/pub/projects/security/keys/container-key.pem",
					CertRef:       "https://ftp.suse.com/pub/projects/security/keys/container-key.pem",
					RekorURL:      options.DefaultRekorURL,
					CheckClaims:   true,
					HashAlgorithm: crypto.SHA256,
				}
				m.On("Verify", context.TODO(), vc, "suse/sles").Return(
					errors.New(`upstream failure`))
			},
			wantErr: errors.New(`upstream failure`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := new(upstreamMock)
			sut := &obs.Verifier{
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
