package testlongpath

import (
	"strings"

	"github.com/reproducible-containers/diffoci/golden/types"
)

var path = strings.Repeat("node_modules/@babel/plugin-transform-modules-commonjs/", 5) + "lib/index.js"

var Tests = types.GoldenTests{
	Name: "test_long_path",
	Images: [2]types.Image{
		{Layers: [][]types.Entry{{types.File(path, "module.exports = {};\n")}}},
		{Layers: [][]types.Entry{{types.File("./"+path, "module.exports = {};\n")}}},
	},
	Tests: []types.GoldenTest{
		{Output: "default"},
		{Args: []string{"--treat-canonical-paths-equal"}, Output: "treat_canonical_paths_equal"},
	},
}
