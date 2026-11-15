package testreaddirorder

import "github.com/reproducible-containers/diffoci/golden/types"

var Tests = types.GoldenTests{
	Name: "test_readdir_order",
	Images: [2]types.Image{
		{Layers: [][]types.Entry{{
			types.Dir("etc/"), types.File("etc/group", "root:x:0:\n"), types.File("etc/hostname", "foo\n"), types.File("etc/passwd", "root:x:0:0::/root:/bin/sh\n"),
		}}},
		{Layers: [][]types.Entry{{
			types.Dir("etc/"), types.File("etc/passwd", "root:x:0:0::/root:/bin/sh\n"), types.File("etc/hostname", "foo\n"), types.File("etc/group", "root:x:0:\n"),
		}}},
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--ignore-file-order"}, Output: "ignore_file_order"},
	},
}
