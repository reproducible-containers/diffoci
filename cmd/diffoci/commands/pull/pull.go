package pull

import (
	"fmt"

	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/flagutil"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/imagegetter"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "pull IMAGE",
		Short:                 "Pull an image",
		Args:                  cobra.ExactArgs(1),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flagutil.AddPlatformFlags(flags)
	return cmd
}

func action(cmd *cobra.Command, args []string) error {
	backend, err := backendmanager.NewBackend(cmd)
	if err != nil {
		return err
	}
	ctx := backend.Context(cmd.Context())
	flags := cmd.Flags()
	plats, err := flagutil.ParsePlatformFlags(flags)
	if err != nil {
		return err
	}
	ig, err := imagegetter.New(cmd.ErrOrStderr(), backend)
	if err != nil {
		return err
	}

	img, err := ig.Get(ctx, args[0], plats, imagegetter.PullAlways)
	if err != nil {
		return err
	}
	stdout := cmd.OutOrStdout()
	fmt.Fprintln(stdout, "Digest: "+img.Target.Digest)
	fmt.Fprintln(stdout, img.Name)
	return nil
}
