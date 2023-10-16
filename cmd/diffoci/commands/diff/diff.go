package diff

import (
	"fmt"
	"os"

	"github.com/containerd/containerd/platforms"
	"github.com/containerd/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend/backendmanager"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/flagutil"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/imagegetter"
	"github.com/reproducible-containers/diffoci/pkg/diff"
	"github.com/reproducible-containers/diffoci/pkg/localpathutil"
	"github.com/spf13/cobra"
)

const Example = `  # Basic
  diffoci diff --semantic alpine:3.18.2 alpine:3.18.3

  # Dump conflicting files to ~/diff
  diffoci diff --semantic --report-dir=~/diff alpine:3.18.2 alpine:3.18.3

  # Compare local Docker images
  diffoci diff --semantic docker://foo docker://bar
`

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diff IMAGE0 IMAGE1",
		Short:   "Diff images",
		Example: Example,
		Args:    cobra.ExactArgs(2),

		PreRunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			if semantic, _ := cmd.Flags().GetBool("semantic"); semantic {
				flagNames := []string{
					"ignore-timestamps",
					"ignore-history",
					"ignore-file-order",
					"ignore-file-mode-redundant-bits",
					"ignore-image-name",
				}
				for _, f := range flagNames {
					if err := flags.Set(f, "true"); err != nil {
						return err
					}
				}
			}
			return nil
		},
		RunE: action,

		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flagutil.AddPlatformFlags(flags)

	flags.Bool("ignore-timestamps", false, "Ignore timestamps")
	flags.Bool("ignore-history", false, "Ignore history")
	flags.Bool("ignore-file-order", false, "Ignore file order in tar layers")
	flags.Bool("ignore-file-mode-redundant-bits", false, "Ignore redundant bits of file mode")
	flags.Bool("ignore-image-name", false, "Ignore image name annotation")
	flags.Bool("semantic", false, "[Recommended] Alias for --ignore-*=true")

	flags.Bool("verbose", false, "Verbose output")
	flags.String("report-file", "", "Create a report file to the specified path")
	flags.String("report-dir", "", "Create a detailed report in the specified directory")
	flags.String("pull", imagegetter.PullMissing, "Pull mode (always|missing|never)")
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
	platMC := platforms.Any(plats...)

	var options diff.Options
	options.IgnoreTimestamps, err = flags.GetBool("ignore-timestamps")
	if err != nil {
		return err
	}
	options.IgnoreHistory, err = flags.GetBool("ignore-history")
	if err != nil {
		return err
	}
	options.IgnoreFileOrder, err = flags.GetBool("ignore-file-order")
	if err != nil {
		return err
	}
	options.IgnoreFileModeRedundantBits, err = flags.GetBool("ignore-file-mode-redundant-bits")
	if err != nil {
		return err
	}
	options.IgnoreImageName, err = flags.GetBool("ignore-image-name")
	if err != nil {
		return err
	}
	options.ReportFile, err = flags.GetString("report-file")
	if err != nil {
		return err
	}
	if options.ReportFile != "" {
		options.ReportFile, err = localpathutil.Expand(options.ReportFile)
		if err != nil {
			return fmt.Errorf("invalid report-file path %q: %w", options.ReportFile, err)
		}
	}
	options.ReportDir, err = flags.GetString("report-dir")
	if err != nil {
		return err
	}
	if options.ReportDir != "" {
		options.ReportDir, err = localpathutil.Expand(options.ReportDir)
		if err != nil {
			return fmt.Errorf("invalid report-dir path %q: %w", options.ReportDir, err)
		}
	}

	options.EventHandler = diff.DefaultEventHandler
	verbose, err := flags.GetBool("verbose")
	if err != nil {
		return err
	}
	if verbose {
		options.EventHandler = diff.VerboseEventHandler
	}

	pullMode, err := flags.GetString("pull")
	if err != nil {
		return err
	}

	ig, err := imagegetter.New(cmd.ErrOrStderr(), backend)
	if err != nil {
		return err
	}

	var imageDescs [2]ocispec.Descriptor
	for i := 0; i < 2; i++ {
		img, err := ig.Get(ctx, args[i], plats, imagegetter.PullMode(pullMode))
		if err != nil {
			return err
		}
		log.G(ctx).Debugf("Input %d: Image %q (%s)", i, img.Name, img.Target.Digest)
		imageDescs[i] = img.Target
	}

	contentStore := backend.ContentStore()

	var exitCode int
	report, err := diff.Diff(ctx, contentStore, imageDescs, platMC, &options)
	if report != nil && len(report.Children) > 0 {
		exitCode = 1
	}
	if err != nil {
		log.G(ctx).Error(err)
		exitCode = 2
	}
	if exitCode != 0 {
		log.G(ctx).Debugf("exiting with code %d", exitCode)
	}
	os.Exit(exitCode)
	/* NOTREACHED */
	return nil
}
