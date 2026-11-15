package testfilecapabilities

import "github.com/reproducible-containers/diffoci/golden/types"

// vfs_cap_data revision 2 with cap_net_raw permitted; the lowest bit of the first byte sets it effective
const (
	capNetRawEP = "\x01\x00\x00\x02\x00\x20\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
	capNetRawP  = "\x00\x00\x00\x02\x00\x20\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
)

func ping(capability string) types.Image {
	f := types.File("usr/bin/ping", "ping\n")
	f.Header.Mode = 0o755
	f.Header.PAXRecords = map[string]string{"SCHILY.xattr.security.capability": capability}
	return types.Image{Layers: [][]types.Entry{{types.Dir("usr/bin/"), f}}}
}

var Tests = types.GoldenTests{
	Name: "test_file_capabilities",
	Images: [2]types.Image{
		ping(capNetRawEP),
		ping(capNetRawP),
	},
	Tests: []types.GoldenTest{
		{Args: []string{"--semantic"}, Output: "semantic"},
		{Args: []string{"--semantic", "--report-dir=$REPORT_DIR"}, Output: "report_dir"},
	},
}
