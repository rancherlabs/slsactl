package imagelist

import (
	"github.com/rancherlabs/slsactl/pkg/verify"
)

type ImageProcessor interface {
	Process(img string) Entry
}

type imageVerifier struct{}

func (i *imageVerifier) Process(img string) Entry {
	entry := Entry{
		Image: img,
	}

	entry.Error = verify.Verify(img)
	entry.Signed = (entry.Error == nil)

	return entry
}
