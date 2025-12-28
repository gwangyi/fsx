package fsx

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

var (
	// ErrBadFileDescriptor is returned when an operation is performed on a file descriptor
	// that is not open for that operation (e.g., writing to a read-only file).
	// It is an alias for syscall.EBADF.
	ErrBadFileDescriptor = syscall.EBADF

	// ErrNotDir is returned when a directory operation is requested on a non-directory file.
	// It is an alias for syscall.ENOTDIR.
	ErrNotDir = syscall.ENOTDIR

	// ErrIsDir is returned when a file operation is requested on a directory.
	// It is an alias for syscall.EISDIR.
	ErrIsDir = syscall.EISDIR
)

func underlyingError(err error) error {
	switch e := err.(type) {
	case *fs.PathError:
		return e.Err
	case *os.LinkError:
		return e.Err
	case *os.SyscallError:
		return e.Err
	}
	return err
}

// intoPathErr wraps the error into an fs.PathError if it's not already one,
// using the provided operation and path.
//
// This helper ensures consistent error reporting across the library, making it
// easier for callers to inspect errors (e.g., checking Op and Path).
func intoPathErr(op, path string, err error) error {
	if err == nil {
		return nil
	}

	return &fs.PathError{Op: op, Path: path, Err: underlyingError(err)}
}

// intoLinkErr wraps the error into an os.LinkError if it's not already one,
// using the provided operation and path.
//
// This helper ensures consistent error reporting across the library, making it
// easier for callers to inspect errors (e.g., checking Op and Path).
func intoLinkErr(op, oldpath, newpath string, err error) error {
	if err == nil {
		return nil
	}

	return &os.LinkError{Op: op, Old: oldpath, New: newpath, Err: underlyingError(err)}
}

// IsInvalid checks if the provided error represents an invalid operation or path.
// It returns true if the error is:
// - fs.ErrInvalid
// - ErrNotDir
// - ErrIsDir
func IsInvalid(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, fs.ErrInvalid):
		return true
	case errors.Is(err, ErrNotDir):
		return true
	case errors.Is(err, ErrIsDir):
		return true
	}

	return false
}

// IsUnsupported checks if the provided error indicates that an operation is not supported.
// It returns true if the error is:
// - errors.ErrUnsupported
// - An error containing the string "not implemented"
func IsUnsupported(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, errors.ErrUnsupported):
		return true
	}

	// Some implementations might return a generic error with "not implemented" string
	// instead of a specific typed error.
	if strings.Contains(err.Error(), "not implemented") {
		return true
	}

	return false
}
