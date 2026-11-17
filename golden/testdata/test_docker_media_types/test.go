package testdockermediatypes

import "github.com/reproducible-containers/diffoci/golden/types"

var layers = [][]types.Entry{{types.File("app", "app\n")}}

var Tests = types.GoldenTests{
	Name: "test_docker_media_types",
	Images: [2]types.Image{
		{Layers: layers, Docker: true},
		{Layers: layers},
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
