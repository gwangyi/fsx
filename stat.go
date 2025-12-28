package fsx

import (
	"io/fs"
)

// Stat returns a FileInfo describing the named file.
// It wraps the standard fs.Stat function but returns an fsx.FileInfo,
// which provides extended file attributes (like owner, group, inode, etc.)
// if available from the underlying system.
//
// If the filesystem does not support extended attributes, default or zero values
// are returned for the extended fields.
//
// Parameters:
//
//	fsys: The filesystem interface.
//	name: The path of the file.
//
// Returns:
//
//	FileInfo: An extended FileInfo interface containing standard and extended attributes.
//	error:    nil on success, or an error if the operation fails.
func Stat(fsys fs.FS, name string) (FileInfo, error) {
	fi, err := fs.Stat(fsys, name)
	return ExtendFileInfo(fi), err
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo describes the symbolic link.
// Lstat makes no attempt to follow the link.
//
// It wraps the standard fs.Lstat function (introduced in Go 1.25) but returns
// an fsx.FileInfo, which provides extended file attributes (like owner, group, inode, etc.)
// if available from the underlying system.
//
// Parameters:
//
//	fsys: The filesystem interface.
//	name: The path of the file.
//
// Returns:
//
//	FileInfo: An extended FileInfo interface containing standard and extended attributes.
//	error:    nil on success, or an error if the operation fails.
func Lstat(fsys fs.FS, name string) (FileInfo, error) {
	fi, err := fs.Lstat(fsys, name)
	return ExtendFileInfo(fi), err
}
