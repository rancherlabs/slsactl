package verify

import (
	"context"
	"errors"
	"testing"

	"github.com/rancherlabs/slsactl/pkg/internal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVerify(t *testing.T) {
	t.Parallel()

	empty := func(t *testing.T) {}
	tests := []struct {
		name      string
		image     string
		verifiers func() ([]internal.Verifier, func(t *testing.T))
		wantErr   error
	}{
		{
			name:  "No verifiers",
			image: "suse/sles",
			verifiers: func() ([]internal.Verifier, func(t *testing.T)) {
				return nil, empty
			},
			wantErr: errors.New("no verifier found for image: \"suse/sles\""),
		},
		{
			name:  "No matching verifiers",
			image: "suse/sles",
			verifiers: func() ([]internal.Verifier, func(t *testing.T)) {
				m1 := &verifierMock{}

				m1.On("Matches", "suse/sles").Return(false)

				return []internal.Verifier{m1}, func(t *testing.T) {
					m1.AssertExpectations(t)
				}
			},
			wantErr: errors.New("no verifier found for image: \"suse/sles\""),
		},
		{
			name:  "Matching verifier",
			image: "suse/sles",
			verifiers: func() ([]internal.Verifier, func(t *testing.T)) {
				m1 := &verifierMock{}

				m1.On("Matches", "suse/sles").Return(true)
				m1.On("Verify", mock.Anything, "suse/sles").Return(nil)

				return []internal.Verifier{m1}, func(t *testing.T) {
					m1.AssertExpectations(t)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, a := tc.verifiers()
			verifiers = v

			err := Verify(tc.image)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr.Error())
			}

			a(t)
		})
	}
}

type verifierMock struct {
	mock.Mock
}

func (m *verifierMock) Matches(image string) bool {
	args := m.Called(image)

	return args.Bool(0)
}

func (m *verifierMock) Verify(ctx context.Context, image string) error {
	args := m.Called(ctx, image)

	return args.Error(0)
}
