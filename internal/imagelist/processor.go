package imagelist

import (
	"github.com/rancherlabs/slsactl/pkg/verify"
)

type ImageProcessor interface {
	Process(string) Entry
}

type imageVerifier struct {
	registry string
}

func (i *imageVerifier) Process(img string) Entry {
	entry := Entry{
		Image: img,
	}

	entry.Error = verify.Verify(i.registry + img)
	entry.Signed = (entry.Error == nil)

	return entry
}
