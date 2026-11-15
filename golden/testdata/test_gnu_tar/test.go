package testgnutar

import (
	"archive/tar"

	"github.com/reproducible-containers/diffoci/golden/types"
)

var gnu = func() types.Image {
	app := types.File("app", "#!/bin/sh\n")
	app.Header.Format = tar.FormatGNU
	return types.Image{Layers: [][]types.Entry{{app}}}
}()

var Tests = types.GoldenTests{
	Name: "test_gnu_tar",
	Images: [2]types.Image{
		{Layers: [][]types.Entry{{types.File("app", "#!/bin/sh\n")}}},
		gnu,
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--ignore-tar-format"}, Output: "ignore_tar_format"},
	},
}
