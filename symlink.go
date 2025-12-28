package fsx

import (
	"errors"
	"io/fs"
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

// Symlink creates a symbolic link at newname, pointing to oldname.
// It uses the provided filesystem fsys.
// If fsys implements SymlinkFS, it calls fsys.Symlink.
// Otherwise, it returns an error indicating that the operation is unsupported.
func Symlink(fsys fs.FS, oldname, newname string) error {
	if sfs, ok := fsys.(SymlinkFS); ok {
		return sfs.Symlink(oldname, newname)
	}
	return intoPathErr("symlink", newname, errors.ErrUnsupported)
}
