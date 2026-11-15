package testsubsecondmtime

import (
	"time"

	"github.com/reproducible-containers/diffoci/golden/types"
)

func app(mtime time.Time) types.Image {
	f := types.File("app", "app\n")
	f.Header.ModTime = mtime
	return types.Image{Layers: [][]types.Entry{{f}}}
}

var Tests = types.GoldenTests{
	Name: "test_subsecond_mtime",
	Images: [2]types.Image{
		app(types.Epoch),
		app(types.Epoch.Add(123456789 * time.Nanosecond)),
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--ignore-file-timestamps"}, Output: "ignore_file_timestamps"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
