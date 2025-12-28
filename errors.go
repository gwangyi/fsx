package fsx

import (
	"io/fs"
	"os"
	"syscall"
)

var (
	// ErrBadFileDescriptor is returned when an operation is performed on a file descriptor
	// that is not open for that operation (e.g., writing to a read-only file).
	// It is an alias for syscall.EBADF.
	ErrBadFileDescriptor = syscall.EBADF
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
