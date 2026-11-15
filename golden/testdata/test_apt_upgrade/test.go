package testaptupgrade

import "github.com/reproducible-containers/diffoci/golden/types"

func upgrade(date string) types.Image {
	return types.Image{Layers: [][]types.Entry{{
		types.Dir("var/lib/dpkg/"),
		types.File("var/lib/dpkg/status", "Package: curl\nStatus: install ok installed\n"),
		types.File("var/lib/dpkg/status-old", "Package: curl\nStatus: install ok installed\n# "+date+"\n"),
		types.Dir("var/log/"),
		types.File("var/log/dpkg.log", date+" upgrade curl:amd64 8.5.0-2 8.5.0-3\n"),
	}}}
}

var Tests = types.GoldenTests{
	Name:   "test_apt_upgrade",
	Images: [2]types.Image{upgrade("2024-01-01 10:00:00"), upgrade("2024-01-02 11:30:00")},
	Tests: []types.GoldenTest{
		{Args: []string{"--semantic"}, Output: "semantic"},
	},
}
