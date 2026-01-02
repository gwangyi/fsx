package fsx

import (
	"errors"
	"io/fs"

	"github.com/gwangyi/fsx/internal"
)

// SymlinkFS is an interface for filesystems that support creating symbolic links.
// It extends fs.ReadLinkFS to include the Symlink method.
type SymlinkFS interface {
	FS
	fs.ReadLinkFS

	// Symlink creates a symbolic link at newname, pointing to oldname.
	// If newname already exists, Symlink should return an error.
	Symlink(oldname, newname string) error
}

// LchownFS is an interface for filesystems that support changing the ownership
// of a symbolic link itself, rather than the file it points to.
type LchownFS interface {
	SymlinkFS

	// Lchown changes the ownership of the named file.
	// If the file is a symbolic link, it changes the ownership of the link itself.
	// It accepts owner and group as strings, which implies that the underlying
	// filesystem may support user/group names or numeric IDs.
	// If the file does not exist, it returns an error.
	Lchown(name, owner, group string) error
}

// Symlink creates a symbolic link at newname, pointing to oldname.
// It uses the provided filesystem fsys.
// If fsys implements SymlinkFS, it calls fsys.Symlink.
// Otherwise, it returns an error indicating that the operation is unsupported.
func Symlink(fsys fs.FS, oldname, newname string) error {
	if sfs, ok := fsys.(SymlinkFS); ok {
		return sfs.Symlink(oldname, newname)
	}
	return internal.IntoPathErr("symlink", newname, errors.ErrUnsupported)
}

// Lchown changes the ownership of the named file.
// If the file is a symbolic link, it changes the ownership of the link itself.
// It uses the provided filesystem fsys.
// If fsys implements LchownFS, it calls fsys.Lchown.
// Otherwise, it returns an error indicating that the operation is unsupported.
func Lchown(fsys fs.FS, name, owner, group string) error {
	if cfs, ok := fsys.(LchownFS); ok {
		return cfs.Lchown(name, owner, group)
	}
	return internal.IntoPathErr("lchown", name, errors.ErrUnsupported)
}
