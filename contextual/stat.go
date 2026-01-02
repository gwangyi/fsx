package contextual

import (
	"context"
	"io/fs"
)

// StatFS is the interface implemented by a file system that supports
// context-aware Stat.
type StatFS interface {
	FS
	// Stat returns a FileInfo describing the file.
	// If there is an error, it should be of type *fs.PathError.
	Stat(ctx context.Context, name string) (fs.FileInfo, error)
}

// Stat returns a FileInfo describing the named file.
func Stat(ctx context.Context, fsys FS, name string) (FileInfo, error) {
	if fsys, ok := fsys.(StatFS); ok {
		fi, err := fsys.Stat(ctx, name)
		return ExtendFileInfo(fi), err
	}

	f, err := fsys.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	return ExtendFileInfo(fi), err
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo describes the symbolic link.
func Lstat(ctx context.Context, fsys FS, name string) (FileInfo, error) {
	if fsys, ok := fsys.(ReadLinkFS); ok {
		fi, err := fsys.Lstat(ctx, name)
		return ExtendFileInfo(fi), err
	}
	// Fallback to Stat if Lstat is not explicitly supported.
	return Stat(ctx, fsys, name)
}
