package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	downloadf = `usage:
    %[1]s download provenance <IMAGE>
    %[1]s download sbom <IMAGE>
`
	provenanceValue = "provenance"
	sbomValue       = "sbom"
)

func downloadCmd(args []string) error {
	f := flag.NewFlagSet("", flag.ContinueOnError)
	err := f.Parse(args)
	if err != nil {
		return err
	}

	if len(f.Args()) < 2 {
		showDownloadUsage()
	}

	var format string
	var platform string
	img := f.Arg(f.NArg() - 1)
	if f.Arg(0) == provenanceValue {
		f.StringVar(&format, "format", "slsav0.2", "The format for the Provenance output. Supported values are slsav0.2 (default) and slsav1.")
		f.StringVar(&platform, "platform", "linux/amd64", "The target platform for the container image. Most supported platforms are linux/amd64 and linux/arm64.")

		err := f.Parse(args[1:])
		if err != nil {
			return err
		}

		return provenanceCmd(img, format, platform)
	}

	if f.Arg(0) == sbomValue {
		f.StringVar(&format, "format", "spdxjson", "The format for the SBOM output. Supported values are spdxjson (default) and cyclonedxjson.")
		f.StringVar(&platform, "platform", "linux/amd64", "The target platform for the container image. Most supported platforms are linux/amd64 and linux/arm64.")

		err := f.Parse(args[1:])
		if err != nil {
			return err
		}

		return sbomCmd(img, format, platform)
	}

	showDownloadUsage()
	return nil
}

func showDownloadUsage() {
	fmt.Printf(downloadf, exeName())
	os.Exit(1)
}

func writeContent(img, format string, w io.Writer) error {
	ref, err := name.ParseReference(img)
	if err != nil {
		return fmt.Errorf("failed to parse image reference: %w", err)
	}

	desc, err := remote.Get(ref)
	if err != nil {
		return fmt.Errorf("failed to fetch image descriptor: %w", err)
	}

	var result any
	if desc.MediaType.IsIndex() {
		idx, err := desc.ImageIndex()
		if err != nil {
			return fmt.Errorf("failed to get image index: %w", err)
		}

		result, err = extractFromIndex(idx, format)
		if err != nil {
			return fmt.Errorf("failed to extract from index: %w", err)
		}
	} else {
		img, err := desc.Image()
		if err != nil {
			return fmt.Errorf("failed to get image: %w", err)
		}

		result, err = extractFromImage(img, format)
		if err != nil {
			return fmt.Errorf("failed to extract from image: %w", err)
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	_, err = w.Write(data)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w)
	return err
}

//nolint:gocognit
func extractFromIndex(idx v1.ImageIndex, attestationType string) (any, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get index manifest: %w", err)
	}

	result := make(map[string]any)
	for _, desc := range manifest.Manifests {
		if isAttestationManifest(desc) {
			attestation, platform, err := extractAttestation(idx, desc, attestationType)
			if err != nil {
				continue
			}
			if attestation != nil && platform != "" {
				result[platform] = attestation
			}
		}
	}

	// Check for single attestation without platform.
	for _, desc := range manifest.Manifests {
		if isAttestationManifest(desc) {
			attestation, _, err := extractAttestation(idx, desc, attestationType)
			if err == nil && attestation != nil {
				if singleAttest, ok := attestation.(map[string]any); ok {
					if slsa, ok := singleAttest["SLSA"]; ok && attestationType == provenanceValue {
						result["SLSA"] = slsa
					} else if spdx, ok := singleAttest["SPDX"]; ok && attestationType == sbomValue {
						result["SPDX"] = spdx
					}
				}
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no %s attestations found", attestationType)
	}

	return result, nil
}

// extractFromImage extracts provenance or SBOM from a single-platform image.
func extractFromImage(img v1.Image, format string) (any, error) {
	config, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to get image config: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// BuildKit may store attestation references in annotations.
	if manifest.Annotations != nil {
		for key, value := range manifest.Annotations {
			if strings.Contains(key, provenanceValue) && format == provenanceValue {
				var result any
				err := json.Unmarshal([]byte(value), &result)
				if err == nil {
					return result, nil
				}
			}
			if strings.Contains(key, sbomValue) && format == sbomValue {
				var result any
				err := json.Unmarshal([]byte(value), &result)
				if err == nil {
					return result, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no attestation data found in image (config: %+v)", config.Config.Labels)
}

func isAttestationManifest(desc v1.Descriptor) bool {
	if desc.MediaType == "application/vnd.oci.image.manifest.v1+json" ||
		desc.MediaType == "application/vnd.docker.distribution.manifest.v2+json" {
		if desc.Annotations != nil {
			if refType, ok := desc.Annotations["vnd.docker.reference.type"]; ok {
				return refType == "attestation-manifest"
			}
		}
	}
	return false
}

func extractAttestation(idx v1.ImageIndex, desc v1.Descriptor, attestationType string) (any, string, error) {
	img, err := idx.Image(desc.Digest)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get attestation image: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get layers: %w", err)
	}

	if len(layers) == 0 {
		return nil, "", errors.New("no layers in attestation manifest")
	}

	// The "vnd.docker.reference.digest" annotation contains the digest of the attested manifest.
	platform := ""
	if desc.Annotations != nil {
		if refDigest, ok := desc.Annotations["vnd.docker.reference.digest"]; ok {
			var err error
			platform, err = getDigestsPlatform(idx, refDigest)
			if err != nil && desc.Platform != nil {
				platform = fmt.Sprintf("%s/%s", desc.Platform.OS, desc.Platform.Architecture)
			}
		}
	}

	if platform == "" && desc.Platform != nil {
		platform = fmt.Sprintf("%s/%s", desc.Platform.OS, desc.Platform.Architecture)
	}

	for _, layer := range layers {
		layerReader, err := layer.Uncompressed()
		if err != nil {
			continue
		}

		var attestation map[string]any
		err = json.NewDecoder(layerReader).Decode(&attestation)
		if err != nil {
			layerReader.Close()
			continue
		}
		layerReader.Close()

		predicateType, ok := attestation["predicateType"].(string)
		if !ok {
			continue
		}

		var result any
		if attestationType == provenanceValue && isProvenanceType(predicateType) {
			if predicate, ok := attestation["predicate"]; ok {
				result = map[string]any{"SLSA": predicate}
				return result, platform, nil
			}
		} else if attestationType == sbomValue && isSBOMType(predicateType) {
			if predicate, ok := attestation["predicate"]; ok {
				result = map[string]any{"SPDX": predicate}
				return result, platform, nil
			}
		}
	}

	return nil, "", fmt.Errorf("no matching %s attestation found in layers", attestationType)
}

func getDigestsPlatform(idx v1.ImageIndex, digestStr string) (string, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return "", fmt.Errorf("failed to get index manifest: %w", err)
	}

	digest, err := v1.NewHash(digestStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse digest: %w", err)
	}

	for _, desc := range manifest.Manifests {
		if desc.Digest == digest && desc.Platform != nil {
			return fmt.Sprintf("%s/%s", desc.Platform.OS, desc.Platform.Architecture), nil
		}
	}

	return "", fmt.Errorf("manifest not found for digest: %s", digestStr)
}

func isProvenanceType(predicateType string) bool {
	return strings.Contains(predicateType, "slsa.dev/provenance") ||
		strings.Contains(predicateType, "slsaprovenance")
}

func isSBOMType(predicateType string) bool {
	return strings.Contains(predicateType, "spdx") ||
		strings.Contains(predicateType, "cyclonedx") ||
		strings.Contains(predicateType, sbomValue)
}
