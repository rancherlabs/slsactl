package product

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrInvalidVersion = errors.New("invalid version")

	versionRegex = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]d*)(\-(?:alpha|beta|rc)\d+)?$`)

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
