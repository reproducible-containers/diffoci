package testrootentry

import "github.com/reproducible-containers/diffoci/golden/types"

func build(rootMode int64) types.Image {
	root := types.Dir("./")
	root.Header.Mode = rootMode
	return types.Image{Layers: [][]types.Entry{{root, types.File("./app", "#!/bin/sh\n")}}}
}

var Tests = types.GoldenTests{
	Name:   "test_root_entry",
	Images: [2]types.Image{build(0o755), build(0o700)},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--treat-canonical-paths-equal"}, Output: "treat_canonical_paths_equal"},
	},
}
