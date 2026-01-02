package fsx

import (
	"io/fs"
	"os"
	"path"

	"github.com/gwangyi/fsx/internal"
)

// RemoveAllFS is the interface implemented by a file system that supports
// an optimized RemoveAll method.
//
// Implementations of this interface can provide a more efficient way to remove
// a directory tree compared to the default recursive implementation.
type RemoveAllFS interface {
	WriterFS

	// RemoveAll removes path and any children it contains.
	// It removes everything it can but returns the first error
	// it encounters. If the path does not exist, RemoveAll returns nil.
	RemoveAll(name string) error
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll returns nil.
//
// If fsys implements RemoveAllFS, it calls fsys.RemoveAll(name).
// Otherwise, it falls back to a recursive implementation using ReadDir and Remove.
func RemoveAll(fsys fs.FS, name string) error {
	// Try the optimized RemoveAllFS implementation first.
	if rfs, ok := fsys.(RemoveAllFS); ok {
		return internal.IntoPathErr("remove", name, rfs.RemoveAll(name))
	}

	// Attempt to remove the path directly.
	// If it's a file or an empty directory, this will succeed.
	err := Remove(fsys, name)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	// If Remove failed (likely because it's a non-empty directory),
	// iterate over the children and remove them recursively.
	entries, readErr := fs.ReadDir(fsys, name)
	if readErr != nil {
		return err // Return the original Remove error if we can't read the directory.
	}

	for _, entry := range entries {
		childPath := path.Join(name, entry.Name())
		if err := RemoveAll(fsys, childPath); err != nil {
			return err
		}
	}

	// Finally, remove the now-empty directory.
	return Remove(fsys, name)
}
