package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

const maxDownloadSize = 5 * (1 << 20) // 5MB

var (
	versionRegex      = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]d*)$`)
	ErrInvalidVersion = errors.New("invalid version")
)

type productInfo struct {
	description      string
	imagesUrl        string
	windowsImagesUrl string
}

var (
	productMapping = map[string]productInfo{
		"rancher-prime": {
			imagesUrl:        "https://github.com/rancher/rancher/releases/download/%s/rancher-images.txt",
			windowsImagesUrl: "https://github.com/rancher/rancher/releases/download/%s/rancher-windows-images.txt",
		},
		"storage": {
			imagesUrl: "https://github.com/longhorn/longhorn/releases/download/%s/longhorn-images.txt",
		},
		"virtualization": {
			imagesUrl: "https://github.com/harvester/harvester/releases/download/%s/harvester-images-list-amd64.txt",
		},
	}
)

func Verify(registry, name, version string, summary bool, outputFile bool) error {
	if !versionRegex.MatchString(version) {
		return fmt.Errorf("%w: %s", ErrInvalidVersion, version)
	}

	info, found := productMapping[name]
	if !found {
		var names []string
		for name := range productMapping {
			names = append(names, name)
		}
		products := strings.Join(names, ", ")
		return fmt.Errorf("product %q not found: options are %s", name, products)
	}

	p := imagelist.NewProcessor(registry)
	result, err := p.Process(fmt.Sprintf(info.imagesUrl, version))
	if err != nil {
		return err
	}

	result.Product = name
	result.Version = version

	if len(info.windowsImagesUrl) > 0 {
		r2, err := p.Process(fmt.Sprintf(info.windowsImagesUrl, version))
		if err == nil {
			result.Entries = append(result.Entries, r2.Entries...)
		} else {
			slog.Error("failed to process windows images", "error", err)
		}
	}

	if summary {
		err = printSummary(result)
		if err != nil {
			return fmt.Errorf("failed to print summary: %w", err)
		}
	}

	if outputFile {
		return saveOutput(result)
	}

	return nil
}

type summary struct {
	count  int
	signed int
	errors int
}

func printSummary(result *imagelist.Result) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 4, ' ', 0)

	s := map[string]*summary{}
	for _, entry := range result.Entries {
		imgType := "official"
		if strings.HasPrefix(entry.Image, "rancher/mirrored") {
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

	fmt.Fprintln(w, "Image Type\tCount\tSigned\tErrors")
	fmt.Fprintln(w, "-----------\t-----\t------\t------")

	for name, data := range s {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", name, data.count, data.signed, data.errors)
	}

	return w.Flush()
}

func saveOutput(result *imagelist.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("fail to marshal JSON: %w", err)
	}

	fn := fmt.Sprintf("%s_%s", result.Product, result.Version)
	err = os.WriteFile(fn, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
