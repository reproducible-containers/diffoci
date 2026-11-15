package types

import (
	"archive/tar"
	"time"

	"github.com/containerd/containerd/v2/pkg/archive/compression"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type GoldenTests struct {
	Name   string
	Images [2]Image
	Tests  []GoldenTest
}

type GoldenTest struct {
	Args   []string
	Output string
}

type Image struct {
	Layers      [][]Entry
	Compression compression.Compression
	Config      ocispec.Image
	Annotations map[string]string
	Manifests   []Image
}

type Entry struct {
	Header tar.Header
	Body   string
}

var Epoch = time.Unix(0, 0).UTC()

func File(name, body string) Entry {
	return Entry{Header: tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o644, ModTime: Epoch, Format: tar.FormatPAX}, Body: body}
}

func Dir(name string) Entry {
	return Entry{Header: tar.Header{Typeflag: tar.TypeDir, Name: name, Mode: 0o755, ModTime: Epoch, Format: tar.FormatPAX}}
}
