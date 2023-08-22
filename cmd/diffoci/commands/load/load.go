package load

import (
	"fmt"
	"os"

	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/flagutil"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/imagegetter"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "load < a.tar",
		Short:                 "Load an image archive (Docker or OCI) from STDIN",
		Args:                  cobra.ExactArgs(0),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flagutil.AddPlatformFlags(flags)
	flags.String("input", "", "Read from tar archive file, instead of STDIN")
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
	r := cmd.InOrStdin()
	input, err := flags.GetString("input")
	if err != nil {
		return err
	}
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	stdout := cmd.OutOrStdout()
	if err := imagegetter.Load(ctx, stdout, backend, r, plats, ""); err != nil {
		return fmt.Errorf("failed to load: %w", err)
	}
	return nil
}
