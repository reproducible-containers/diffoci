package testlayertoolarge

import (
	"strings"

	"github.com/reproducible-containers/diffoci/golden/types"
)

var image = types.Image{Layers: [][]types.Entry{{
	types.File("model.bin", strings.Repeat("\x00", 5<<20)),
}}}

var Tests = types.GoldenTests{
	Name:   "test_layer_too_large",
	Images: [2]types.Image{{Manifests: []types.Image{image}}, image},
	Tests: []types.GoldenTest{
		{Args: []string{"--max-scale=0.001"}, Output: "max_scale"},
	},
}
