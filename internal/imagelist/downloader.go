package imagelist

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	provenanceType = "provenance"
	sbomType       = "sbom"
)

type imageDownloader struct {
	m sync.Mutex
}

func (d *imageDownloader) Download(img, outputDir string) Entry {
	d.m.Lock()
	defer d.m.Unlock()

	entry := Entry{
		Image: img,
	}

	ref, err := name.ParseReference(img)
	if err != nil {
		entry.Error = fmt.Errorf("failed to parse image reference: %w", err)
		return entry
	}

	desc, err := remote.Get(ref)
	if err != nil {
		entry.Error = fmt.Errorf("failed to fetch image descriptor: %w", err)
		return entry
	}

	imgName := sanitizeImageName(img)

	var sbomData, provData any
	if desc.MediaType.IsIndex() {
		idx, err := desc.ImageIndex()
		if err != nil {
			entry.Error = fmt.Errorf("failed to get image index: %w", err)
			return entry
		}

		sbomData, _ = extractFromIndex(idx, sbomType)
		provData, _ = extractFromIndex(idx, provenanceType)
	} else {
		image, err := desc.Image()
		if err != nil {
			entry.Error = fmt.Errorf("failed to get image: %w", err)
			return entry
		}

		sbomData, _ = extractFromImage(image, sbomType)
		provData, _ = extractFromImage(image, provenanceType)
	}

	if sbomData != nil {
		sbomFile := filepath.Join(outputDir, imgName+"_sbom.json")
		err := saveJSON(sbomFile, sbomData)
		if err == nil {
			entry.SBOMFile = sbomFile
		}
	}

	if provData != nil {
		provFile := filepath.Join(outputDir, imgName+"_provenance.json")
		err := saveJSON(provFile, provData)
		if err == nil {
			entry.ProvFile = provFile
		}
	}

	if entry.SBOMFile == "" && entry.ProvFile == "" {
		entry.Error = errors.New("no attestations found")
	}

	return entry
}

func sanitizeImageName(img string) string {
	var result string

	// Attempt to drop the registry name first by parsing the reference.
	// If that doesn't work, then a simple cut which assumes the first
	// part is the registry name.
	ref, err := name.ParseReference(img, name.WeakValidation)
	if err != nil {
		before, after, ok := strings.Cut(img, "/")
		if ok && strings.Contains(before, ".") {
			result = after
		} else {
			result = img
		}
	} else {
		result = ref.Context().RepositoryStr() + "_" + ref.Identifier()
	}

	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, ":", "_")
	result = strings.ReplaceAll(result, "@", "_")
	result = strings.ReplaceAll(result, "..", "")

	return result
}

func saveJSON(filename string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonData, 0o600)
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
			if err == nil {
				if attestation != nil && platform != "" {
					result[platform] = attestation
				}
			}
		}
	}

	// Check for single attestation without platform
	for _, desc := range manifest.Manifests {
		if isAttestationManifest(desc) {
			attestation, _, err := extractAttestation(idx, desc, attestationType)
			if err == nil && attestation != nil {
				if singleAttest, ok := attestation.(map[string]any); ok {
					if slsa, ok := singleAttest["SLSA"]; ok && attestationType == provenanceType {
						result["SLSA"] = slsa
					} else if spdx, ok := singleAttest["SPDX"]; ok && attestationType == sbomType {
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

func extractFromImage(img v1.Image, format string) (any, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	if manifest.Annotations != nil {
		for key, value := range manifest.Annotations {
			if strings.Contains(key, provenanceType) && format == provenanceType {
				var result any
				err := json.Unmarshal([]byte(value), &result)
				if err == nil {
					return result, nil
				}
			}
			if strings.Contains(key, sbomType) && format == sbomType {
				var result any
				err := json.Unmarshal([]byte(value), &result)
				if err == nil {
					return result, nil
				}
			}
		}
	}

	return nil, errors.New("no attestation data found in image")
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

		if attestationType == provenanceType && isProvenanceType(predicateType) {
			if predicate, ok := attestation["predicate"]; ok {
				return map[string]any{"SLSA": predicate}, platform, nil
			}
		} else if attestationType == sbomType && isSBOMType(predicateType) {
			if predicate, ok := attestation["predicate"]; ok {
				return map[string]any{"SPDX": predicate}, platform, nil
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
		strings.Contains(predicateType, sbomType)
}
