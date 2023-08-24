package diff

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/platforms"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/reproducible-containers/diffoci/pkg/untar"
)

type IgnoranceOptions struct {
	IgnoreTimestamps            bool
	IgnoreHistory               bool
	IgnoreFileOrder             bool
	IgnoreFileModeRedundantBits bool
	IgnoreImageName             bool
}

type Options struct {
	IgnoranceOptions
	EventHandler
	ReportFile string
	ReportDir  string
}

func (o *Options) digestMayChange() bool {
	return o.IgnoranceOptions != IgnoranceOptions{}
}

func (o *Options) sizeMayChange() bool {
	// over-estimated
	return o.digestMayChange()
}

func Diff(ctx context.Context, cs content.Provider, descs [2]ocispec.Descriptor,
	platMC platforms.MatchComparer, opts *Options) (*EventTreeNode, error) {
	for i, desc := range descs {
		available, _, _, missing, err := images.Check(ctx, cs, desc, platMC)
		if err == nil && !available {
			err = errdefs.ErrUnavailable
		}
		if err != nil {
			log.G(ctx).Debugf("missing=%+v", missing)
			for _, f := range missing {
				if f.Platform != nil {
					p := platforms.Format(*f.Platform)
					return nil, fmt.Errorf("image %d is not available for platform %q: %w", i, p, err)
				}
			}
			return nil, fmt.Errorf("image %d is not available for the requested platform: %w", i, err)
		}
	}
	var o Options
	if opts != nil {
		o = *opts
	}
	if o.EventHandler == nil {
		o.EventHandler = DefaultEventHandler
	}
	var reportFiles []string
	if o.ReportFile != "" {
		reportFiles = append(reportFiles, o.ReportFile)
	}
	if o.ReportDir != "" {
		if err := os.MkdirAll(o.ReportDir, 0755); err != nil {
			return nil, err
		}
		for _, f := range ReportDirRootFilenames {
			p := filepath.Join(o.ReportDir, filepath.Clean(f))
			log.G(ctx).Debugf("Removing %q (if exists)", p)
			if err := os.RemoveAll(p); err != nil {
				return nil, err
			}
		}
		reportFiles = append(reportFiles, filepath.Join(o.ReportDir, ReportDirReportJSON))
	}
	d := differ{
		cs:     cs,
		platMC: platMC,
		o:      o,
	}
	eventTreeRootNode := &EventTreeNode{
		Context: "/",
	}
	inputs := [2]EventInput{
		{
			Descriptor: &descs[0],
		}, {
			Descriptor: &descs[1],
		},
	}
	var errs []error
	if err := d.diff(ctx, eventTreeRootNode, inputs); err != nil {
		errs = append(errs, err)
	}
	if flusher, ok := o.EventHandler.(Flusher); ok {
		if err := flusher.Flush(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, reportFile := range reportFiles {
		if err := writeReportFile(reportFile, eventTreeRootNode); err != nil {
			errs = append(errs, err)
		}
	}
	return eventTreeRootNode, errors.Join(errs...)
}

func writeReportFile(p string, node *EventTreeNode) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	return enc.Encode(node)
}

type differ struct {
	cs     content.Provider
	platMC platforms.MatchComparer
	o      Options
}

func (d *differ) raiseEvent(ctx context.Context, node *EventTreeNode, ev Event, evContextName string) error {
	return d.raiseEventWithEventTreeNode(ctx, node, &EventTreeNode{Context: path.Join(node.Context, path.Clean(evContextName)), Event: ev})
}

func (d *differ) raiseEventWithEventTreeNode(ctx context.Context, node, newNode *EventTreeNode) error {
	eventErr := d.o.EventHandler.HandleEventTreeNode(ctx, newNode)
	node.Append(newNode)
	return eventErr
}

func (d *differ) diff(ctx context.Context, node *EventTreeNode, in [2]EventInput) error {
	var errs []error
	negligibleFields := []string{"Annotations"}
	if d.o.digestMayChange() {
		negligibleFields = append(negligibleFields, "Digest", "Data")
	}
	if d.o.sizeMayChange() {
		negligibleFields = append(negligibleFields, "Size")
	}
	if diff := cmp.Diff(in[0].Descriptor, in[1].Descriptor,
		cmpopts.IgnoreFields(ocispec.Descriptor{}, negligibleFields...)); diff != "" {
		ev := Event{
			Type:   EventTypeDescriptorMismatch,
			Inputs: in,
			Diff:   diff,
		}
		if err := d.raiseEvent(ctx, node, ev, "desc"); err != nil {
			errs = append(errs, err)
		}
	}
	if err := d.diffAnnotationsField(ctx, node, in, EventTypeDescriptorMismatch,
		[2]map[string]string{
			in[0].Descriptor.Annotations,
			in[1].Descriptor.Annotations,
		}, "Annotations"); err != nil {
		errs = append(errs, err)
	}
	switch mt := in[0].Descriptor.MediaType; {
	case images.IsIndexType(mt):
		if err := d.diffIndex(ctx, node, in); err != nil {
			errs = append(errs, err)
		}
	case images.IsManifestType(mt):
		if err := d.diffManifest(ctx, node, in); err != nil {
			errs = append(errs, err)
		}
	case images.IsConfigType(mt):
		if err := d.diffConfig(ctx, node, in); err != nil {
			errs = append(errs, err)
		}
	case images.IsLayerType(mt):
		if err := d.diffLayer(ctx, node, in); err != nil {
			errs = append(errs, err)
		}
	default:
		log.G(ctx).Warnf("Unknown media type %q", mt)
		if diff := cmp.Diff(in[0].Descriptor, in[1].Descriptor); diff != "" {
			ev := Event{
				Type:   EventTypeDescriptorMismatch,
				Inputs: in,
				Diff:   diff,
			}
			if err := d.raiseEvent(ctx, node, ev, "desc"); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (d *differ) diffDescriptorPtrField(ctx context.Context, node *EventTreeNode, in [2]EventInput, evType EventType, descs [2]*ocispec.Descriptor, fieldName string) error {
	if (descs[0] != nil && descs[1] == nil) || (descs[0] == nil && descs[1] != nil) {
		ev := Event{
			Type:   evType,
			Inputs: in,
			Diff:   cmp.Diff(descs[0], descs[1]),
			Note:   fmt.Sprintf("field %q: only present in a single input", fieldName),
		}
		return d.raiseEvent(ctx, node, ev, strings.ToLower(fieldName))
	}
	if descs[0] == nil {
		return nil
	}
	newNode := EventTreeNode{
		Context: path.Join(node.Context, path.Clean(strings.ToLower(fieldName))),
		Event: Event{
			Type:   evType,
			Inputs: in,
			Diff:   cmp.Diff(*descs[0], *descs[1]),
			Note:   fmt.Sprintf("field %q", fieldName),
		},
	}
	childInputs := [2]EventInput{
		{
			Descriptor: descs[0],
		}, {
			Descriptor: descs[1],
		},
	}
	var err error
	if diffErr := d.diff(ctx, &newNode, childInputs); diffErr != nil {
		err = fmt.Errorf("field %q: %w", fieldName, diffErr)
	}
	if len(newNode.Children) > 0 {
		if err2 := d.raiseEventWithEventTreeNode(ctx, node, &newNode); err2 != nil {
			err = errors.Join(err, err2)
		}
	} // else no event happens
	return err
}

func (d *differ) diffDescriptorSliceField(ctx context.Context, node *EventTreeNode, in [2]EventInput, evType EventType, descSlices [2][]ocispec.Descriptor, fieldName string,
	maxEnts int, validateDesc func(ocispec.Descriptor) (tolerable bool, vErr error)) error {
	if len(descSlices[0]) != len(descSlices[1]) {
		ev := Event{
			Type:   evType,
			Inputs: in,
			Diff:   cmp.Diff(descSlices, descSlices),
			Note:   fmt.Sprintf("field %q: length mismatch", fieldName),
		}
		return d.raiseEvent(ctx, node, ev, strings.ToLower(fieldName))
	}
	if len(descSlices[0]) > maxEnts {
		return fmt.Errorf("field %q: too many manifests (> %d)", fieldName, maxEnts)
	}
	var errs []error
	// TODO: paralellize the loop
	for i := range descSlices[0] {
		i := i
		fieldNameI := fmt.Sprintf("%s[%d]", fieldName, i)
		newNode := EventTreeNode{
			Context: path.Join(node.Context, path.Clean(strings.ToLower(fieldName)+"-"+strconv.Itoa(i))),
			Event: Event{
				Type:   evType,
				Inputs: in,
				Diff:   cmp.Diff(descSlices[0], descSlices[1]),
				Note:   fmt.Sprintf("field %q", fieldNameI),
			},
		}
		if tolerable, err := validateDesc(descSlices[0][i]); err != nil {
			if !tolerable {
				errs = append(errs, fmt.Errorf("field %q: invalid: %w", fieldNameI, err))
			}
			continue
		}
		childInputs := [2]EventInput{
			{
				Descriptor: &descSlices[0][i],
			}, {
				Descriptor: &descSlices[1][i],
			},
		}
		if err := d.diff(ctx, &newNode, childInputs); err != nil {
			errs = append(errs, fmt.Errorf("field %q: %w", fieldNameI, err))
		}
		if len(newNode.Children) > 0 {
			if err2 := d.raiseEventWithEventTreeNode(ctx, node, &newNode); err2 != nil {
				errs = append(errs, err2)
			}
		} // else no event happens
	}
	return errors.Join(errs...)
}

func (d *differ) diffAnnotationsField(ctx context.Context, node *EventTreeNode, in [2]EventInput, evType EventType, maps [2]map[string]string, fieldName string) error {
	negligible := map[string]struct{}{}
	if d.o.IgnoreTimestamps {
		negligible[ocispec.AnnotationCreated] = struct{}{}
	}
	if d.o.IgnoreImageName {
		negligible[images.AnnotationImageName] = struct{}{} // "io.containerd.image.name": "docker.io/library/alpine:3.18"
		negligible[ocispec.AnnotationRefName] = struct{}{}  // "org.opencontainers.image.ref.name": "3.18"
	}
	if len(negligible) > 0 {
		for i := 0; i < 2; i++ {
			if maps[i] == nil {
				maps[i] = make(map[string]string)
			}
		}
	}
	discardFunc := func(k, _ string) bool {
		_, ok := negligible[k]
		return ok
	}
	if diff := cmp.Diff(maps[0], maps[1], cmpopts.IgnoreMapEntries(discardFunc)); diff != "" {
		ev := Event{
			Type:   evType,
			Inputs: in,
			Diff:   diff,
		}
		if fieldName != "" {
			ev.Note = fmt.Sprintf("field %q", fieldName)
		}
		return d.raiseEvent(ctx, node, ev, strings.ToLower(fieldName))
	}
	return nil
}

func (d *differ) diffIndex(ctx context.Context, node *EventTreeNode, in [2]EventInput) error {
	for i := 0; i < 2; i++ {
		var err error
		in[i].Index, err = readBlobWithType[ocispec.Index](ctx, d.cs, *in[i].Descriptor)
		if err != nil {
			return fmt.Errorf("failed to read index (%v): %w", in[i].Descriptor, err) // critical, not joined
		}
	}

	var negligibleFields []string
	if d.o.digestMayChange() {
		negligibleFields = append(negligibleFields, "Manifests", "Subject", "Annotations")
	}
	var errs []error
	if diff := cmp.Diff(*in[0].Index, *in[1].Index, cmpopts.IgnoreFields(ocispec.Index{}, negligibleFields...)); diff != "" {
		ev := Event{
			Type:   EventTypeIndexBlobMismatch,
			Inputs: in,
			Diff:   diff,
		}
		if err := d.raiseEvent(ctx, node, ev, "index"); err != nil {
			errs = append(errs, err)
		}
	}

	// Compare Manifests
	// TODO: allow comparing multi-platform image vs single-platform image
	if err := d.diffDescriptorSliceField(ctx, node, in, EventTypeIndexBlobMismatch, [2][]ocispec.Descriptor{
		in[0].Index.Manifests,
		in[1].Index.Manifests,
	}, "Manifests", maxManifests,
		func(desc ocispec.Descriptor) (tolerable bool, vErr error) {
			if !images.IsManifestType(desc.MediaType) {
				return false, fmt.Errorf("expected a manifest type, got %q", desc.MediaType)
			}
			if desc.Platform != nil && !d.platMC.Match(*desc.Platform) {
				return true, fmt.Errorf("unexpected platform %q", platforms.Format(*desc.Platform))
			}
			return true, nil
		}); err != nil {
		errs = append(errs, err)
	}

	// Compare Subject
	if err := d.diffDescriptorPtrField(ctx, node, in, EventTypeIndexBlobMismatch, [2]*ocispec.Descriptor{
		in[0].Index.Subject,
		in[1].Index.Subject,
	}, "Subject"); err != nil {
		errs = append(errs, err)
	}

	// Compare Annotations
	if err := d.diffAnnotationsField(ctx, node, in, EventTypeIndexBlobMismatch, [2]map[string]string{
		in[0].Index.Annotations,
		in[1].Index.Annotations,
	}, "Annotations"); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (d *differ) diffManifest(ctx context.Context, node *EventTreeNode, in [2]EventInput) error {
	if in[0].Descriptor.Platform != nil && !d.platMC.Match(*in[0].Descriptor.Platform) {
		return nil
	}
	for i := 0; i < 2; i++ {
		var err error
		in[i].Manifest, err = readBlobWithType[ocispec.Manifest](ctx, d.cs, *in[i].Descriptor)
		if err != nil {
			return fmt.Errorf("failed to read manifest (%v): %w", in[i].Descriptor, err)
		}
	}
	var negligibleFields []string
	if d.o.digestMayChange() {
		negligibleFields = append(negligibleFields, "Config", "Layers", "Subject", "Annotations")
	}
	var errs []error
	if diff := cmp.Diff(*in[0].Manifest, *in[1].Manifest, cmpopts.IgnoreFields(ocispec.Manifest{}, negligibleFields...)); diff != "" {
		ev := Event{
			Type:   EventTypeManifestBlobMismatch,
			Inputs: in,
			Diff:   diff,
		}
		if err := d.raiseEvent(ctx, node, ev, "manifest"); err != nil {
			errs = append(errs, err)
		}
	}

	// Compare Config
	if err := d.diffDescriptorPtrField(ctx, node, in, EventTypeManifestBlobMismatch, [2]*ocispec.Descriptor{
		&in[0].Manifest.Config,
		&in[1].Manifest.Config,
	}, "Config"); err != nil {
		errs = append(errs, err)
	}

	// Compare Layers
	if err := d.diffDescriptorSliceField(ctx, node, in, EventTypeManifestBlobMismatch, [2][]ocispec.Descriptor{
		in[0].Manifest.Layers,
		in[1].Manifest.Layers,
	}, "Layers", maxLayers,
		func(desc ocispec.Descriptor) (tolerable bool, vErr error) {
			if !images.IsLayerType(desc.MediaType) {
				return false, fmt.Errorf("expected a layer type, got %q", desc.MediaType)
			}
			return true, nil
		}); err != nil {
		errs = append(errs, err)
	}

	// Compare Subject
	if err := d.diffDescriptorPtrField(ctx, node, in, EventTypeManifestBlobMismatch, [2]*ocispec.Descriptor{
		in[0].Manifest.Subject,
		in[1].Manifest.Subject,
	}, "Subject"); err != nil {
		errs = append(errs, err)
	}

	// Compare Annotations
	if err := d.diffAnnotationsField(ctx, node, in, EventTypeManifestBlobMismatch, [2]map[string]string{
		in[0].Manifest.Annotations,
		in[1].Manifest.Annotations,
	}, "Annotations"); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (d *differ) diffConfig(ctx context.Context, node *EventTreeNode, in [2]EventInput) error {
	for i := 0; i < 2; i++ {
		var err error
		in[i].Config, err = readBlobWithType[ocispec.Image](ctx, d.cs, *in[i].Descriptor)
		if err != nil {
			return fmt.Errorf("failed to read config (%v): %w", in[i].Descriptor, err)
		}
	}
	var negligibleFields []string
	if d.o.digestMayChange() {
		negligibleFields = append(negligibleFields, "RootFS")
	}
	if d.o.IgnoreTimestamps {
		// history contains timestamps
		negligibleFields = append(negligibleFields, "Created", "History")
	}
	if d.o.IgnoreHistory {
		negligibleFields = append(negligibleFields, "History")
	}
	var errs []error
	if diff := cmp.Diff(*in[0].Config, *in[1].Config, cmpopts.IgnoreFields(ocispec.Image{}, negligibleFields...)); diff != "" {
		ev := Event{
			Type:   EventTypeConfigBlobMismatch,
			Inputs: in,
			Diff:   diff,
		}
		if err := d.raiseEvent(ctx, node, ev, "config"); err != nil {
			errs = append(errs, err)
		}
	}

	// Compare partial RootFS
	if slices.Contains(negligibleFields, "RootFS") {
		if diff := cmp.Diff(in[0].Config.RootFS, in[1].Config.RootFS, cmpopts.IgnoreFields(ocispec.RootFS{}, "DiffIDs")); diff != "" {
			ev := Event{
				Type:   EventTypeConfigBlobMismatch,
				Inputs: in,
				Diff:   diff,
				Note:   "field \"RootFS\"",
			}
			if err := d.raiseEvent(ctx, node, ev, "config/rootfs"); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Compare partial History
	if slices.Contains(negligibleFields, "History") && !d.o.IgnoreHistory {
		if len(in[0].Config.History) != len(in[1].Config.History) {
			ev := Event{
				Type:   EventTypeConfigBlobMismatch,
				Inputs: in,
				Diff:   cmp.Diff(in[0].Config.History, in[1].Config.History),
				Note:   "field \"History\": length mismatch",
			}
			if err := d.raiseEvent(ctx, node, ev, "config/history"); err != nil {
				errs = append(errs, err)
			}
		} else {
			var negligibleHistoryFields []string
			if d.o.IgnoreTimestamps {
				negligibleHistoryFields = append(negligibleHistoryFields, "Created")
			}
			for i := range in[0].Config.History {
				if diff := cmp.Diff(in[0].Config.History[i], in[1].Config.History[i],
					cmpopts.IgnoreFields(ocispec.History{}, negligibleHistoryFields...)); diff != "" {
					ev := Event{
						Type:   EventTypeConfigBlobMismatch,
						Inputs: in,
						Diff:   diff,
						Note:   fmt.Sprintf("field \"History[%d]\"", i),
					}
					if err := d.raiseEvent(ctx, node, ev, fmt.Sprintf("config/history-%d", i)); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (d *differ) diffLayer(ctx context.Context, node *EventTreeNode, in [2]EventInput) error {
	tr0, trCloser0, err := openTarReader(ctx, d.cs, *in[0].Descriptor)
	if err != nil {
		return err
	}
	defer func() {
		if trCloserErr0 := trCloser0(); trCloserErr0 != nil {
			log.G(ctx).WithError(trCloserErr0).Warn("failed to close tar reader 0")
		}
	}()

	tr1, trCloser1, err := openTarReader(ctx, d.cs, *in[1].Descriptor)
	if err != nil {
		return err
	}
	defer func() {
		if trCloserErr1 := trCloser1(); trCloserErr1 != nil {
			log.G(ctx).WithError(trCloserErr1).Warn("failed to close tar reader 1")
		}
	}()

	var (
		tarEntries0 []*TarEntry
		tarEntries1 []*TarEntry
		errs        []error
	)
	tarEntriesByName0 := make(map[string][]*TarEntry)
	tarEntriesByName1 := make(map[string][]*TarEntry)
	var finalizers []func() error
	defer func() {
		for _, finalizer := range finalizers {
			if finalizerErr := finalizer(); finalizerErr != nil {
				log.G(ctx).WithError(finalizerErr).Debug("Failed to execute a layer finalizer")
			}
		}
	}()
	for i := 0; ; i++ {
		hdr0, hdrErr0 := tr0.Next()
		hdr1, hdrErr1 := tr1.Next()
		if errors.Is(hdrErr0, io.EOF) && errors.Is(hdrErr1, io.EOF) {
			break
		}
		if errors.Is(hdrErr0, io.EOF) && !errors.Is(hdrErr1, io.EOF) {
			ev := Event{
				Type:   EventTypeLayerBlobMismatch,
				Inputs: in,
				Note:   fmt.Sprintf("input 1 is longer than input 0 (%d entries)", i+1),
			}
			if err := d.raiseEvent(ctx, node, ev, "layer"); err != nil {
				errs = append(errs, err)
			}
			break
		}
		if errors.Is(hdrErr1, io.EOF) && !errors.Is(hdrErr0, io.EOF) {
			ev := Event{
				Type:   EventTypeLayerBlobMismatch,
				Inputs: in,
				Note:   fmt.Sprintf("input 0 is longer than input 1 (%d entries)", i+1),
			}
			if err := d.raiseEvent(ctx, node, ev, "layer"); err != nil {
				errs = append(errs, err)
			}
			break
		}
		if hdrErr0 != nil {
			errs = append(errs, hdrErr0)
			break
		}
		if hdrErr1 != nil {
			errs = append(errs, hdrErr1)
			break
		}
		var (
			dgst           [2]digest.Digest
			extractedPaths [2]string
		)
		if repDir := d.o.ReportDir; repDir != "" {
			dirx := filepath.Clean(node.Context) // "/manifests-0/layers-0"
			dir0 := filepath.Join(repDir, ReportDirInput0, dirx)
			dir1 := filepath.Join(repDir, ReportDirInput1, dirx)
			untar0, err := untar.Entry(ctx, dir0, hdr0, tr0)
			if err != nil {
				errs = append(errs, err)
				break
			}
			untar1, err := untar.Entry(ctx, dir1, hdr1, tr1)
			if err != nil {
				errs = append(errs, err)
				break
			}
			dgst[0], dgst[1] = untar0.Digest, untar1.Digest
			extractedPaths[0], extractedPaths[1] = untar0.Path, untar1.Path
		} else {
			dgst[0], err = digest.SHA256.FromReader(tr0)
			if err != nil {
				errs = append(errs, err)
				break
			}
			dgst[1], err = digest.SHA256.FromReader(tr1)
			if err != nil {
				errs = append(errs, err)
				break
			}
		}
		ent0 := &TarEntry{
			Index:         i,
			Header:        hdr0,
			Digest:        dgst[0],
			extractedPath: extractedPaths[0],
		}
		ent1 := &TarEntry{
			Index:         i,
			Header:        hdr1,
			Digest:        dgst[1],
			extractedPath: extractedPaths[1],
		}
		tarEntries0 = append(tarEntries0, ent0)
		tarEntriesByName0[hdr0.Name] = append(tarEntriesByName0[hdr0.Name], ent0)
		tarEntries1 = append(tarEntries1, ent1)
		tarEntriesByName1[hdr1.Name] = append(tarEntriesByName1[hdr1.Name], ent1)
	}
	newNode := EventTreeNode{
		Context: path.Join(node.Context, "layer"),
		Event: Event{
			Type:   EventTypeLayerBlobMismatch,
			Inputs: in,
		},
	}
	var dirsToBeRemovedIfEmpty []string
	if d.o.IgnoreFileOrder {
		for name, ents0 := range tarEntriesByName0 {
			ents1 := tarEntriesByName1[name]
			if len(ents0) != len(ents1) {
				ev := Event{
					Type:   EventTypeLayerBlobMismatch,
					Inputs: in,
					Note:   eventNoteNameAppearanceMismatch(name, len(ents0), len(ents1)),
				}
				if err := d.raiseEvent(ctx, node /* not NewNode */, ev, "layer"); err != nil {
					errs = append(errs, err)
				}
				continue
			}
			dd, err := d.diffTarEntries(ctx, &newNode, in, [2][]*TarEntry{ents0, ents1})
			dirsToBeRemovedIfEmpty = append(dirsToBeRemovedIfEmpty, dd...)
			if err != nil {
				errs = append(errs, err)
			}
		}
		// Iterate again to find entries that only appear in input 1
		for name, ents1 := range tarEntriesByName1 {
			ents0 := tarEntriesByName0[name]
			if len(ents0) != len(ents1) {
				ev := Event{
					Type:   EventTypeLayerBlobMismatch,
					Inputs: in,
					Note:   eventNoteNameAppearanceMismatch(name, len(ents0), len(ents1)),
				}
				if err := d.raiseEvent(ctx, node /* not newNode */, ev, "layer"); err != nil {
					errs = append(errs, err)
				}
			}
		}
	} else {
		dd, err := d.diffTarEntries(ctx, &newNode, in, [2][]*TarEntry{tarEntries0, tarEntries1})
		dirsToBeRemovedIfEmpty = append(dirsToBeRemovedIfEmpty, dd...)
		if err != nil {
			errs = append(errs, err)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(dirsToBeRemovedIfEmpty)))
	for _, d := range dirsToBeRemovedIfEmpty {
		_ = os.Remove(d) // Not RemoveAll
	}

	if len(newNode.Children) > 0 {
		if err2 := d.raiseEventWithEventTreeNode(ctx, node, &newNode); err2 != nil {
			errs = append(errs, err2)
		}
	} // else no event happens
	return errors.Join(errs...)
}

func eventNoteNameAppearanceMismatch(name string, len0, len1 int) string {
	if len0 != 0 && len1 == 0 {
		return fmt.Sprintf("name %q only appears in input 0", name)
	}
	if len0 == 0 && len1 != 0 {
		return fmt.Sprintf("name %q only appears in input 1", name)
	}
	return fmt.Sprintf("name %q appears %d times in input 0, %d times in input 1",
		name, len0, len1)
}

func (d *differ) diffTarEntries(ctx context.Context, node *EventTreeNode, in [2]EventInput, ents [2][]*TarEntry) (dirsToBeRemoved []string, retErr error) {
	var (
		dirsToBeRemovedIfEmpty []string
		errs                   []error
	)
	for i, ent0 := range ents[0] {
		ent1 := ents[1][i]
		childInputs := in
		childInputs[0].TarEntry = ent0
		childInputs[1].TarEntry = ent1
		dd, err := d.diffTarEntry(ctx, node, childInputs)
		dirsToBeRemovedIfEmpty = append(dirsToBeRemovedIfEmpty, dd...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return dirsToBeRemovedIfEmpty, errors.Join(errs...)
}

func (d *differ) diffTarEntry(ctx context.Context, node *EventTreeNode, in [2]EventInput) (dirsToBeRemovedIfEmpty []string, retErr error) {
	var negligibleTarFields []string
	if d.o.IgnoreTimestamps {
		negligibleTarFields = append(negligibleTarFields, "ModTime", "AccessTime", "ChangeTime")
	}
	cmpOpts := []cmp.Option{cmpopts.IgnoreUnexported(TarEntry{}), cmpopts.IgnoreFields(tar.Header{}, negligibleTarFields...)}
	ent0, ent1 := *in[0].TarEntry, *in[1].TarEntry
	if d.o.IgnoreFileOrder {
		// cmpopts.IgnoreFields cannot be used for int
		ent0.Index = -1
		ent1.Index = -1
	}
	if d.o.IgnoreFileModeRedundantBits {
		// Ignore 0x4000 (directory), 0x8000 (regular), etc.
		// BuildKit sets these redundant bits. The legacy builder does not.
		ent0.Header.Mode &= 0x0FFF
		ent1.Header.Mode &= 0x0FFF
	}
	var errs []error
	if diff := cmp.Diff(ent0, ent1, cmpOpts...); diff != "" {
		ev := Event{
			Type:   EventTypeTarEntryMismatch,
			Inputs: in,
			Diff:   diff,
			Note:   fmt.Sprintf("name %q", ent0.Header.Name),
		}
		if err := d.raiseEvent(ctx, node, ev, "tarentry"); err != nil {
			errs = append(errs, err)
		}
	} else {
		// entry matches, so no need to retain the extracted files and dirs
		// (but dirs cannot be removed until processing all the tar entries in the layer)
		if ent0.Header.Typeflag == tar.TypeDir {
			if ent0.extractedPath != "" {
				dirsToBeRemovedIfEmpty = append(dirsToBeRemovedIfEmpty, ent0.extractedPath)
			}
			if ent1.extractedPath != "" {
				dirsToBeRemovedIfEmpty = append(dirsToBeRemovedIfEmpty, ent1.extractedPath)
			}
		} else {
			if ent0.extractedPath != "" {
				_ = os.Remove(ent0.extractedPath)
			}
			if ent1.extractedPath != "" {
				_ = os.Remove(ent1.extractedPath)
			}
		}
	}
	return dirsToBeRemovedIfEmpty, errors.Join(errs...)
}

func openTarReader(ctx context.Context, cs content.Provider, desc ocispec.Descriptor) (tr *tar.Reader, closer func() error, err error) {
	if desc.Size > int64(maxTarBlobSize) {
		return nil, nil, fmt.Errorf("too large tar blob (%d > %d bytes)", desc.Size, int64(maxTarBlobSize))
	}
	ra, err := cs.ReaderAt(ctx, desc)
	if err != nil {
		return nil, nil, err
	}
	cr := content.NewReader(ra)
	dr, err := compression.DecompressStream(cr)
	if err != nil {
		ra.Close()
		return nil, nil, err
	}
	lr := io.LimitReader(dr, maxTarStreamSize)
	return tar.NewReader(lr), ra.Close, nil
}

func readBlobWithType[T interface {
	ocispec.Index | ocispec.Manifest | ocispec.Image
}](ctx context.Context, cs content.Provider, desc ocispec.Descriptor) (*T, error) {
	if desc.Size > maxJSONBlobSize {
		return nil, fmt.Errorf("too large JSON blob (%d > %d bytes)", desc.Size, maxJSONBlobSize)
	}
	b, err := content.ReadBlob(ctx, cs, desc)
	if err != nil {
		return nil, err
	}
	var t T
	if err = json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

type EventTreeNode struct {
	Context      string `json:"context"` // Not unique
	Event        `json:"event"`
	Children     []*EventTreeNode `json:"children,omitempty"`
	sync.RWMutex `json:"-"`
}

func (n *EventTreeNode) Append(newNode *EventTreeNode) {
	n.Lock()
	n.Children = append(n.Children, newNode)
	n.Unlock()
}

type Event struct {
	Type   EventType     `json:"type,omitempty"`
	Inputs [2]EventInput `json:"inputs,omitempty"`
	Diff   string        `json:"diff,omitempty"` // Not machine-parsable
	Note   string        `json:"note,omitempty"` // Not machine-parsable
}

// String implements [fmt.Stringer].
// The returned string is not machine-parsable.
func (ev *Event) String() string {
	s := fmt.Sprintf("%q", ev.Type)
	if ev.Note != "" {
		s += " (" + ev.Note + ")"
	}
	if ev.Diff != "" {
		s += "\n" + ev.Diff
	}
	return s
}

type TarEntry struct {
	Index  int           `json:"index"`
	Header *tar.Header   `json:"header,omitempty"`
	Digest digest.Digest `json:"digest,omitempty"`

	extractedPath string `json:"-"` // path on local filesystem
}

type EventInput struct {
	Descriptor *ocispec.Descriptor `json:"descriptor,omitempty"`
	Index      *ocispec.Index      `json:"index,omitempty"`
	Manifest   *ocispec.Manifest   `json:"manifest,omitempty"`
	Config     *ocispec.Image      `json:"config,omitempty"`
	TarEntry   *TarEntry           `json:"tarEntry,omitempty"`
}

type EventType string

const (
	EventTypeNone                 = EventType("")
	EventTypeDescriptorMismatch   = EventType("DescriptorMismatch")
	EventTypeIndexBlobMismatch    = EventType("IndexBlobMismatch")
	EventTypeManifestBlobMismatch = EventType("ManifestBlobMismatch")
	EventTypeConfigBlobMismatch   = EventType("ConfigBlobMismatch")
	EventTypeLayerBlobMismatch    = EventType("LayerBlobMismatch")
	EventTypeTarEntryMismatch     = EventType("TarEntryMismatch")
)

const (
	maxManifests     = 4096
	maxLayers        = 4096
	maxJSONBlobSize  = 1024 * 1024
	maxTarBlobSize   = 1024 * 1024 * 1024 * 4
	maxTarStreamSize = 1024 * 1024 * 1024 * 32
)

// EventHandler handles an event.
// EventHandler blocks.
type EventHandler interface {
	HandleEventTreeNode(context.Context, *EventTreeNode) error
}

type Flusher interface {
	Flush() error
}

var DefaultEventHandler = NewDefaultEventHandler(os.Stdout)

func NewDefaultEventHandler(w io.Writer) EventHandler {
	tw := tabwriter.NewWriter(w, 4, 8, 4, ' ', 0)
	return &defaultEventHandler{tw: tw}
}

type defaultEventHandler struct {
	twHeaderOnce sync.Once
	tw           *tabwriter.Writer
}

func (h *defaultEventHandler) HandleEventTreeNode(ctx context.Context, node *EventTreeNode) error {
	ev := node.Event
	log.G(ctx).Debugf("Event: " + ev.String())
	// Only print leaf events to stdout
	if len(node.Children) > 0 {
		return nil
	}
	h.twHeaderOnce.Do(func() {
		fmt.Fprintln(h.tw, "TYPE\tNAME\tINPUT-0\tINPUT-1")
	})
	in0, in1 := ev.Inputs[0], ev.Inputs[1]
	d0, d1 := "?", "?"
	if ev.Note != "" {
		d0, d1 = ev.Note, ""
	}
	name := "-"
	if node.Context != "" {
		name = "ctx:" + node.Context
	}
	// TODO: colorize
	switch ev.Type {
	case EventTypeDescriptorMismatch:
		desc0, desc1 := in0.Descriptor, in1.Descriptor
		name = desc0.MediaType
		if desc0.MediaType != desc1.MediaType {
			d0, d1 = desc0.MediaType, desc1.MediaType
		} else if desc0.Digest != desc1.Digest {
			d0, d1 = desc0.Digest.String(), desc1.Digest.String()
			d0, d1 = strings.TrimPrefix(d0, "sha256:"), strings.TrimPrefix(d1, "sha256:")
		}
		fmt.Fprintln(h.tw, "Desc\t"+name+"\t"+d0+"\t"+d1)
	case EventTypeIndexBlobMismatch:
		fmt.Fprintln(h.tw, "Idx\t"+name+"\t"+d0+"\t"+d1)
	case EventTypeManifestBlobMismatch:
		fmt.Fprintln(h.tw, "Mani\t"+name+"\t"+d0+"\t"+d1)
	case EventTypeConfigBlobMismatch:
		fmt.Fprintln(h.tw, "Cfg\t"+name+"\t"+d0+"\t"+d1)
	case EventTypeLayerBlobMismatch:
		fmt.Fprintln(h.tw, "Layer\t"+name+"\t"+d0+"\t"+d1)
	case EventTypeTarEntryMismatch:
		name := "?"
		d0, d1 := "?", "?"
		ent0, ent1 := in0.TarEntry, in1.TarEntry
		if ent0 == nil {
			d0 = "missing"
		} else {
			name = ent0.Header.Name
		}
		if ent1 == nil {
			d1 = "missing"
		} else if ent0 == nil {
			name = ent1.Header.Name
		}
		if ent0 != nil && ent1 != nil {
			hdr0, hdr1 := ent0.Header, ent1.Header
			if hdr0.Name != hdr1.Name {
				d0, d1 = hdr0.Name, hdr1.Name
			} else if hdr0.Linkname != hdr1.Linkname {
				d0, d1 = "Linkname "+hdr0.Linkname, "Linkname "+hdr1.Linkname
			} else if hdr0.Mode != hdr1.Mode {
				d0, d1 = fmt.Sprintf("Mode 0x%0x", hdr0.Mode), fmt.Sprintf("Mode 0x%0x", hdr1.Mode)
			} else if hdr0.Uid != hdr1.Uid {
				d0, d1 = fmt.Sprintf("Uid %d", hdr0.Uid), fmt.Sprintf("Uid %d", hdr1.Uid)
			} else if hdr0.Gid != hdr1.Gid {
				d0, d1 = fmt.Sprintf("Gid %d", hdr0.Gid), fmt.Sprintf("Gid %d", hdr1.Gid)
			} else if hdr0.Uname != hdr1.Uname {
				d0, d1 = "Uname "+hdr0.Uname, "Uname "+hdr1.Uname
			} else if hdr0.Gname != hdr1.Gname {
				d0, d1 = "Gname "+hdr0.Gname, "Gname "+hdr1.Gname
			} else if hdr0.Devmajor != hdr1.Devmajor || hdr0.Devminor != hdr1.Devminor {
				d0, d1 = fmt.Sprintf("Dev %d:%d", hdr0.Devmajor, hdr0.Devminor), fmt.Sprintf("Dev %d:%d", hdr1.Devmajor, hdr1.Devminor)
			} else if ent0.Digest != ent1.Digest {
				d0, d1 = ent0.Digest.String(), ent1.Digest.String()
				d0, d1 = strings.TrimPrefix(d0, "sha256:"), strings.TrimPrefix(d1, "sha256:")
			} else if !hdr0.ModTime.Equal(hdr1.ModTime) {
				d0, d1 = hdr0.ModTime.String(), hdr1.ModTime.String()
			} else if !hdr0.AccessTime.Equal(hdr1.AccessTime) {
				d0, d1 = "Atime "+hdr0.AccessTime.String(), "Atime "+hdr1.AccessTime.String()
			} else if !hdr0.ChangeTime.Equal(hdr1.ChangeTime) {
				d0, d1 = "Ctime "+hdr0.ChangeTime.String(), "Ctime "+hdr1.ChangeTime.String()
			}
			// TODO: Xattrs
		}
		fmt.Fprintln(h.tw, "File\t"+name+"\t"+d0+"\t"+d1)
	default:
		log.G(ctx).Warnf("Unknown event: " + node.Event.String())
	}
	return nil
}

func (h *defaultEventHandler) Flush() error {
	return h.tw.Flush()
}

var VerboseEventHandler = newVerboseEventHandler()

func newVerboseEventHandler() EventHandler {
	return &verboseEventHandler{}
}

type verboseEventHandler struct {
}

func (h *verboseEventHandler) HandleEventTreeNode(ctx context.Context, node *EventTreeNode) error {
	fmt.Println("Event: " + node.Event.String())
	return nil
}

const (
	ReportDirReportJSON = "report.json"
	ReportDirInput0     = "input-0"
	ReportDirInput1     = "input-1"
)

var ReportDirRootFilenames = []string{
	ReportDirReportJSON,
	ReportDirInput0,
	ReportDirInput1,
}
