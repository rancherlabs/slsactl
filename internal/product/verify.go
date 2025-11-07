package product

import (
	"fmt"
	"log/slog"
	"os"
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
		fn := fmt.Sprintf("%s_%s.json", result.Product, result.Version)
		return saveOutput(fn, result)
	}

	return nil
}

func printVerifySummary(result *imagelist.Result) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 4, ' ', 0)

	fmt.Print("\n\n ✨ VERIFICATION SUMMARY ✨ \n")
	fmt.Fprintln(w, "Image Type\tSigned images")
	fmt.Fprintln(w, "-----------\t--------------")

	s := resultSummary(result)
	for name, data := range s {
		fmt.Fprintf(w, "%s\t%d (%d)\n", name, data.signed, data.count)
	}

	return w.Flush()
}
