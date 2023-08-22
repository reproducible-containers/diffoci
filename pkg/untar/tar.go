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
	"strings"
	"sync"
	"syscall"

	"github.com/containerd/containerd/log"
	"github.com/containerd/continuity/fs"
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

	digester := digest.SHA256.Digester()
	hasher := digester.Hash()
	teeR := io.TeeReader(r, hasher)
	if err := createTarFile(ctx, path, root, hdr, teeR); err != nil {
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

// bufPool is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L39-L44 .
var bufPool = &sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 32*1024)
		return &buffer
	},
}

const (
	// paxSchilyXattr is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L133 .
	paxSchilyXattr = "SCHILY.xattr."
)

// createTarFile is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L329-L432 .
// Modified to allow permission errors.
//
// TODO: propagate the allowed permission errors to [EntryResult].
func createTarFile(ctx context.Context, path, extractDir string, hdr *tar.Header, reader io.Reader) error {
	// hdr.Mode is in linux format, which we can use for syscalls,
	// but for os.Foo() calls we need the mode converted to os.FileMode,
	// so use hdrInfo.Mode() (they differ for e.g. setuid bits)
	hdrInfo := hdr.FileInfo()

	switch hdr.Typeflag {
	case tar.TypeDir:
		// Create directory unless it exists as a directory already.
		// In that case we just want to merge the two
		if fi, err := os.Lstat(path); !(err == nil && fi.IsDir()) {
			if err := mkdir(path, hdrInfo.Mode()); err != nil {
				return err
			}
		}

	//nolint:staticcheck // TypeRegA is deprecated but we may still receive an external tar with TypeRegA
	case tar.TypeReg, tar.TypeRegA:
		file, err := openFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdrInfo.Mode())
		if err != nil {
			return err
		}

		_, err = copyBuffered(ctx, file, reader)
		if err1 := file.Close(); err == nil {
			err = err1
		}
		if err != nil {
			return err
		}

	case tar.TypeBlock, tar.TypeChar:
		// Handle this is an OS-specific way
		if err := handleTarTypeBlockCharFifo(hdr, path); err != nil {
			return err
		}

	case tar.TypeFifo:
		// Handle this is an OS-specific way
		if err := handleTarTypeBlockCharFifo(hdr, path); err != nil {
			return err
		}

	case tar.TypeLink:
		targetPath, err := hardlinkRootPath(extractDir, hdr.Linkname)
		if err != nil {
			return err
		}

		if err := link(targetPath, path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}

	case tar.TypeXGlobalHeader:
		log.G(ctx).Debug("PAX Global Extended Headers found and ignored")
		return nil

	default:
		return fmt.Errorf("unhandled tar header type %d", hdr.Typeflag)
	}

	if err := os.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
		log.G(ctx).WithError(err).Debugf("failed to Lchown %q for UID %d, GID %d", path, hdr.Uid, hdr.Gid)
	}

	for key, value := range hdr.PAXRecords {
		if strings.HasPrefix(key, paxSchilyXattr) {
			key = key[len(paxSchilyXattr):]
			if err := setxattr(path, key, value); err != nil {
				log.G(ctx).WithError(err).Debugf("failed to setxattr %q for key %q", path, key)
			}
		}
	}

	// call lchmod after lchown since lchown can modify the file mode
	if err := lchmod(path, hdrInfo.Mode()); err != nil {
		return err
	}

	return chtimes(path, boundTime(latestTime(hdr.AccessTime, hdr.ModTime)), boundTime(hdr.ModTime))
}

// mkparent is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L434-L492 .
func mkparent(ctx context.Context, path, root string, parents []string) error {
	if dir, err := os.Lstat(path); err == nil {
		if dir.IsDir() {
			return nil
		}
		return &os.PathError{
			Op:   "mkparent",
			Path: path,
			Err:  syscall.ENOTDIR,
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	i := len(path)
	for i > len(root) && !os.IsPathSeparator(path[i-1]) {
		i--
	}

	if i > len(root)+1 {
		if err := mkparent(ctx, path[:i-1], root, parents); err != nil {
			return err
		}
	}

	if err := mkdir(path, 0755); err != nil {
		// Check that still doesn't exist
		dir, err1 := os.Lstat(path)
		if err1 == nil && dir.IsDir() {
			return nil
		}
		return err
	}

	for _, p := range parents {
		ppath, err := fs.RootPath(p, path[len(root):])
		if err != nil {
			return err
		}

		dir, err := os.Lstat(ppath)
		if err == nil {
			if !dir.IsDir() {
				// Replaced, do not copy attributes
				break
			}
			if err := copyDirInfo(dir, path); err != nil {
				return err
			}
			return copyUpXAttrs(path, ppath)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	log.G(ctx).Debugf("parent directory %q not found: default permissions(0755) used", path)

	return nil
}

// copyBuffered is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L739-L775 .
func copyBuffered(ctx context.Context, dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufPool.Get().(*[]byte)
	defer bufPool.Put(buf)

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		nr, er := src.Read(*buf)
		if nr > 0 {
			nw, ew := dst.Write((*buf)[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err

}

// hardlinkRootPath is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar.go#L777-L799 .
// hardlinkRootPath returns target linkname, evaluating and bounding any
// symlink to the parent directory.
//
// NOTE: Allow hardlink to the softlink, not the real one. For example,
//
//	touch /tmp/zzz
//	ln -s /tmp/zzz /tmp/xxx
//	ln /tmp/xxx /tmp/yyy
//
// /tmp/yyy should be softlink which be same of /tmp/xxx, not /tmp/zzz.
func hardlinkRootPath(root, linkname string) (string, error) {
	ppath, base := filepath.Split(linkname)
	ppath, err := fs.RootPath(root, ppath)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(ppath, base)
	if !strings.HasPrefix(targetPath, root) {
		targetPath = root
	}
	return targetPath, nil
}
