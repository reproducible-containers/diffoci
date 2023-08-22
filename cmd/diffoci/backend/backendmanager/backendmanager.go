package backendmanager

import (
	"fmt"

	"github.com/containerd/containerd/log"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/containerdbackend"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/localbackend"
	"github.com/reproducible-containers/diffoci/pkg/envutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func AddFlags(flags *pflag.FlagSet) {
	containerdbackend.AddFlags(flags)
	localbackend.AddFlags(flags)
	flags.String("backend", envutil.String("DIFFOCI_BACKEND", "auto"),
		"backend (auto|containerd|local) [$DIFFOCI_BACKEND]")
}

func NewBackend(cmd *cobra.Command) (backend.Backend, error) {
	ctx := cmd.Context()
	flags := cmd.Flags()
	b, err := flags.GetString("backend")
	if err != nil {
		return nil, err
	}
	switch b {
	case "auto":
		cb, err := containerdbackend.New(cmd)
		if err == nil {
			log.G(ctx).Debug("auto backend: choosing \"containerd\"")
			return cb, nil
		}
		log.G(ctx).WithError(err).Debug("auto backend: failed to choose \"containerd\", falling back to \"local\"")
		return localbackend.New(cmd)
	case "containerd":
		return containerdbackend.New(cmd)
	case "local":
		return localbackend.New(cmd)
	default:
		return nil, fmt.Errorf("unknown backend %q (valid values are \"auto\", \"containerd\", and \"local\")", b)
	}
}
