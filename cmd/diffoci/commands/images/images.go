package images

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	refdocker "github.com/containerd/containerd/reference/docker"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "images",
		Short:                 "List images",
		Aliases:               []string{"list", "ls"},
		Args:                  cobra.NoArgs,
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
	var o printImageOptions
	imgStore := backend.ImageService()
	imgs, err := imgStore.List(ctx)
	if err != nil {
		return err
	}
	contentStore := backend.ContentStore()
	w := cmd.OutOrStdout()
	tw := tabwriter.NewWriter(w, 4, 8, 4, ' ', 0)
	defer tw.Flush()
	fmt.Fprintln(tw, "REPOSITORY\tTAG\tIMAGE ID\tPLATFORM")
	for _, img := range imgs {
		plats, err := images.Platforms(ctx, contentStore, img.Target)
		if err != nil {
			log.G(ctx).WithError(err).Debugf("failed to get platforms for %q", img.Name)
			if err2 := printImage(tw, img, "", o); err2 != nil {
				log.G(ctx).WithError(err2).Warnf("Failed to print image %q", img.Name)
			}
		} else {
			for _, plat := range plats {
				if avail, _, _, _, _ := images.Check(ctx, contentStore, img.Target, platforms.OnlyStrict(plat)); avail {
					platStr := platforms.Format(plat)
					if err := printImage(tw, img, platStr, o); err != nil {
						log.G(ctx).WithError(err).Warnf("Failed to print image %q for %q", img.Name, platStr)
					}
				}
			}
		}
	}
	return nil
}

type printImageOptions struct {
	// reserved
}

func printImage(w io.Writer, img images.Image, plat string, o printImageOptions) error {
	ref, err := refdocker.ParseDockerRef(img.Name)
	if err != nil {
		return err
	}
	repo := refdocker.FamiliarName(ref)
	var tag string
	if tagged, ok := ref.(refdocker.Tagged); ok {
		tag = tagged.Tag()
	}
	imgId := img.Target.Digest
	_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", repo, tag, imgId, plat)
	return err
}
