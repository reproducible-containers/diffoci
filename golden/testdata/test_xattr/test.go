package testxattr

import "github.com/reproducible-containers/diffoci/golden/types"

func download(pax map[string]string) types.Image {
	f := types.File("app.tar.gz", "app\n")
	f.Header.PAXRecords = pax
	return types.Image{Layers: [][]types.Entry{{f}}}
}

var Tests = types.GoldenTests{
	Name: "test_xattr",
	Images: [2]types.Image{
		download(nil),
		download(map[string]string{"SCHILY.xattr.user.xdg.origin.url": "https://example.com/app.tar.gz"}),
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
