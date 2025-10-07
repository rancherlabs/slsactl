package imagelist

import (
	"fmt"
	"io"
	"net/http"
)

type Fetcher interface {
	Fetch(string) (io.ReadCloser, error)
}

type HttpFetcher struct {
}

func (h *HttpFetcher) Fetch(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	return resp.Body, nil
}
