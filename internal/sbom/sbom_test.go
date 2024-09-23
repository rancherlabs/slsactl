package sbom

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/anchore/syft/syft/sbom"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

//go:embed cis-operator-amd64.spdx
var cisoperatorAMD64Spdx string

func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		img        string
		outformat  string
		createSBOM func(img string) (*sbom.SBOM, error)
		wantErr    bool
	}{
		{
			name:      "successful generation with cyclonedxjson",
			img:       "test-image",
			outformat: "cyclonedxjson",
			createSBOM: func(img string) (*sbom.SBOM, error) {
				return &sbom.SBOM{}, nil
			},
			wantErr: false,
		},
		{
			name:      "successful generation with spdxjson",
			img:       "test-image",
			outformat: "spdxjson",
			createSBOM: func(img string) (*sbom.SBOM, error) {
				return &sbom.SBOM{}, nil
			},
			wantErr: false,
		},
		{
			name:      "invalid format",
			img:       "test-image",
			outformat: "invalidformat",
			createSBOM: func(img string) (*sbom.SBOM, error) {
				return &sbom.SBOM{}, nil
			},
			wantErr: true,
		},
		{
			name:      "failed to get source",
			img:       "test-image",
			outformat: "cyclonedxjson",
			createSBOM: func(img string) (*sbom.SBOM, error) {
				return nil, fmt.Errorf("failed to get source")
			},
			wantErr: true,
		},
		{
			name:      "failed to create SBOM",
			img:       "test-image",
			outformat: "cyclonedxjson",
			createSBOM: func(img string) (*sbom.SBOM, error) {
				return nil, fmt.Errorf("failed to generate SBOM")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createSBOM = tt.createSBOM

			var buf bytes.Buffer
			err := Generate(tt.img, tt.outformat, &buf)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
	createSBOM = defaultCreateSBOM
}

func TestConvertToCyclonedxJson(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid SPDX",
			input:   cisoperatorAMD64Spdx,
			wantErr: false,
		},
		{
			name:    "invalid input format",
			input:   `<not valid>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tt.input))
			var writer bytes.Buffer

			err := ConvertToCyclonedxJson(reader, &writer)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, writer.Bytes())
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, writer.Bytes())
			}
		})
	}
}
