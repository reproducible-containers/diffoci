package golden

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/pkg/archive/compression"
	contentlocal "github.com/containerd/containerd/v2/plugins/content/local"
	"github.com/containerd/platforms"
	"github.com/google/go-cmp/cmp"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	diffcmd "github.com/reproducible-containers/diffoci/cmd/diffoci/commands/diff"
	"github.com/reproducible-containers/diffoci/cmd/diffoci/flagutil"
	testapt "github.com/reproducible-containers/diffoci/golden/testdata/test_apt"
	testbuildkitvslegacy "github.com/reproducible-containers/diffoci/golden/testdata/test_buildkit_vs_legacy"
	testbusyboxlinks "github.com/reproducible-containers/diffoci/golden/testdata/test_busybox_links"
	testchown "github.com/reproducible-containers/diffoci/golden/testdata/test_chown"
	testdockermanifestlist "github.com/reproducible-containers/diffoci/golden/testdata/test_docker_manifest_list"
	testdockermediatypes "github.com/reproducible-containers/diffoci/golden/testdata/test_docker_media_types"
	testdotprefix "github.com/reproducible-containers/diffoci/golden/testdata/test_dot_prefix"
	testgnutar "github.com/reproducible-containers/diffoci/golden/testdata/test_gnu_tar"
	testidentical "github.com/reproducible-containers/diffoci/golden/testdata/test_identical"
	testlayertoolarge "github.com/reproducible-containers/diffoci/golden/testdata/test_layer_too_large"
	testlongpath "github.com/reproducible-containers/diffoci/golden/testdata/test_long_path"
	testmultiplatform "github.com/reproducible-containers/diffoci/golden/testdata/test_multiplatform"
	testrebuild "github.com/reproducible-containers/diffoci/golden/testdata/test_rebuild"
	testrecompressed "github.com/reproducible-containers/diffoci/golden/testdata/test_recompressed"
	testretag "github.com/reproducible-containers/diffoci/golden/testdata/test_retag"
	testrootentry "github.com/reproducible-containers/diffoci/golden/testdata/test_root_entry"
	testsingleplatformsave "github.com/reproducible-containers/diffoci/golden/testdata/test_single_platform_save"
	testsquash "github.com/reproducible-containers/diffoci/golden/testdata/test_squash"
	testsubsecondmtime "github.com/reproducible-containers/diffoci/golden/testdata/test_subsecond_mtime"
	testswappedinputs "github.com/reproducible-containers/diffoci/golden/testdata/test_swapped_inputs"
	testxattr "github.com/reproducible-containers/diffoci/golden/testdata/test_xattr"
	testzstd "github.com/reproducible-containers/diffoci/golden/testdata/test_zstd"
	"github.com/reproducible-containers/diffoci/golden/types"
	"github.com/reproducible-containers/diffoci/pkg/diff"
)

var allTests = []types.GoldenTests{
	testapt.Tests,
	testbuildkitvslegacy.Tests,
	testbusyboxlinks.Tests,
	testchown.Tests,
	testdockermanifestlist.Tests,
	testdockermediatypes.Tests,
	testdotprefix.Tests,
	testgnutar.Tests,
	testidentical.Tests,
	testlayertoolarge.Tests,
	testlongpath.Tests,
	testmultiplatform.Tests,
	testrebuild.Tests,
	testrecompressed.Tests,
	testzstd.Tests,
	testretag.Tests,
	testrootentry.Tests,
	testsingleplatformsave.Tests,
	testsquash.Tests,
	testsubsecondmtime.Tests,
	testswappedinputs.Tests,
	testxattr.Tests,
}

var update = flag.Bool("update", false, "Whether to update expected output instead of testing it")

var defaultPlatform = ocispec.Platform{OS: "linux", Architecture: "amd64"}

// archive/tar returns mtimes in local time, and the output prints them as such.
func init() {
	time.Local = time.UTC
}

func TestGolden(t *testing.T) {
	for _, tests := range allTests {
		t.Run(tests.Name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			cs, err := contentlocal.NewStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			descs := [2]ocispec.Descriptor{
				writeImage(ctx, t, cs, tests.Images[0]),
				writeImage(ctx, t, cs, tests.Images[1]),
			}
			for _, test := range tests.Tests {
				t.Run(test.Output, func(t *testing.T) {
					got := runDiff(ctx, t, cs, descs, test.Args)
					p := filepath.Join("testdata", tests.Name, "outputs", test.Output)
					if *update {
						if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
							t.Fatal(err)
						}
						return
					}
					want, err := os.ReadFile(p)
					if err != nil {
						t.Fatal(err)
					}
					if d := cmp.Diff(string(want), got); d != "" {
						t.Errorf("%s mismatch (-want +got):\n%s", p, d)
					}
				})
			}
		})
	}
}

// mirrors action in cmd/diffoci/commands/diff, with the images read from cs instead of a backend
func runDiff(ctx context.Context, t *testing.T, cs content.Provider, descs [2]ocispec.Descriptor, args []string) string {
	t.Helper()
	reportDir := t.TempDir()
	argv := make([]string, len(args))
	for i, a := range args {
		argv[i] = strings.ReplaceAll(a, "$REPORT_DIR", reportDir)
	}
	cmd := diffcmd.NewCommand()
	flags := cmd.Flags()
	if err := cmd.ParseFlags(argv); err != nil {
		t.Fatal(err)
	}
	if !flags.Changed("platform") {
		if err := flags.Set("platform", platforms.Format(defaultPlatform)); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if flags.Changed("verbose") {
		t.Fatal("--verbose output is randomized by go-cmp")
	}
	plats, err := flagutil.ParsePlatformFlags(flags)
	if err != nil {
		t.Fatal(err)
	}
	var options diff.Options
	if err := diffcmd.ParseOptions(ctx, flags, &options); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	fmt.Fprintln(&stdout, strings.Join(append([]string{"$ diffoci diff"}, args...), " "))
	options.EventHandler = diff.NewDefaultEventHandler(&stdout)
	var exitCode int
	report, err := diff.Diff(ctx, cs, descs, platforms.Any(plats...), &options)
	if report != nil && len(report.Children) > 0 {
		exitCode = 1
	}
	if err != nil {
		fmt.Fprintf(&stdout, "error: %v\n", err)
		exitCode = 2
	}
	fmt.Fprintf(&stdout, "exit %d\n", exitCode)
	if options.ReportDir != "" {
		if err := filepath.WalkDir(reportDir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				fmt.Fprintf(&stdout, "$REPORT_DIR%s\n", strings.TrimPrefix(p, reportDir))
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	return stdout.String()
}

type mediaTypes struct {
	index, manifest, config string
	layer                   map[compression.Compression]string
}

var ociMediaTypes = mediaTypes{
	index:    ocispec.MediaTypeImageIndex,
	manifest: ocispec.MediaTypeImageManifest,
	config:   ocispec.MediaTypeImageConfig,
	layer: map[compression.Compression]string{
		compression.Uncompressed: ocispec.MediaTypeImageLayer,
		compression.Gzip:         ocispec.MediaTypeImageLayerGzip,
		compression.Zstd:         ocispec.MediaTypeImageLayerZstd,
	},
}

var dockerMediaTypes = mediaTypes{
	index:    images.MediaTypeDockerSchema2ManifestList,
	manifest: images.MediaTypeDockerSchema2Manifest,
	config:   images.MediaTypeDockerSchema2Config,
	layer: map[compression.Compression]string{
		compression.Uncompressed: images.MediaTypeDockerSchema2Layer,
		compression.Gzip:         images.MediaTypeDockerSchema2LayerGzip,
	},
}

func writeImage(ctx context.Context, t *testing.T, cs content.Ingester, img types.Image) ocispec.Descriptor {
	t.Helper()
	mt := ociMediaTypes
	if img.Docker {
		mt = dockerMediaTypes
	}
	if len(img.Manifests) > 0 {
		var manifests []ocispec.Descriptor
		for _, m := range img.Manifests {
			if m.Config.Architecture == "" {
				m.Config.Platform = defaultPlatform
			}
			desc := writeImage(ctx, t, cs, m)
			desc.Platform = &m.Config.Platform
			manifests = append(manifests, desc)
		}
		return writeJSON(ctx, t, cs, mt.index, ocispec.Index{
			Versioned:   specs.Versioned{SchemaVersion: 2},
			MediaType:   mt.index,
			Manifests:   manifests,
			Annotations: img.Annotations,
		})
	}
	config := img.Config
	if config.Architecture == "" {
		config.Platform = defaultPlatform
	}
	config.RootFS.Type = "layers"
	var layers []ocispec.Descriptor
	for _, l := range img.Layers {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		for _, e := range l {
			hdr := e.Header
			hdr.Size = int64(len(e.Body))
			if err := tw.WriteHeader(&hdr); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write([]byte(e.Body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		config.RootFS.DiffIDs = append(config.RootFS.DiffIDs, digest.FromBytes(buf.Bytes()))
		var compressed bytes.Buffer
		zw, err := compression.CompressStream(&compressed, img.Compression)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := zw.Write(buf.Bytes()); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		layers = append(layers, writeBlob(ctx, t, cs, mt.layer[img.Compression], compressed.Bytes()))
	}
	return writeJSON(ctx, t, cs, mt.manifest, ocispec.Manifest{
		Versioned:   specs.Versioned{SchemaVersion: 2},
		MediaType:   mt.manifest,
		Config:      writeJSON(ctx, t, cs, mt.config, config),
		Layers:      layers,
		Annotations: img.Annotations,
	})
}

func writeJSON(ctx context.Context, t *testing.T, cs content.Ingester, mediaType string, v any) ocispec.Descriptor {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return writeBlob(ctx, t, cs, mediaType, b)
}

func writeBlob(ctx context.Context, t *testing.T, cs content.Ingester, mediaType string, b []byte) ocispec.Descriptor {
	t.Helper()
	desc := ocispec.Descriptor{MediaType: mediaType, Digest: digest.FromBytes(b), Size: int64(len(b))}
	if err := content.WriteBlob(ctx, cs, desc.Digest.String(), bytes.NewReader(b), desc); err != nil {
		t.Fatal(err)
	}
	return desc
}
