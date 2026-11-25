package testswappedinputs

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/golden/types"
)

var amd64 = types.Image{
	Layers: [][]types.Entry{{types.File("app", "amd64\n")}},
	Config: ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "amd64"}},
}

var arm64 = types.Image{
	Layers: [][]types.Entry{{types.File("app", "arm64\n")}},
	Config: ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "arm64"}},
}

var Tests = types.GoldenTests{
	Name:   "test_swapped_inputs",
	Images: [2]types.Image{amd64, {Manifests: []types.Image{amd64, arm64}}},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
	},
}
