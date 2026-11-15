package testbuildkitvslegacy

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/golden/types"
)

func build(mode int64, comment string) types.Image {
	app := types.File("app", "#!/bin/sh\n")
	app.Header.Mode = mode
	return types.Image{
		Layers: [][]types.Entry{{app}},
		Config: ocispec.Image{
			History: []ocispec.History{{CreatedBy: "COPY app /app # buildkit", Comment: comment}},
		},
	}
}

var Tests = types.GoldenTests{
	Name: "test_buildkit_vs_legacy",
	Images: [2]types.Image{
		build(0o100755, "buildkit.dockerfile.v0"),
		build(0o755, ""),
	},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--ignore-file-mode-redundant-bits"}, Output: "ignore_file_mode_redundant_bits"},
		{Args: []string{"--ignore-file-mode-redundant-bits", "--verbose"}, Output: "ignore_file_mode_redundant_bits_verbose"},
		{Args: []string{"--ignore-history"}, Output: "ignore_history"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
