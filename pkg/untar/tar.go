// Forked from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go .
// This fork ignores permission errors.
/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package untar

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	_ "unsafe"

	"github.com/containerd/continuity/fs"
	"github.com/containerd/log"
	"github.com/opencontainers/go-digest"
)

type EntryResult struct {
	Path      string // path on filesystem
	Digest    digest.Digest
	Finalizer func() error
}

// Entry untars a tar entry.
//
// Entry contains a portion from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L159-L327 .
func Entry(ctx context.Context, root string, hdr *tar.Header, r io.Reader) (*EntryResult, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	// Normalize name, for safety and for a simple is-root check
	hdr.Name = filepath.Clean(hdr.Name)

	// Split name and resolve symlinks for root directory.
	ppath, base := filepath.Split(hdr.Name)
	var err error
	ppath, err = fs.RootPath(root, ppath)
	if err != nil {
		return nil, fmt.Errorf("failed to get root path: %w", err)
	}

	// Join to root before joining to parent path to ensure relative links are
	// already resolved based on the root before adding to parent.
	path := filepath.Join(ppath, filepath.Join("/", base))
	if path == root {
		log.G(ctx).Debugf("file %q ignored: resolved to root", hdr.Name)
		return &EntryResult{
			Path: path,
		}, nil
	}

	// If file is not directly under root, ensure parent directory
	// exists or is created.
	if ppath != root {
		parentPath := ppath
		if base == "" {
			parentPath = filepath.Dir(path)
		}
		if err := mkparent(ctx, parentPath, root, nil); err != nil {
			return nil, err
		}
	}

	// If path exits we almost always just want to remove and replace it.
	// The only exception is when it is a directory *and* the file from
	// the layer is also a directory. Then we want to merge them (i.e.
	// just apply the metadata from the layer).
	if fi, err := os.Lstat(path); err == nil {
		if !(fi.IsDir() && hdr.Typeflag == tar.TypeDir) {
			if err := os.RemoveAll(path); err != nil {
				return nil, err
			}
		}
	}

	digester := digest.SHA256.Digester()
	hasher := digester.Hash()
	teeR := io.TeeReader(r, hasher)
	if err := createTarFile(ctx, path, root, hdr, teeR, true); err != nil {
		return nil, err
	}

	res := &EntryResult{
		Path:   path,
		Digest: digester.Digest(),
	}
	// Directory mtimes must be handled at the end to avoid further
	// file creation in them to modify the directory mtime
	if hdr.Typeflag == tar.TypeDir {
		res.Finalizer = func() error {
			return chtimes(path, boundTime(latestTime(hdr.AccessTime, hdr.ModTime)), boundTime(hdr.ModTime))
		}
	}
	return res, nil
}

//go:linkname createTarFile github.com/containerd/containerd/archive.createTarFile
func createTarFile(ctx context.Context, path, extractDir string, hdr *tar.Header, reader io.Reader, noSameOwner bool) error

//go:linkname chtimes github.com/containerd/containerd/archive.chtimes
func chtimes(path string, atime, mtime time.Time) error

//go:linkname mkparent github.com/containerd/containerd/archive.mkparent
func mkparent(ctx context.Context, path, root string, parents []string) error

//go:linkname boundTime github.com/containerd/containerd/archive.boundTime
func boundTime(t time.Time) time.Time

//go:linkname latestTime github.com/containerd/containerd/archive.latestTime
func latestTime(t1, t2 time.Time) time.Time
