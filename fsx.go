// Package fsx provides extended filesystem interfaces that support write operations.
// It builds upon the standard library's io/fs package, adding support for creating,
// writing, and removing files.
//
// The standard io/fs package defines a read-only filesystem interface. fsx extends
// this to support common write operations, making it suitable for virtual filesystems,
// overlays, and testing mocks where write capabilities are required.
package fsx

//go:generate mockgen -destination mockfs/mockfs.go -package mockfs . FS,DirEntry,File,FileInfo,ChangeFS,DirFS,LchownFS,MkdirAllFS,RemoveAllFS,RenameFS,SymlinkFS,TruncateFS,WriteFileFS

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

const (
	// O_ACCMODE is the mask for access modes (O_RDONLY, O_WRONLY, O_RDWR).
	// It is used to mask the flag argument in OpenFile to extract the access mode.
	O_ACCMODE = 3
)

// File is an open file that supports reading, writing, and truncation.
// It extends fs.File (which only supports read-related operations) with
// io.Writer and a Truncate method.
//
// Implementations of this interface allow users to modify the file content
// after opening it.
type File interface {
	fs.File
	io.Writer
	// Truncate changes the size of the file.
	// It returns an error if the file was not opened with write permissions.
	Truncate(size int64) error
}

// FileInfo is a type alias for fs.FileInfo, allowing it to be mocked by mockgen.
type FileInfo = fs.FileInfo

// DirEntry is a type alias for fs.DirEntry, allowing it to be mocked by mockgen.
type DirEntry = fs.DirEntry

// readOnlyFile wraps an fs.File to implement the fsx.File interface,
// explicitly returning errors for any write-related operations.
//
// This struct is used when an underlying fs.FS only supports Open (read-only)
// but the context requires an fsx.File interface. It ensures that calls to
// Write or Truncate fail gracefully with a specific error.
type readOnlyFile struct {
	fs.File
}

// Write returns ErrBadFileDescriptor as readOnlyFile does not support writing.
// This enforces the read-only nature of the wrapped file.
func (readOnlyFile) Write(d []byte) (int, error) {
	return 0, ErrBadFileDescriptor
}

// Truncate returns ErrBadFileDescriptor as readOnlyFile does not support truncation.
// This enforces the read-only nature of the wrapped file.
func (readOnlyFile) Truncate(size int64) error {
	return ErrBadFileDescriptor
}

// FS is a filesystem interface that extends fs.FS to support creating, opening with flags,
// and removing files.
//
// While fs.FS is read-only, fsx.FS adds the necessary methods to modify the filesystem structure
// and file contents.
type FS interface {
	fs.FS

	// Create creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created with mode 0666
	// (before umask). If successful, methods on the returned File can
	// be used for I/O; the associated file descriptor has mode O_RDWR.
	// If there is an error, it will be of type *PathError.
	Create(name string) (File, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// (O_RDONLY etc.) and perm (0666 etc.) if applicable. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *PathError.
	OpenFile(name string, flag int, mode fs.FileMode) (File, error)

	// Remove removes the named file or (empty) directory.
	// If there is an error, it will be of type *PathError.
	Remove(name string) error
}

// Create creates or truncates the named file in the given filesystem.
// It acts as a helper function that checks if the filesystem implements fsx.FS.
//
// If fsys implements fsx.FS, it calls fsys.Create.
// If fsys does not implement fsx.FS, it returns errors.ErrUnsupported.
func Create(fsys fs.FS, name string) (File, error) {
	if xfs, ok := fsys.(FS); ok {
		f, err := xfs.Create(name)
		return f, intoPathErr("open", name, err)
	}

	return nil, errors.ErrUnsupported
}

// OpenFile opens the named file with specified flag and mode in the given filesystem.
// It provides a generalized open call similar to os.OpenFile.
//
// If fsys implements fsx.FS, it calls fsys.OpenFile.
// If the operation is not supported by the filesystem implementation (returns ErrUnsupported)
// or if fsys is not an fsx.FS, it attempts a fallback for read-only access:
// if the flag requests read-only access (O_RDONLY), it falls back to fsys.Open.
// Otherwise, it returns errors.ErrUnsupported.
func OpenFile(fsys fs.FS, name string, flag int, mode fs.FileMode) (File, error) {
	if xfs, ok := fsys.(FS); ok {
		// Try the specific OpenFile implementation first.
		if f, err := xfs.OpenFile(name, flag, mode); !errors.Is(err, errors.ErrUnsupported) {
			return f, intoPathErr("open", name, err)
		}
	}

	// Fallback for read-only access if OpenFile is not supported or not implemented.
	if flag&O_ACCMODE == os.O_RDONLY {
		f, err := fsys.Open(name)
		if err != nil {
			return nil, intoPathErr("open", name, err)
		}
		// Wrap the standard fs.File in a readOnlyFile to satisfy the fsx.File interface.
		return readOnlyFile{File: f}, nil
	}

	return nil, errors.ErrUnsupported
}

// Remove removes the named file or (empty) directory from the filesystem.
//
// If fsys implements fsx.FS, it calls fsys.Remove.
// Otherwise, it returns errors.ErrUnsupported.
func Remove(fsys fs.FS, name string) error {
	if xfs, ok := fsys.(FS); ok {
		return intoPathErr("remove", name, xfs.Remove(name))
	}

	return errors.ErrUnsupported
}
