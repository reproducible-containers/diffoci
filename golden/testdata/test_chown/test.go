package testchown

import "github.com/reproducible-containers/diffoci/golden/types"

var copied = types.Image{Layers: [][]types.Entry{{types.File("app", "#!/bin/sh\n")}}}

var chowned = func() types.Image {
	app := types.File("app", "#!/bin/sh\n")
	app.Header.Uid = 1000
	app.Header.Gid = 1000
	return types.Image{Layers: [][]types.Entry{{app}}}
}()

var Tests = types.GoldenTests{
	Name:   "test_chown",
	Images: [2]types.Image{copied, chowned},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
