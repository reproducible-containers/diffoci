package testbusyboxlinks

import (
	"archive/tar"

	"github.com/reproducible-containers/diffoci/golden/types"
)

func link(typeflag byte, mode int64, name, target string) types.Entry {
	return types.Entry{Header: tar.Header{Typeflag: typeflag, Name: name, Linkname: target, Mode: mode, ModTime: types.Epoch, Format: tar.FormatPAX}}
}

func install(typeflag byte, mode int64, target string) types.Image {
	return types.Image{Layers: [][]types.Entry{{
		types.Dir("bin/"),
		types.File("bin/busybox", "busybox\n"),
		link(typeflag, mode, "bin/sh", target),
	}}}
}

var Tests = types.GoldenTests{
	Name: "test_busybox_links",
	Images: [2]types.Image{
		install(tar.TypeLink, 0o644, "bin/busybox"),
		install(tar.TypeSymlink, 0o777, "busybox"),
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--semantic", "--report-dir=$REPORT_DIR"}, Output: "report_dir"},
	},
}
