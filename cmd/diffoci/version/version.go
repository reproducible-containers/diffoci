package version

import (
	"runtime/debug"
	"strconv"
)

// Version can be fulfilled on compilation time: -ldflags="-X github.com/reproducible-containers/diffoci/cmd/diffoci/version.Version=v0.1.2"
var Version string

func GetVersion() string {
	if Version != "" {
		return Version
	}
	const unknown = "(unknown)"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return unknown
	}

	/*
	 * go install example.com/cmd/foo@vX.Y.Z: bi.Main.Version="vX.Y.Z",                               vcs.revision is unset
	 * go install example.com/cmd/foo@latest: bi.Main.Version="vX.Y.Z",                               vcs.revision is unset
	 * go install example.com/cmd/foo@master: bi.Main.Version="vX.Y.Z-N.yyyyMMddhhmmss-gggggggggggg", vcs.revision is unset
	 * go install ./cmd/foo:                  bi.Main.Version="(devel)", vcs.revision="gggggggggggggggggggggggggggggggggggggggg"
	 *                                        vcs.time="yyyy-MM-ddThh:mm:ssZ", vcs.modified=("false"|"true")
	 */
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	var (
		vcsRevision string
		vcsModified bool
	)
	for _, f := range bi.Settings {
		switch f.Key {
		case "vcs.revision":
			vcsRevision = f.Value
		case "vcs.modified":
			vcsModified, _ = strconv.ParseBool(f.Value)
		}
	}
	if vcsRevision == "" {
		return unknown
	}
	v := vcsRevision
	if vcsModified {
		v += ".m"
	}
	return v
}
