package testfilecapabilities

import (
	"encoding/binary"

	"github.com/reproducible-containers/diffoci/golden/types"
)

// from linux/capability.h
const (
	vfsCapRevision2      = 0x02000000
	vfsCapFlagsEffective = 0x000001
	capNetRaw            = 13
)

// struct vfs_cap_data revision 2, as stored in the security.capability xattr
func fileCap(permitted uint64, effective bool) string {
	magic := uint32(vfsCapRevision2)
	if effective {
		magic |= vfsCapFlagsEffective
	}
	b := binary.LittleEndian.AppendUint32(nil, magic)
	b = binary.LittleEndian.AppendUint32(b, uint32(permitted))
	b = binary.LittleEndian.AppendUint32(b, 0)
	b = binary.LittleEndian.AppendUint32(b, uint32(permitted>>32))
	b = binary.LittleEndian.AppendUint32(b, 0)
	return string(b)
}

func ping(capability string) types.Image {
	f := types.File("usr/bin/ping", "ping\n")
	f.Header.Mode = 0o755
	f.Header.PAXRecords = map[string]string{"SCHILY.xattr.security.capability": capability}
	return types.Image{Layers: [][]types.Entry{{types.Dir("usr/bin/"), f}}}
}

var Tests = types.GoldenTests{
	Name: "test_file_capabilities",
	Images: [2]types.Image{
		ping(fileCap(1<<capNetRaw, true)),
		ping(fileCap(1<<capNetRaw, false)),
	},
	Tests: []types.GoldenTest{
		{Args: []string{"--semantic"}, Output: "semantic"},
		{Args: []string{"--semantic", "--report-dir=$REPORT_DIR"}, Output: "report_dir"},
	},
}
