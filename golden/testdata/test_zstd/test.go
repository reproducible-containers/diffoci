package testzstd

import (
	"github.com/containerd/containerd/v2/pkg/archive/compression"
	"github.com/reproducible-containers/diffoci/golden/types"
)

var layers = [][]types.Entry{{
	types.Dir("etc/"),
	types.File("etc/os-release", "ID=debian\n"),
}}

var Tests = types.GoldenTests{
	Name: "test_zstd",
	Images: [2]types.Image{
		{Layers: layers, Compression: compression.Gzip},
		{Layers: layers, Compression: compression.Zstd},
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
