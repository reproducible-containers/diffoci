package testrebuild

import (
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/golden/types"
)

func build(t time.Time) types.Image {
	hostname := types.File("etc/hostname", "foo\n")
	hostname.Header.ModTime = t
	return types.Image{
		Layers: [][]types.Entry{{types.Dir("etc/"), hostname}},
		Config: ocispec.Image{
			Created: &t,
			History: []ocispec.History{{Created: &t, CreatedBy: "COPY hostname /etc/hostname # buildkit"}},
		},
	}
}

var Tests = types.GoldenTests{
	Name:   "test_rebuild",
	Images: [2]types.Image{build(types.Epoch), build(types.Epoch.Add(time.Hour))},
	Tests: []types.GoldenTest{
		{Args: nil, Output: "default"},
		{Args: []string{"--ignore-file-timestamps"}, Output: "ignore_file_timestamps"},
		{Args: []string{"--ignore-file-timestamps", "--verbose"}, Output: "ignore_file_timestamps_verbose"},
		{Args: []string{"--ignore-image-timestamps"}, Output: "ignore_image_timestamps"},
		{Args: []string{"--ignore-timestamps"}, Output: "ignore_timestamps"},
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
