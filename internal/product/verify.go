package product

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

func Verify(registry, name, version string, summary bool, outputFile bool) error {
	info, err := product(name, version)
	if err != nil {
		return err
	}

	fmt.Printf("Verifying container images for %s %s:\n\n", info.description, version)

	p := imagelist.NewProcessor(registry)
	result, err := p.Verify(fmt.Sprintf(info.imagesUrl, version))
	if err != nil {
		return err
	}

	result.Product = name
	result.Version = version

	if len(info.windowsImagesUrl) > 0 {
		r2, err := p.Verify(fmt.Sprintf(info.windowsImagesUrl, version))
		if err == nil {
			result.Entries = append(result.Entries, r2.Entries...)
		} else {
			slog.Error("failed to process windows images", "error", err)
		}
	}

	if summary {
		err = printVerifySummary(result)
		if err != nil {
			return fmt.Errorf("failed to print summary: %w", err)
		}
	}

	if outputFile {
		return savePrintOutput(result)
	}

	return nil
}

func printVerifySummary(result *imagelist.Result) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 4, ' ', 0)

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

	fmt.Print("\n\n ✨ VERIFICATION SUMMARY ✨ \n")
	fmt.Fprintln(w, "Image Type\tSigned images")
	fmt.Fprintln(w, "-----------\t--------------")

	for name, data := range s {
		fmt.Fprintf(w, "%s\t%d (%d)\n", name, data.signed, data.count)
	}

	return w.Flush()
}

func savePrintOutput(result *imagelist.Result) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("fail to marshal JSON: %w", err)
	}

	fn := fmt.Sprintf("%s_%s", result.Product, result.Version)
	err = os.WriteFile(fn, data, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	} else {
		fmt.Printf("\nreport saved as %q\n", fn)
	}

	return nil
}
