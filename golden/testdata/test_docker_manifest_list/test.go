package testdockermanifestlist

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/golden/types"
)

func platformImage(arch string, docker bool) types.Image {
	return types.Image{
		Layers: [][]types.Entry{{types.File("app", arch+"\n")}},
		Config: ocispec.Image{Platform: ocispec.Platform{OS: "linux", Architecture: arch}},
		Docker: docker,
	}
}

func index(docker bool) types.Image {
	return types.Image{
		Manifests: []types.Image{platformImage("amd64", docker), platformImage("arm64", docker)},
		Docker:    docker,
	}
}

var Tests = types.GoldenTests{
	Name:   "test_docker_manifest_list",
	Images: [2]types.Image{index(true), index(false)},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
