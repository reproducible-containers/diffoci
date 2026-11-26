package testapt

import "github.com/reproducible-containers/diffoci/golden/types"

func install(log string) types.Image {
	return types.Image{Layers: [][]types.Entry{{
		types.Dir("usr/bin/"),
		types.File("usr/bin/curl", "curl\n"),
		types.Dir("var/log/"),
		types.File("var/log/dpkg.log", log),
	}}}
}

var Tests = types.GoldenTests{
	Name: "test_apt",
	Images: [2]types.Image{
		install("2024-01-01 10:00:00 install curl:amd64 <none> 8.5.0-2\n"),
		install("2024-01-02 11:30:00 install curl:amd64 <none> 8.5.0-2\n"),
	},
	Tests: []types.GoldenTest{
		{Args: []string{"--semantic"}, Output: "semantic"},
		{Args: []string{"--semantic", "--report-dir=$REPORT_DIR"}, Output: "report_dir"},
		{Args: []string{"--semantic", "--ignore-path=/var/log/dpkg.log"}, Output: "ignore_path"},
		{Args: []string{"--semantic", "--ignore-path=/var/log"}, Output: "ignore_path_dir"},
	},
}
