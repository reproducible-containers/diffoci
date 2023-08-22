//go:build !windows

// From https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go .
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
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/containerd/continuity/fs"
	"github.com/containerd/continuity/sysx"
	"golang.org/x/sys/unix"
)

// openFile is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L76-L87 .
func openFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	// Call chmod to avoid permission mask
	if err := os.Chmod(name, perm); err != nil {
		f.Close()
		return nil, err
	}
	return f, err
}

// mkdir is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L89-L96 .
func mkdir(path string, perm os.FileMode) error {
	if err := os.Mkdir(path, perm); err != nil {
		return err
	}
	// Only final created directory gets explicit permission
	// call to avoid permission mask
	return os.Chmod(path, perm)
}

// handleTarTypeBlockCharFifois is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L108-L124 .
//
// handleTarTypeBlockCharFifo is an OS-specific helper function used by
// createTarFile to handle the following types of header: Block; Char; Fifo.
// This function must not be called for Block and Char when running in userns.
// (skipFile() should return true for them.)
func handleTarTypeBlockCharFifo(hdr *tar.Header, path string) error {
	mode := uint32(hdr.Mode & 07777)
	switch hdr.Typeflag {
	case tar.TypeBlock:
		mode |= unix.S_IFBLK
	case tar.TypeChar:
		mode |= unix.S_IFCHR
	case tar.TypeFifo:
		mode |= unix.S_IFIFO
	}

	return mknod(path, mode, unix.Mkdev(uint32(hdr.Devmajor), uint32(hdr.Devminor)))
}

// setxattr is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L134-L40 .
func setxattr(path, key, value string) error {
	// Do not set trusted attributes
	if strings.HasPrefix(key, "trusted.") {
		return fmt.Errorf("admin attributes from archive not supported: %w", unix.ENOTSUP)
	}
	return unix.Lsetxattr(path, key, []byte(value), 0)
}

// cpyDirInfo is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L142-L174 .
func copyDirInfo(fi os.FileInfo, path string) error {
	st := fi.Sys().(*syscall.Stat_t)
	if err := os.Lchown(path, int(st.Uid), int(st.Gid)); err != nil {
		if os.IsPermission(err) {
			// Normally if uid/gid are the same this would be a no-op, but some
			// filesystems may still return EPERM... for instance NFS does this.
			// In such a case, this is not an error.
			if dstStat, err2 := os.Lstat(path); err2 == nil {
				st2 := dstStat.Sys().(*syscall.Stat_t)
				if st.Uid == st2.Uid && st.Gid == st2.Gid {
					err = nil
				}
			}
		}
		if err != nil {
			return fmt.Errorf("failed to chown %s: %w", path, err)
		}
	}

	if err := os.Chmod(path, fi.Mode()); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", path, err)
	}

	timespec := []unix.Timespec{
		unix.NsecToTimespec(syscall.TimespecToNsec(fs.StatAtime(st))),
		unix.NsecToTimespec(syscall.TimespecToNsec(fs.StatMtime(st))),
	}
	if err := unix.UtimesNanoAt(unix.AT_FDCWD, path, timespec, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return fmt.Errorf("failed to utime %s: %w", path, err)
	}

	return nil
}

// copyUpXAttrs is from https://github.com/containerd/containerd/blob/v1.7.3/archive/tar_unix.go#L176-L202 .
func copyUpXAttrs(dst, src string) error {
	xattrKeys, err := sysx.LListxattr(src)
	if err != nil {
		if err == unix.ENOTSUP || err == sysx.ENODATA {
			return nil
		}
		return fmt.Errorf("failed to list xattrs on %s: %w", src, err)
	}
	for _, xattr := range xattrKeys {
		// Do not copy up trusted attributes
		if strings.HasPrefix(xattr, "trusted.") {
			continue
		}
		data, err := sysx.LGetxattr(src, xattr)
		if err != nil {
			if err == unix.ENOTSUP || err == sysx.ENODATA {
				continue
			}
			return fmt.Errorf("failed to get xattr %q on %s: %w", xattr, src, err)
		}
		if err := lsetxattrCreate(dst, xattr, data); err != nil {
			return fmt.Errorf("failed to set xattr %q on %s: %w", xattr, dst, err)
		}
	}

	return nil
}
