package sbom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/anchore/syft/syft"
	"github.com/anchore/syft/syft/cataloging/pkgcataloging"
	"github.com/anchore/syft/syft/format"
	"github.com/anchore/syft/syft/format/cyclonedxjson"
	"github.com/anchore/syft/syft/format/spdxjson"
	"github.com/anchore/syft/syft/sbom"
)

type BuildKitSBOM struct {
	LinuxAmd64 *archSBOM `json:"linux/amd64,omitempty"`
	LinuxArm64 *archSBOM `json:"linux/arm64,omitempty"`
}

type archSBOM struct {
	SPDX json.RawMessage `json:"SPDX,omitempty"`
}

func Generate(img, outformat string, writer io.Writer) error {
	defer cleanup()

	src, err := syft.GetSource(context.Background(), img, nil)
	if err != nil {
		return fmt.Errorf("failed to get source: %w", err)
	}

	cfg := syft.DefaultCreateSBOMConfig().
		WithCatalogerSelection(
			pkgcataloging.NewSelectionRequest().
				WithDefaults(
					pkgcataloging.InstalledTag,
					pkgcataloging.PackageTag,
				),
		)

	s, err := syft.CreateSBOM(context.Background(), src, cfg)
	if err != nil {
		return fmt.Errorf("failed to create SBOM: %w", err)
	}

	var enc sbom.FormatEncoder
	if outformat == "cyclonedxjson" {
		enc, _ = cyclonedxjson.NewFormatEncoderWithConfig(cyclonedxjson.DefaultEncoderConfig())
	} else if outformat == "spdxjson" {
		enc, _ = spdxjson.NewFormatEncoderWithConfig(spdxjson.DefaultEncoderConfig())
	}

	if enc == nil {
		return fmt.Errorf("invalid format %s: failed to create encoder", outformat)
	}

	data, err := format.Encode(*s, enc)
	if err != nil {
		return fmt.Errorf("failed to encode sbom: %w", err)
	}

	_, err = io.Copy(writer, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to copy sbom to writer: %w", err)
	}

	return nil
}

// During the sbom generation process, Syft uses temporary dirs in the
// format /tmp/stereoscope-* that contains both the compressed and inflated
// versions of the image. This ensures that both are removed.
func cleanup() error {
	m, err := filepath.Glob("/tmp/stereoscope-*")
	if err != nil {
		return err
	}

	for _, d := range m {
		_ = os.RemoveAll(d)
	}
	return nil
}

func ConvertToCyclonedxJson(reader io.Reader, writer io.Writer) error {
	s, _, _, err := format.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode SPDX JSON SBOM: %w", err)
	}

	enc, err := cyclonedxjson.NewFormatEncoderWithConfig(cyclonedxjson.DefaultEncoderConfig())
	if err != nil {
		return fmt.Errorf("failed to create cyclonedxjson encoder: %w", err)
	}

	data, err := format.Encode(*s, enc)
	if err != nil {
		return fmt.Errorf("failed to encode sbom: %w", err)
	}

	_, err = io.Copy(writer, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to copy sbom to writer: %w", err)
	}

	return nil
}
