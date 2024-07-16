package remove

import (
	"errors"
	"fmt"

	"github.com/containerd/containerd/images"
	refdocker "github.com/distribution/reference"
	"github.com/containerd/log"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove IMAGE...",
		Aliases: []string{"rm", "rmi"},
		Short:   "Remove images",
		Long: `Remove images

BUG: when using the "local" backend, the remove command
does not remove blobs associated with the image.
(Workaround: remove the cache directory periodically by yourself.)
		`,
		Args:                  cobra.MinimumNArgs(1),
		RunE:                  action,
		DisableFlagsInUseLine: true,
	}
	return cmd
}

func action(cmd *cobra.Command, args []string) error {
	backend, err := backendmanager.NewBackend(cmd)
	if err != nil {
		return err
	}
	ctx := backend.Context(cmd.Context())
	imageStore := backend.ImageService()
	stdout := cmd.OutOrStdout()
	var errs []error
	for _, rawRef := range args {
		ref, err := refdocker.ParseDockerRef(rawRef)
		if err != nil {
			return fmt.Errorf("failed to parse %q: %w", rawRef, err)
		}
		img, err := imageStore.Get(ctx, ref.String())
		if err != nil {
			return err
		}
		if err := imageStore.Delete(ctx, img.Name, images.SynchronousDelete()); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %q: %w", img.Name, err))
		}
		fmt.Fprintln(stdout, img.Name)
	}
	if gcErr := backend.MaybeGC(ctx); gcErr != nil {
		log.G(ctx).WithError(gcErr).Warn("Failed to do GC")
	}
	return errors.Join(errs...)
}
