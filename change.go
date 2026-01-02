package fsx

import (
	"errors"
	"io/fs"
	"time"
)

// ChangeFS is an interface that extends the basic fs.FS to support modification operations.
// It groups the methods for changing file ownership (Chown), file modes (Chmod),
// and file timestamps (Chtimes).
//
// File systems that support these operations should implement this interface
// to allow clients to modify file metadata in a unified way.
type ChangeFS interface {
	WriterFS

	// Chown changes the numeric uid and gid of the named file.
	// It is similar to os.Chown.
	//
	// Parameters:
	//   name:  The name of the file within the filesystem.
	//   owner: The string representation of the owner (e.g., username or numeric ID).
	//   group: The string representation of the group (e.g., groupname or numeric ID).
	//
	// Behavior:
	//   If the file is a symbolic link, it changes the owner and group of the link's target.
	//   If the filesystem does not support this operation, it should return an error.
	Chown(name, owner, group string) error

	// Chmod changes the file mode bits of the named file to the specified mode.
	// It is similar to os.Chmod.
	//
	// Parameters:
	//   name: The name of the file within the filesystem.
	//   mode: The new file mode (permissions).
	//
	// Behavior:
	//   If the file is a symbolic link, it changes the mode of the link's target.
	//   If the filesystem does not support this operation, it should return an error.
	Chmod(name string, mode fs.FileMode) error

	// Chtimes changes the access and modification times of the named file.
	// It is similar to os.Chtimes.
	//
	// Parameters:
	//   name:  The name of the file within the filesystem.
	//   atime: The new access time.
	//   ctime: The new modification time (note: often referred to as mtime in other contexts).
	//
	// Behavior:
	//   The underlying filesystem may truncate or round the values to a less precise time unit.
	//   If the filesystem does not support this operation, it should return an error.
	Chtimes(name string, atime, ctime time.Time) error
}

// Chown changes the owner and group of the named file.
//
// It checks if the provided filesystem (fs.FS) implements the ChangeFS interface.
// If it does, it calls the interface's Chown method.
//
// Parameters:
//
//	fs:    The filesystem interface.
//	name:  The path of the file.
//	owner: The new owner.
//	group: The new group.
//
// Returns:
//
//	error: nil on success, or an error if the operation fails or is unsupported.
//	       Returns errors.ErrUnsupported if fs does not implement ChangeFS.
func Chown(fs fs.FS, name, owner, group string) error {
	if cfs, ok := fs.(ChangeFS); ok {
		return cfs.Chown(name, owner, group)
	}

	return errors.ErrUnsupported
}

// Chmod changes the mode of the named file to mode.
//
// It checks if the provided filesystem (fs.FS) implements the ChangeFS interface.
// If it does, it calls the interface's Chmod method.
//
// Parameters:
//
//	fs:   The filesystem interface.
//	name: The path of the file.
//	mode: The new file mode bits.
//
// Returns:
//
//	error: nil on success, or an error if the operation fails or is unsupported.
//	       Returns errors.ErrUnsupported if fs does not implement ChangeFS.
func Chmod(fs fs.FS, name string, mode fs.FileMode) error {
	if cfs, ok := fs.(ChangeFS); ok {
		return cfs.Chmod(name, mode)
	}

	return errors.ErrUnsupported
}

// Chtimes changes the access and modification times of the named file.
//
// It checks if the provided filesystem (fs.FS) implements the ChangeFS interface.
// If it does, it calls the interface's Chtimes method.
//
// Parameters:
//
//	fs:    The filesystem interface.
//	name:  The path of the file.
//	atime: The new access time.
//	mtime: The new modification time.
//
// Returns:
//
//	error: nil on success, or an error if the operation fails or is unsupported.
//	       Returns errors.ErrUnsupported if fs does not implement ChangeFS.
func Chtimes(fs fs.FS, name string, atime, mtime time.Time) error {
	if cfs, ok := fs.(ChangeFS); ok {
		return cfs.Chtimes(name, atime, mtime)
	}

	return errors.ErrUnsupported
}
