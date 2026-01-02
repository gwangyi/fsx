package contextual

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"sort"
	"syscall"
)

// ReadDirFS is the interface implemented by a file system that supports
// context-aware ReadDir.
type ReadDirFS interface {
	FS
	// ReadDir reads the named directory
	// and returns a list of directory entries sorted by filename.
	ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error)
}

// ReadDirFile is a file that supports reading directory entries.
type ReadDirFile = fs.ReadDirFile

// DirFS is an interface for filesystems that support creating directories.
type DirFS interface {
	WriterFS
	ReadDirFS

	// Mkdir creates a new directory with the specified name and permission bits.
	Mkdir(ctx context.Context, name string, perm fs.FileMode) error
}

// MkdirAllFS is an interface for filesystems that support creating a directory
// along with any necessary parents.
type MkdirAllFS interface {
	DirFS

	// MkdirAll creates a directory named path, along with any necessary parents.
	MkdirAll(ctx context.Context, name string, perm fs.FileMode) error
}

// Mkdir creates a new directory with the specified name and permission bits.
func Mkdir(ctx context.Context, fsys FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(DirFS); ok {
		return intoPathErr("mkdir", name, fsys.Mkdir(ctx, name, perm))
	}

	return errors.ErrUnsupported
}

// MkdirAll creates a directory named path, along with any necessary parents.
func MkdirAll(ctx context.Context, fsys FS, name string, perm fs.FileMode) error {
	if fsys, ok := fsys.(MkdirAllFS); ok {
		if err := fsys.MkdirAll(ctx, name, perm); !errors.Is(err, errors.ErrUnsupported) {
			return intoPathErr("mkdir", name, err)
		}
	}

	if info, err := Stat(ctx, fsys, name); err == nil {
		if info.IsDir() {
			return nil
		}
		return &fs.PathError{Op: "mkdir", Path: name, Err: syscall.ENOTDIR}
	}

	parent := path.Dir(name)
	if parent != "." && parent != name {
		if err := MkdirAll(ctx, fsys, parent, perm); err != nil {
			return err
		}
	}

	return Mkdir(ctx, fsys, name, perm)
}

// ReadDir reads the named directory and returns a list of directory entries sorted by filename.
func ReadDir(ctx context.Context, fsys FS, name string) ([]fs.DirEntry, error) {
	if fsys, ok := fsys.(ReadDirFS); ok {
		return fsys.ReadDir(ctx, name)
	}

	file, err := fsys.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	dir, ok := file.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: errors.ErrUnsupported}
	}

	list, err := dir.ReadDir(-1)
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, err
}
