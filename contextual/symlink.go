package contextual

import (
	"context"
	"errors"
	"io/fs"
)

// ReadLinkFS is the interface implemented by a file system that supports
// context-aware ReadLink.
type ReadLinkFS interface {
	FS
	// ReadLink returns the destination of the named symbolic link.
	// If there is an error, it should be of type *fs.PathError.
	ReadLink(ctx context.Context, name string) (string, error)

	// Lstat returns a FileInfo describing the named file.
	// If the file is a symbolic link, the returned FileInfo describes the symbolic link.
	// Lstat makes no attempt to follow the link.
	// If there is an error, it should be of type *fs.PathError.
	Lstat(ctx context.Context, name string) (fs.FileInfo, error)
}

// SymlinkFS is an interface for filesystems that support creating symbolic links.
type SymlinkFS interface {
	WriterFS
	ReadLinkFS

	// Symlink creates a symbolic link at newname, pointing to oldname.
	Symlink(ctx context.Context, oldname, newname string) error
}

// LchownFS is an interface for filesystems that support changing the ownership
// of a symbolic link itself.
type LchownFS interface {
	SymlinkFS

	// Lchown changes the ownership of the named file (or link).
	Lchown(ctx context.Context, name, owner, group string) error
}

// Symlink creates a symbolic link at newname, pointing to oldname.
func Symlink(ctx context.Context, fsys FS, oldname, newname string) error {
	if sfs, ok := fsys.(SymlinkFS); ok {
		return sfs.Symlink(ctx, oldname, newname)
	}
	return intoPathErr("symlink", newname, errors.ErrUnsupported)
}

// Lchown changes the ownership of the named file.
func Lchown(ctx context.Context, fsys FS, name, owner, group string) error {
	if cfs, ok := fsys.(LchownFS); ok {
		return cfs.Lchown(ctx, name, owner, group)
	}
	return intoPathErr("lchown", name, errors.ErrUnsupported)
}

// ReadLink returns the destination of the named symbolic link.
func ReadLink(ctx context.Context, fsys FS, name string) (string, error) {
	if rfs, ok := fsys.(ReadLinkFS); ok {
		return rfs.ReadLink(ctx, name)
	}
	return "", intoPathErr("readlink", name, errors.ErrUnsupported)
}
