package testdotprefix

import "github.com/reproducible-containers/diffoci/golden/types"

var Tests = types.GoldenTests{
	Name: "test_dot_prefix",
	Images: [2]types.Image{
		{Layers: [][]types.Entry{{types.File("app", "#!/bin/sh\n")}}},
		{Layers: [][]types.Entry{{types.File("./app", "#!/bin/sh\n")}}},
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--treat-canonical-paths-equal"}, Output: "treat_canonical_paths_equal"},
	},
}
