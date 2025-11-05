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

type Processor struct {
	ip       ImageProcessor
	fetcher  Fetcher
	registry string
}

func NewProcessor(registry string) *Processor {
	if !strings.HasSuffix(registry, "/") {
		registry = registry + "/"
	}

	return &Processor{
		registry: registry,
		ip:       new(imageVerifier),
		fetcher:  new(HttpFetcher),
	}
}

func (p *Processor) Verify(url string) (*Result, error) {
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

	s = spinner.New("Verify images")
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

		entry := p.ip.Verify(image)

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

type Result struct {
	Product string  `json:"product,omitempty"`
	Version string  `json:"version,omitempty"`
	Entries []Entry `json:"entries,omitempty"`
}

type Entry struct {
	Image  string `json:"image,omitempty"`
	Error  error  `json:"error,omitempty"`
	Signed bool   `json:"signed,omitempty"`
}
