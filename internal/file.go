package internal

import (
	"io"
	"io/fs"
)

const (
	// O_ACCMODE is the mask for access modes (O_RDONLY, O_WRONLY, O_RDWR).
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

// ReadOnlyFile wraps an fs.File to implement the File interface,
// explicitly returning errors for any write-related operations.
type ReadOnlyFile struct {
	fs.File
}

// Write returns ErrBadFileDescriptor as ReadOnlyFile does not support writing.
func (ReadOnlyFile) Write(d []byte) (int, error) {
	return 0, ErrBadFileDescriptor
}

// Truncate returns ErrBadFileDescriptor as ReadOnlyFile does not support truncation.
func (ReadOnlyFile) Truncate(size int64) error {
	return ErrBadFileDescriptor
}
