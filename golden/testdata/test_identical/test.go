package testidentical

import "github.com/reproducible-containers/diffoci/golden/types"

var image = types.Image{Layers: [][]types.Entry{{types.Dir("etc/"), types.File("etc/hostname", "foo\n")}}}

var Tests = types.GoldenTests{
	Name:   "test_identical",
	Images: [2]types.Image{image, image},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
	},
}
