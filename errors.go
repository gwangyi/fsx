package fsx

import (
	"github.com/gwangyi/fsx/internal"
)

var (
	// ErrBadFileDescriptor is returned when an operation is performed on a file descriptor
	// that is not open for that operation (e.g., writing to a read-only file).
	ErrBadFileDescriptor = internal.ErrBadFileDescriptor

	// ErrNotDir is returned when a directory operation is requested on a non-directory file.
	ErrNotDir = internal.ErrNotDir

	// ErrIsDir is returned when a file operation is requested on a directory.
	ErrIsDir = internal.ErrIsDir
)

// IsInvalid checks if the provided error represents an invalid operation or path.
func IsInvalid(err error) bool {
	return internal.IsInvalid(err)
}

// IsUnsupported checks if the provided error indicates that an operation is not supported.
func IsUnsupported(err error) bool {
	return internal.IsUnsupported(err)
}
