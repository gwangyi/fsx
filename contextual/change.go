package contextual

import (
	"context"
	"errors"
	"io/fs"
	"time"
)

// ChangeFS is an interface that extends WriterFS to support modification operations
// like Chown, Chmod, and Chtimes.
type ChangeFS interface {
	WriterFS

	// Chown changes the numeric uid and gid of the named file.
	Chown(ctx context.Context, name, owner, group string) error

	// Chmod changes the file mode bits of the named file to the specified mode.
	Chmod(ctx context.Context, name string, mode fs.FileMode) error

	// Chtimes changes the access and modification times of the named file.
	Chtimes(ctx context.Context, name string, atime, ctime time.Time) error
}

// Chown changes the owner and group of the named file.
func Chown(ctx context.Context, fsys FS, name, owner, group string) error {
	if cfs, ok := fsys.(ChangeFS); ok {
		return cfs.Chown(ctx, name, owner, group)
	}

	return errors.ErrUnsupported
}

// Chmod changes the mode of the named file to mode.
func Chmod(ctx context.Context, fsys FS, name string, mode fs.FileMode) error {
	if cfs, ok := fsys.(ChangeFS); ok {
		return cfs.Chmod(ctx, name, mode)
	}

	return errors.ErrUnsupported
}

// Chtimes changes the access and modification times of the named file.
func Chtimes(ctx context.Context, fsys FS, name string, atime, mtime time.Time) error {
	if cfs, ok := fsys.(ChangeFS); ok {
		return cfs.Chtimes(ctx, name, atime, mtime)
	}

	return errors.ErrUnsupported
}
