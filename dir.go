package fsx

import (
	"errors"
	"io/fs"
	"path"
	"syscall"

	"github.com/gwangyi/fsx/internal"
)

// DirFS is an interface for filesystems that support creating directories.
// It extends WriterFS (and thus fs.FS) and fs.ReadDirFS with the Mkdir method.
type DirFS interface {
	WriterFS
	fs.ReadDirFS

	// Mkdir creates a new directory with the specified name and permission bits.
	// If there is an error, it will be of type *PathError.
	Mkdir(name string, perm fs.FileMode) error
}

// ReadDirFile is a file that supports reading directory entries.
type ReadDirFile = fs.ReadDirFile

// MkdirAllFS is an interface for filesystems that support creating a directory
// along with any necessary parents (mkdir -p).
type MkdirAllFS interface {
	DirFS

	// MkdirAll creates a directory named path, along with any necessary parents,
	// and returns nil, or else returns an error.
	// The permission bits perm (before umask) are used for all directories that
	// MkdirAll creates. If path is already a directory, MkdirAll does nothing
	// and returns nil.
	MkdirAll(name string, perm fs.FileMode) error
}

// Mkdir creates a new directory with the specified name and permission bits
// (before umask).
//
// If fsys implements DirFS, it calls fsys.Mkdir.
// Otherwise, it returns errors.ErrUnsupported.
func Mkdir(fsys fs.FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(DirFS); ok {
		return internal.IntoPathErr("mkdir", name, fsys.Mkdir(name, perm))
	}

	return errors.ErrUnsupported
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that
// MkdirAll creates. If path is already a directory, MkdirAll does nothing
// and returns nil.
//
// If fsys implements MkdirAllFS, it calls fsys.MkdirAll.
// If the operation is not supported or not implemented, it falls back to a
// naive implementation using Mkdir and recursion.
func MkdirAll(fsys fs.FS, name string, perm fs.FileMode) error {
	// Try the optimized MkdirAllFS implementation first.
	if fsys, ok := fsys.(MkdirAllFS); ok {
		if err := fsys.MkdirAll(name, perm); !errors.Is(err, errors.ErrUnsupported) {
			return internal.IntoPathErr("mkdir", name, err)
		}
	}

	// If the path already exists, check if it is a directory.
	if info, err := fs.Stat(fsys, name); err == nil {
		if info.IsDir() {
			return nil
		}
		return &fs.PathError{Op: "mkdir", Path: name, Err: syscall.ENOTDIR}
	}

	// Create parent directories recursively.
	parent := path.Dir(name)
	if parent != "." && parent != name {
		if err := MkdirAll(fsys, parent, perm); err != nil {
			return err
		}
	}

	// Create the directory itself.
	return Mkdir(fsys, name, perm)
}
