package imagelist

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		setup   func(*DepsMock)
		want    *Result
		wantErr error
	}{
		{
			name: "two images",
			url:  "https://.../image.txt",
			setup: func(m *DepsMock) {
				m.On("Fetch", "https://.../image.txt").Return(
					io.NopCloser(bytes.NewReader(
						[]byte("image:v1\nimage2:v2\n"),
					)), nil)

				m.On("Process", "image:v1").Return(Entry{
					Image: "image:v1"})
				m.On("Process", "image2:v2").Return(Entry{
					Image: "image2:v2"})
			},
			want: &Result{
				Entries: []Entry{
					{Image: "image:v1"},
					{Image: "image2:v2"},
				},
			},
		},
		{
			name: "ignore empty lines",
			url:  "https://.../image.txt",
			setup: func(m *DepsMock) {
				m.On("Fetch", "https://.../image.txt").Return(
					io.NopCloser(bytes.NewReader(
						[]byte("\n\nimage:v1\n"),
					)), nil)

				m.On("Process", "image:v1").Return(Entry{
					Image: "image:v1"})
			},
			want: &Result{
				Entries: []Entry{
					{Image: "image:v1"},
				},
			},
		},
		{
			name: "no images found",
			url:  "https://.../image.txt",
			setup: func(m *DepsMock) {
				m.On("Fetch", "https://.../image.txt").Return(
					io.NopCloser(bytes.NewReader(
						[]byte("\n\n\n\n"),
					)), nil)
			},
			wantErr: ErrNoImagesFound,
		},
		{
			name: "continue to process on error",
			url:  "https://.../image.txt",
			setup: func(m *DepsMock) {
				m.On("Fetch", "https://.../image.txt").Return(
					io.NopCloser(bytes.NewReader(
						[]byte("image:v1\nimage2:v2\n"),
					)), nil)

				m.On("Process", "image:v1").Return(Entry{
					Image: "image:v1", Error: fmt.Errorf("image not found")})
				m.On("Process", "image2:v2").Return(Entry{
					Image: "image2:v2"})
			},
			want: &Result{
				Entries: []Entry{
					{Image: "image:v1", Error: fmt.Errorf("image not found")},
					{Image: "image2:v2"},
				},
			},
		},
		{
			name: "empty URL",
			url:  "",
			setup: func(m *DepsMock) {
			},
			wantErr: ErrURLCannotBeEmpty,
		},
		{
			name: "fetching errors",
			url:  "https://.../image.txt",
			setup: func(m *DepsMock) {
				m.On("Fetch", "https://.../image.txt").Return(nil, fmt.Errorf("not able to fetch resource"))
			},
			wantErr: ErrCannotFetchURL,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := new(DepsMock)

			sut := NewProcessor("some.registry")

			tc.setup(m)
			sut.fetcher = m
			sut.ip = m

			got, err := sut.Process(tc.url)

			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.EqualExportedValues(t, tc.want, got)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			}

			m.AssertExpectations(t)
		})
	}
}

type DepsMock struct {
	mock.Mock
}

func (m *DepsMock) Process(img string) Entry {
	args := m.Called(img)
	return args.Get(0).(Entry)
}

func (m *DepsMock) Fetch(img string) (io.ReadCloser, error) {
	args := m.Called(img)

	var r io.ReadCloser
	if v, ok := args.Get(0).(io.ReadCloser); ok {
		r = v
	}

	return r, args.Error(1)
}
