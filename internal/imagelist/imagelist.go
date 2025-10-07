package imagelist

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var (
	ErrNoImagesFound    = errors.New("no images found")
	ErrURLCannotBeEmpty = errors.New("URL cannot be empty")
	ErrCannotFetchURL   = errors.New("cannot fetch URL")
)

const maxProcessingSizeInBytes = 5 * (1 << 20) // 5MB

func NewProcessor(registry string) *Processor {
	if !strings.HasSuffix(registry, "/") {
		registry = registry + "/"
	}

	return &Processor{
		ip:      &imageVerifier{registry: registry},
		fetcher: new(HttpFetcher),
	}
}

type Processor struct {
	ip      ImageProcessor
	fetcher Fetcher
}

func (p *Processor) Process(url string) (*Result, error) {
	url = strings.TrimSpace(url)
	if len(url) == 0 {
		return nil, ErrURLCannotBeEmpty
	}

	r, err := p.fetcher.Fetch(url)
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
	for scanner.Scan() {
		image := strings.TrimSpace(scanner.Text())

		if len(image) == 0 || strings.HasPrefix(image, "#") {
			continue
		}

		fmt.Println("processing", image)
		entry := p.ip.Process(image)
		result.Entries = append(result.Entries, entry)
	}

	if err := scanner.Err(); err != nil {
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
