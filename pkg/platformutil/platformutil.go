package platformutil

import (
	"fmt"

	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func FormatSlice(ps []ocispec.Platform) string {
	ss := make([]string, len(ps))
	for i := range ps {
		ss[i] = platforms.Format(ps[i])
	}
	return fmt.Sprintf("%v", ss)
}
