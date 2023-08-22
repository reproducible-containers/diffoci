package main

import (
	_ "crypto/sha256"

	"github.com/containerd/containerd/log"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/diff"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/images"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/info"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/load"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/pull"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/commands/remove"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/version"
	"github.com/reproducible-containers/diffoci/pkg/envutil"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		log.L.Fatal(err)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "diffoci",
		Short:         "diff for container images",
		Example:       diff.Example,
		Version:       version.GetVersion(),
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	flags := cmd.PersistentFlags()
	flags.Bool("debug", envutil.Bool("DEBUG", false), "debug mode [$DEBUG]")
	backendmanager.AddFlags(flags)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			if err := log.SetLevel(log.DebugLevel.String()); err != nil {
				log.L.WithError(err).Warn("Failed to enable debug logs")
			}
		}
		return nil
	}

	cmd.AddCommand(
		diff.NewCommand(),
		images.NewCommand(),
		pull.NewCommand(),
		load.NewCommand(),
		remove.NewCommand(),
		info.NewCommand(),
	)
	return cmd
}
