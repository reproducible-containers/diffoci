package flagutil

import (
	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
)

func AddPlatformFlags(flags *pflag.FlagSet) {
	flags.Bool("all-platforms", false, "Specify all the image platforms")
	flags.StringSlice("platform", []string{platforms.DefaultString()}, "Specify the image platform")
}

func ParsePlatformFlags(flags *pflag.FlagSet) ([]ocispec.Platform, error) {
	allPlatforms, err := flags.GetBool("all-platforms")
	if err != nil {
		return nil, err
	}
	if allPlatforms {
		return nil, nil
	}
	platformSS, err := flags.GetStringSlice("platform")
	if err != nil {
		return nil, err
	}
	ps := make([]ocispec.Platform, len(platformSS))
	for i := range platformSS {
		var err error
		ps[i], err = platforms.Parse(platformSS[i])
		if err != nil {
			return nil, err
		}
	}
	return ps, nil
}
