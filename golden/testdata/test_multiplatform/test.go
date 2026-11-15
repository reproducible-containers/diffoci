package testmultiplatform

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/golden/types"
)

func build(arm64Binary string) types.Image {
	return types.Image{Manifests: []types.Image{
		{
			Layers: [][]types.Entry{{types.File("app", "amd64\n")}},
			Config: ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "amd64"}},
		},
		{
			Layers: [][]types.Entry{{types.File("app", arm64Binary)}},
			Config: ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: "arm64"}},
		},
	}}
}

var Tests = types.GoldenTests{
	Name:   "test_multiplatform",
	Images: [2]types.Image{build("arm64\n"), build("arm64, rebuilt\n")},
	Tests: []types.GoldenTest{
		{Args: []string{"--platform=linux/amd64"}, Output: "amd64"},
		{Args: []string{"--platform=linux/amd64", "--verbose"}, Output: "amd64_verbose"},
		{Args: []string{"--platform=linux/amd64", "--semantic"}, Output: "amd64_semantic"},
		{Args: []string{"--platform=linux/arm64"}, Output: "arm64"},
		{Args: []string{"--platform=linux/amd64,linux/arm64"}, Output: "both"},
	},
}
