package testsquash

import "github.com/reproducible-containers/diffoci/golden/types"

var Tests = types.GoldenTests{
	Name: "test_squash",
	Images: [2]types.Image{
		{Layers: [][]types.Entry{
			{types.File("app", "#!/bin/sh\n")},
			{types.File("config.json", "{}\n")},
		}},
		{Layers: [][]types.Entry{
			{types.File("app", "#!/bin/sh\n"), types.File("config.json", "{}\n")},
		}},
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
