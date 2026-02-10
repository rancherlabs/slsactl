package product

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

func Download(registry, name, version string) error {
	info, err := product(name, version)
	if err != nil {
		return err
	}

	outputDir := fmt.Sprintf("%s-%s", name, version)
	err = os.MkdirAll(outputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Downloading SBOM and Provenance for %s %s:\n\n", info.description, version)
	fmt.Printf("Output directory: %s\n\n", outputDir)

	p := imagelist.NewProcessor(registry)
	result, err := p.Download(fmt.Sprintf(info.imagesUrl, version), outputDir)
	if err != nil {
		return err
	}

	result.Product = name
	result.Version = version

	if len(info.windowsImagesUrl) > 0 {
		r2, err := p.Download(fmt.Sprintf(info.windowsImagesUrl, version), outputDir)
		if err == nil {
			result.Entries = append(result.Entries, r2.Entries...)
		} else {
			slog.Error("failed to process windows images", "error", err)
		}
	}

	err = printDownloadSummary(result)
	if err != nil {
		return fmt.Errorf("failed to print summary: %w", err)
	}

	fn := fmt.Sprintf("%s/%s_%s_download.json", outputDir, result.Product, result.Version)
	return saveOutput(fn, result)
}

func printDownloadSummary(result *imagelist.Result) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 4, ' ', 0)

	fmt.Print("\n\n ✨ DOWNLOAD SUMMARY ✨ \n")
	fmt.Fprintln(w, "Type\tCount\tWith SBOM\tWith Provenance")
	fmt.Fprintln(w, "----\t-----\t---------\t---------------")

	s := downloadSummary(result)
	for name, data := range s {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", name, data.count, data.sbom, data.prov)
	}

	return w.Flush()
}

type downloadStats struct {
	count int
	sbom  int
	prov  int
}

func downloadSummary(result *imagelist.Result) map[string]*downloadStats {
	s := map[string]*downloadStats{}
	for _, entry := range result.Entries {
		imgType := "rancher"
		if strings.Contains(entry.Image, "rancher/mirrored") {
			imgType = "third-party"
		}

		if _, ok := s[imgType]; !ok {
			s[imgType] = &downloadStats{}
		}

		s[imgType].count++
		if entry.SBOMFile != "" {
			s[imgType].sbom++
		}
		if entry.ProvFile != "" {
			s[imgType].prov++
		}
	}
	return s
}
