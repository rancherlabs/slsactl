package product

import (
	"fmt"
	"log/slog"
	"os"
	"text/tabwriter"

	"github.com/rancherlabs/slsactl/internal/imagelist"
)

func Copy(registry, name, version, targetRegistry string) error {
	info, err := product(name, version)
	if err != nil {
		return err
	}

	fmt.Printf("Copying %s %s signatures to %q:\n\n", info.description, version, targetRegistry)

	p := imagelist.NewProcessor(registry)
	result, err := p.Copy(fmt.Sprintf(info.imagesUrl, version), targetRegistry)
	if err != nil {
		return err
	}

	result.Product = name
	result.Version = version

	if len(info.windowsImagesUrl) > 0 {
		r2, err := p.Copy(fmt.Sprintf(info.windowsImagesUrl, version), targetRegistry)
		if err == nil {
			result.Entries = append(result.Entries, r2.Entries...)
		} else {
			slog.Error("failed to process windows images", "error", err)
		}
	}

	err = printCopySummary(result)
	if err != nil {
		return fmt.Errorf("failed to print summary: %w", err)
	}

	fn := fmt.Sprintf("%s_%s_copy.json", result.Product, result.Version)
	return saveOutput(fn, result)
}

func printCopySummary(result *imagelist.Result) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 12, 12, 4, ' ', 0)

	fmt.Print("\n\n ✨ COPY SUMMARY ✨ \n")
	fmt.Fprintln(w, "Image Type\tImages Count\tSignatures")
	fmt.Fprintln(w, "-----------\t------------\t------------")

	s := resultSummary(result)
	for name, data := range s {
		fmt.Fprintf(w, "%s\t%d \t%d\n", name, data.count, data.signed)
	}

	return w.Flush()
}
