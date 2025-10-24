package imagelist

import (
	"os"
	"sync"

	"github.com/rancherlabs/slsactl/pkg/verify"
)

type ImageProcessor interface {
	Process(img string) Entry
}

type imageVerifier struct {
	m sync.Mutex
}

func (i *imageVerifier) Process(img string) Entry {
	entry := Entry{
		Image: img,
	}

	// Reset stdout/stderr to avoid verbose output from cosign.
	i.m.Lock()
	stdout := os.Stdout
	stderr := os.Stderr
	os.Stdout = nil
	os.Stderr = nil

	entry.Error = verify.Verify(img)
	entry.Signed = (entry.Error == nil)

	os.Stdout = stdout
	os.Stderr = stderr
	i.m.Unlock()

	return entry
}
