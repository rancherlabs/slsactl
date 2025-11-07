package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

var (
	ErrInvalidVersion = errors.New("invalid version")

	versionRegex = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(\-(?:alpha|beta|rc)\d+)?$`)

	productMapping = map[string]productInfo{
		"rancher-prime": {
			description:      "SUSE Rancher Prime",
			imagesUrl:        "https://github.com/rancher/rancher/releases/download/%s/rancher-images.txt",
			windowsImagesUrl: "https://github.com/rancher/rancher/releases/download/%s/rancher-windows-images.txt",
		},
		"storage": {
			description: "SUSE Storage",
			imagesUrl:   "https://github.com/longhorn/longhorn/releases/download/%s/longhorn-images.txt",
		},
		"virtualization": {
			description: "SUSE Virtualization",
			imagesUrl:   "https://github.com/harvester/harvester/releases/download/%s/harvester-images-list-amd64.txt",
		},
	}
)

type productInfo struct {
	description      string
	imagesUrl        string
	windowsImagesUrl string
}

type summary struct {
	count  int
	signed int
	errors int
}

func product(name, version string) (*productInfo, error) {
	if !versionRegex.MatchString(version) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidVersion, version)
	}

	info, found := productMapping[name]
	if !found {
		var names []string
		for name := range productMapping {
			names = append(names, name)
		}
		products := strings.Join(names, ", ")
		return nil, fmt.Errorf("product %q not found: options are %s", name, products)
	}

	return &info, nil
}

func saveOutput(fn string, result *imagelist.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("fail to marshal JSON: %w", err)
	}

	err = os.WriteFile(fn, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	} else {
		fmt.Printf("\nreport saved as %q\n", fn)
	}

	return nil
}

func resultSummary(result *imagelist.Result) map[string]*summary {
	s := map[string]*summary{}
	for _, entry := range result.Entries {
		imgType := "rancher"
		if strings.Contains(entry.Image, "rancher/mirrored") {
			imgType = "third-party"
		}

		if _, ok := s[imgType]; !ok {
			s[imgType] = &summary{}
		}

		s[imgType].count++
		if entry.Signed {
			s[imgType].signed++
		}
		if entry.Error != nil {
			s[imgType].errors++
		}
	}
	return s
}
