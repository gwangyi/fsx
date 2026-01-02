package contextual

import (
	"context"
	"os"
	"path"
)

// RemoveAllFS is the interface implemented by a file system that supports
// an optimized RemoveAll method.
type RemoveAllFS interface {
	WriterFS

	// RemoveAll removes path and any children it contains.
	RemoveAll(ctx context.Context, name string) error
}

// RemoveAll removes path and any children it contains.
func RemoveAll(ctx context.Context, fsys FS, name string) error {
	if rfs, ok := fsys.(RemoveAllFS); ok {
		return intoPathErr("remove", name, rfs.RemoveAll(ctx, name))
	}

	err := Remove(ctx, fsys, name)
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	entries, readErr := ReadDir(ctx, fsys, name)
	if readErr != nil {
		return err
	}

	for _, entry := range entries {
		childPath := path.Join(name, entry.Name())
		if err := RemoveAll(ctx, fsys, childPath); err != nil {
			return err
		}
	}

	return Remove(ctx, fsys, name)
}
