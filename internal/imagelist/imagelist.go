package imagelist

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancherlabs/slsactl/internal/spinner"
)

var (
	ErrNoImagesFound    = errors.New("no images found")
	ErrURLCannotBeEmpty = errors.New("URL cannot be empty")
	ErrCannotFetchURL   = errors.New("cannot fetch URL")
)

const maxProcessingSizeInBytes = 5 * (1 << 20) // 5MB

type ImageVerifier interface {
	Verify(img string) Entry
}

type ImageCopier interface {
	Copy(img, targetRegistry string) Entry
}

type ImageDownloader interface {
	Download(img, outputDir string) Entry
}

type Result struct {
	Product string  `json:"product,omitempty"`
	Version string  `json:"version,omitempty"`
	Entries []Entry `json:"entries,omitempty"`
}

type Entry struct {
	Image    string `json:"image,omitempty"`
	Error    error  `json:"error,omitempty"`
	Signed   bool   `json:"signed,omitempty"`
	SBOMFile string `json:"sbomFile,omitempty"`
	ProvFile string `json:"provFile,omitempty"`
}

type Processor struct {
	ip         ImageVerifier
	copier     ImageCopier
	downloader ImageDownloader
	fetcher    Fetcher
	registry   string
}

func NewProcessor(registry string) *Processor {
	if !strings.HasSuffix(registry, "/") {
		registry = registry + "/"
	}

	copier := &imageCopier{
		mirroredOnly: true,
	}

	return &Processor{
		registry:   registry,
		ip:         new(imageVerifier),
		fetcher:    new(HttpFetcher),
		copier:     copier,
		downloader: new(imageDownloader),
	}
}

func (p *Processor) Verify(url string) (*Result, error) {
	return p.process(url, "Verify images", "", func(img, _ string) Entry {
		return p.ip.Verify(img)
	})
}

func (p *Processor) Copy(url, dstRegistry string) (*Result, error) {
	return p.process(url, "Copy images", dstRegistry, func(img, dstRegistry string) Entry {
		return p.copier.Copy(img, dstRegistry)
	})
}

func (p *Processor) Download(url, outputDir string) (*Result, error) {
	return p.process(url, "Download attestations", outputDir, func(img, outputDir string) Entry {
		return p.downloader.Download(img, outputDir)
	})
}

func (p *Processor) process(url, status, dstRegistry string, action func(string, string) Entry) (*Result, error) {
	url = strings.TrimSpace(url)
	if len(url) == 0 {
		return nil, ErrURLCannotBeEmpty
	}

	s := spinner.New("Fetch product manifest")
	s.Start()
	s.UpdateStatus(url)

	r, err := p.fetcher.Fetch(url)
	s.Stop(err == nil)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrCannotFetchURL, url, err)
	}

	defer func() {
		err := r.Close()
		if err != nil {
			slog.Error("error closing fetched reader from %q: %w", url, err)
		}
	}()

	result := Result{}

	scanner := bufio.NewScanner(io.LimitReader(r, maxProcessingSizeInBytes))

	s = spinner.New(status)
	s.Start()

	for scanner.Scan() {
		image := strings.TrimSpace(scanner.Text())

		if len(image) == 0 || strings.HasPrefix(image, "#") {
			continue
		}

		image = strings.TrimPrefix(image, "docker.io/")

		ref, err := name.ParseReference(image, name.WeakValidation)
		if err != nil {
			return nil, fmt.Errorf("failed to parse image name: %w", err)
		}

		if ref.Context().Registry.Name() == "" || ref.Context().Registry.Name() == "index.docker.io" {
			image = p.registry + image
		}

		s.UpdateStatus(image)

		entry := action(image, dstRegistry)

		result.Entries = append(result.Entries, entry)
	}

	err = scanner.Err()
	s.Stop(err == nil && len(result.Entries) > 0)
	if err != nil {
		return nil, fmt.Errorf("error found scanning image list: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, ErrNoImagesFound
	}

	return &result, nil
}
