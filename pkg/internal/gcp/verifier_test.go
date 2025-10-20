package gcp_test

import (
	"context"
	"crypto"
	"fmt"
	"testing"

	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/rancherlabs/slsactl/pkg/internal/gcp"

	"github.com/sigstore/cosign/v3/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
	cosign "github.com/sigstore/cosign/v3/cmd/cosign/cli/verify"
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
		{image: "sig-storage/snapshot-controller", want: true},
		{image: "sig-storage/snapshot-validation-webhook", want: true},
		{image: "rancher/mirrored-sig-storage-csi-node-driver-registrar", want: true},
		{image: "rancher/mirrored-sig-storage-csi-attacher", want: true},
		{image: "rancher/mirrored-sig-storage-csi-provisioner", want: true},
		{image: "rancher/mirrored-sig-storage-csi-resizer", want: true},
		{image: "rancher/mirrored-sig-storage-csi-snapshotter", want: true},
		{image: "rancher/mirrored-sig-storage-livenessprobe", want: true},
		{image: "rancher/mirrored-sig-storage-snapshot-controller", want: true},
		{image: "rancher/mirrored-kube-state-metrics-kube-state-metrics", want: true},
		{image: "rancher/mirrored-cluster-api-controller", want: true},
		{image: "registry.k8s.io/bar/fuzz", want: true, hasRegistry: true},
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

				v := gcp.Verifier{}
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
		context context.Context
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
			image:   "sig-storage/snapshot-controller",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					CertVerifyOptions: options.CertVerifyOptions{
						CertIdentityRegexp: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
						CertOidcIssuer:     "https://accounts.google.com",
					},
					RekorURL:      options.DefaultRekorURL,
					CheckClaims:   true,
					HashAlgorithm: crypto.SHA256,
				}
				m.On("Verify", context.TODO(), vc, "sig-storage/snapshot-controller").Return(nil)
			},
		},
		{
			name:    "Return upstream error",
			context: context.TODO(),
			image:   "sig-storage/snapshot-controller",
			setup: func(m *upstreamMock) {
				vc := verify.VerifyCommand{
					CertVerifyOptions: options.CertVerifyOptions{
						CertIdentityRegexp: "krel-trust@k8s-releng-prod.iam.gserviceaccount.com",
						CertOidcIssuer:     "https://accounts.google.com",
					},
					RekorURL:      options.DefaultRekorURL,
					CheckClaims:   true,
					HashAlgorithm: crypto.SHA256,
				}
				m.On("Verify", context.TODO(), vc, "sig-storage/snapshot-controller").Return(
					fmt.Errorf(`upstream failure`))
			},
			wantErr: fmt.Errorf(`upstream failure`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := new(upstreamMock)
			sut := &gcp.Verifier{
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

func (m *upstreamMock) Verify(ctx context.Context, vc cosign.VerifyCommand, image string) error {
	args := m.Called(ctx, vc, image)

	return args.Error(0)
}
