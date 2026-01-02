package fsx

import (
	"errors"
	"io/fs"
	"os"

	"github.com/gwangyi/fsx/internal"
)

// WriteFileFS is the interface implemented by a filesystem that provides
// an optimized WriteFile method.
type WriteFileFS interface {
	WriterFS
	// WriteFile writes data to the named file, creating it if necessary.
	// It is similar to os.WriteFile.
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

// WriteFile writes data to the named file, creating it if necessary.
// If the file does not exist, WriteFile creates it with permissions perm (before umask);
// otherwise WriteFile truncates it before writing, without changing permissions.
//
// If the filesystem implements WriteFileFS, its WriteFile method is used.
// Otherwise, it falls back to using OpenFile (with O_WRONLY|O_CREATE|O_TRUNC)
// and writing the data to the file.
// If the filesystem does not implement fsx.WriterFS, it returns errors.ErrUnsupported.
func WriteFile(fsys fs.FS, name string, data []byte, perm fs.FileMode) error {
	// Try optimized WriteFileFS first
	if fsysImpl, ok := fsys.(WriteFileFS); ok {
		if err := fsysImpl.WriteFile(name, data, perm); !errors.Is(err, errors.ErrUnsupported) {
			return internal.IntoPathErr("writefile", name, err)
		}
	}

	// Fallback to OpenFile -> Write
	file, err := OpenFile(fsys, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return internal.IntoPathErr("writefile", name, err)
	}
	defer func() { _ = file.Close() }()
	_, err = file.Write(data)
	return internal.IntoPathErr("writefile", name, err)
}
