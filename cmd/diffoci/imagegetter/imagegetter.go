package imagegetter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/containerd/containerd/archive/compression"
	ctrimages "github.com/containerd/containerd/cmd/ctr/commands/images"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/pkg/transfer"
	"github.com/containerd/containerd/pkg/transfer/archive"
	"github.com/containerd/containerd/pkg/transfer/image"
	transimage "github.com/containerd/containerd/pkg/transfer/image"
	"github.com/containerd/containerd/pkg/transfer/registry"
	"github.com/containerd/errdefs"
	"github.com/containerd/log"
	"github.com/containerd/platforms"
	refdocker "github.com/distribution/reference"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/backend"
	"github.com/reproducible-containers/diffoci/pkg/dockercred"
	"github.com/reproducible-containers/diffoci/pkg/platformutil"
)

func wrapTransferProgressFunc(ctx context.Context, pf transfer.ProgressFunc) transfer.ProgressFunc {
	return func(p transfer.Progress) {
		log.G(ctx).Debugf("transfer progress %+v", p)
		pf(p)
	}
}

func Load(ctx context.Context, stdout io.Writer, transferrer transfer.Transferrer, tarR io.Reader, plats []ocispec.Platform, foreknownRef string) error {
	decompressed, err := compression.DecompressStream(tarR)
	if err != nil {
		return err
	}
	iis := archive.NewImageImportStream(decompressed, "")

	sOpts := []transimage.StoreOpt{
		transimage.WithPlatforms(plats...),
		image.WithPlatforms(plats...),
		image.WithAllMetadata,
		image.WithNamedPrefix("unused", true),
	}
	is := transimage.NewStore(foreknownRef, sOpts...)

	pf, done := ctrimages.ProgressHandler(ctx, stdout)
	defer done()

	if err := transferrer.Transfer(ctx, iis, is, transfer.WithProgress(wrapTransferProgressFunc(ctx, pf))); err != nil {
		return fmt.Errorf("failed to load: %w", err)
	}
	return nil
}

func Pull(ctx context.Context, stdout io.Writer, transferrer transfer.Transferrer, credHelper registry.CredentialHelper, ref string, plats []ocispec.Platform) error {
	reg := registry.NewOCIRegistry(ref, nil, credHelper)

	sOpts := []transimage.StoreOpt{
		transimage.WithPlatforms(plats...),
	}
	is := transimage.NewStore(ref, sOpts...)

	pf, done := ctrimages.ProgressHandler(ctx, stdout)
	defer done()

	if err := transferrer.Transfer(ctx, reg, is, transfer.WithProgress(wrapTransferProgressFunc(ctx, pf))); err != nil {
		return fmt.Errorf("failed to pull %q: %w", ref, err)
	}
	return nil
}

type ImageGetter struct {
	progressWriter io.Writer // stderr
	imageStore     images.Store
	contentStore   content.Store
	transferrer    transfer.Transferrer
	credHelper     registry.CredentialHelper
}

func New(progressWriter io.Writer, backend backend.Backend) (*ImageGetter, error) {
	credHelper, err := dockercred.NewCredentialHelper()
	if err != nil {
		return nil, err
	}
	return &ImageGetter{
		progressWriter: progressWriter,
		imageStore:     backend.ImageService(),
		contentStore:   backend.ContentStore(),
		transferrer:    backend,
		credHelper:     credHelper,
	}, nil
}

type PullMode string

const (
	PullAlways  = "always"
	PullMissing = "missing"
	PullNever   = "never"

	dockerImagePrefix = "docker://"
	podmanImagePrefix = "podman://"
)

func (g *ImageGetter) isDocker(rawRef string) bool {
	return strings.HasPrefix(rawRef, dockerImagePrefix)
}

func (g *ImageGetter) isPodman(rawRef string) bool {
	return strings.HasPrefix(rawRef, podmanImagePrefix)
}

func (g *ImageGetter) getDocker(ctx context.Context, rawRef string, plats []ocispec.Platform) (*images.Image, error) {
	rawRefTrimmed := strings.TrimPrefix(rawRef, dockerImagePrefix)
	ref, err := refdocker.ParseDockerRef(rawRefTrimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", rawRefTrimmed, err)
	}
	name := ref.String()
	docker := os.Getenv("DOCKER")
	if docker == "" {
		docker = "docker"
	}
	return g.loadDocker(ctx, docker, name, plats)
}

func (g *ImageGetter) getPodman(ctx context.Context, rawRef string, plats []ocispec.Platform) (*images.Image, error) {
	rawRefTrimmed := strings.TrimPrefix(rawRef, podmanImagePrefix)
	ref, err := refdocker.ParseDockerRef(rawRefTrimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", rawRefTrimmed, err)
	}
	name := ref.String()
	podman := os.Getenv("PODMAN")
	if podman == "" {
		podman = "podman"
	}
	return g.loadDocker(ctx, podman, name, plats)
}

type readerWithEOF struct {
	io.Reader
}

func (r *readerWithEOF) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if errors.Is(err, os.ErrClosed) {
		err = io.EOF
	}
	return n, err
}

// getDockerMajorVersion returns the major version of Docker CLI.
// Returns 0 if version cannot be determined.
func getDockerMajorVersion(ctx context.Context, docker string) int {
	cmd := exec.Command(docker, "version", "--format", "{{.Client.Version}}")
	output, err := cmd.Output()
	if err != nil {
		log.G(ctx).Debugf("Failed to get %s version: %v", docker, err)
		return 0
	}
	versionStr := strings.TrimSpace(string(output))
	// Parse major version from version string
	parts := strings.Split(versionStr, ".")
	if len(parts) < 1 {
		return 0
	}
	var major int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		log.G(ctx).Debugf("Failed to parse %s version %q: %v", docker, versionStr, err)
		return 0
	}
	return major
}

// loadDocker runs `docker save` and loads the result
func (g *ImageGetter) loadDocker(ctx context.Context, docker, name string, plats []ocispec.Platform) (*images.Image, error) {
	log.G(ctx).Infof("Loading image %q from %q", name, docker)

	// Build docker save command with platform filtering
	// Docker v28+ supports --platform flag to save only specific platforms
	// References:
	// - v28 (single platform): https://github.com/docker/cli/commit/a20eb45b26aa600cb2e942eac743b2d54445d01d
	// - v29 (multi-platform):  https://github.com/docker/cli/commit/8993f54fc32829355d1e0f5949d3d241bcae7bff
	args := []string{"save"}
	if len(plats) > 0 {
		majorVersion := getDockerMajorVersion(ctx, docker)
		// --platform flag was introduced in Docker v28
		if majorVersion >= 28 {
			platStrings := make([]string, len(plats))
			for i, p := range plats {
				platStrings[i] = platforms.Format(p)
			}
			args = append(args, "--platform", strings.Join(platStrings, ","))
		} else {
			// TODO: Once Docker v25 reaches EOL, consider requiring Docker v28+ for docker:// images
			// and returning an error here instead
			log.G(ctx).Debugf("%s version %d does not support --platform flag for save (requires v28+)", docker, majorVersion)
		}
	}
	args = append(args, name)

	dockerCmd := exec.Command(docker, args...)
	dockerCmd.Stderr = os.Stderr
	r, err := dockerCmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	log.G(ctx).Debugf("Running %v", dockerCmd.Args)
	if err = dockerCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to run %v: %w", dockerCmd.Args, err)
	}
	if err = Load(ctx, g.progressWriter, g.transferrer, &readerWithEOF{r}, plats, name); err != nil {
		return nil, fmt.Errorf("failed to load an archive (from %v): %w", dockerCmd.Args, err)
	}
	if err = r.Close(); err != nil {
		return nil, err
	}
	img, err := g.imageStore.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("should have loaded an archive (from %v), but the loaded image is not accessible: %w", dockerCmd.Args, err)
	}

	// Check platforms
	platMC := platforms.Any(plats...)
	available, _, _, _, err := images.Check(ctx, g.contentStore, img.Target, platMC)
	if err != nil {
		return nil, err
	}
	if !available {
		return nil, fmt.Errorf("image %q lacks blobs for additional platforms (%v): %w",
			name, platformutil.FormatSlice(plats), errdefs.ErrUnavailable)
	}
	return &img, nil
}

func (g *ImageGetter) Get(ctx context.Context, rawRef string, plats []ocispec.Platform, pullMode PullMode) (*images.Image, error) {
	if g.isDocker(rawRef) {
		return g.getDocker(ctx, rawRef, plats)
	}
	if g.isPodman(rawRef) {
		return g.getPodman(ctx, rawRef, plats)
	}
	ref, err := refdocker.ParseDockerRef(rawRef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %w", rawRef, err)
	}
	name := ref.String()

	switch pullMode {
	case PullAlways:
		log.G(ctx).Infof("Pulling %q", name)
		if err := Pull(ctx, g.progressWriter, g.transferrer, g.credHelper, name, plats); err != nil {
			return nil, fmt.Errorf("failed to pull %q: %w", name, err)
		}
	case PullMissing, PullNever:
		// NOP
	default:
		return nil, fmt.Errorf("unknown pull mode %q", pullMode)
	}

	// Get the image object
	img, err := g.imageStore.Get(ctx, name)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) && pullMode != PullNever {
			log.G(ctx).Infof("Pulling %q", name)
			if pullErr := Pull(ctx, g.progressWriter, g.transferrer, g.credHelper, name, plats); pullErr != nil {
				return nil, fmt.Errorf("failed to pull %q: %w", name, pullErr)
			}
			var retryErr error
			img, retryErr = g.imageStore.Get(ctx, name)
			if retryErr != nil {
				return nil, fmt.Errorf("should have pulled %q, but still not accessible in the local store: %w", name, retryErr)
			}
			err = nil
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image %q: %w", name, err)
	}

	// Check platforms
	platMC := platforms.Any(plats...)
	available, _, _, _, err := images.Check(ctx, g.contentStore, img.Target, platMC)
	if err != nil {
		return nil, err
	}
	if !available {
		if pullMode == PullNever {
			return nil, fmt.Errorf("image %q lacks blobs for additional platforms (%s): %w",
				name, platformutil.FormatSlice(plats), errdefs.ErrUnavailable)
		} else {
			log.G(ctx).Infof("Pulling %q for additional platforms (%s)", name, platformutil.FormatSlice(plats))
			if err := Pull(ctx, g.progressWriter, g.transferrer, g.credHelper, name, plats); err != nil {
				return nil, fmt.Errorf("failed to pull %q: %w", name, err)
			}
		}
	}
	return &img, nil
}
