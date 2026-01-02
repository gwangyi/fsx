package contextual

import (
	"context"
	"errors"
	"os"
)

// TruncateFS is the interface implemented by a file system that supports
// truncating files by name.
type TruncateFS interface {
	WriterFS

	// Truncate changes the size of the named file.
	Truncate(ctx context.Context, name string, size int64) error
}

// Truncate changes the size of the named file.
func Truncate(ctx context.Context, fsys FS, name string, size int64) error {
	if tfs, ok := fsys.(TruncateFS); ok {
		if err := tfs.Truncate(ctx, name, size); !errors.Is(err, errors.ErrUnsupported) {
			return intoPathErr("truncate", name, err)
		}
	}

	f, err := OpenFile(ctx, fsys, name, os.O_WRONLY, 0)
	if err != nil {
		return intoPathErr("truncate", name, err)
	}
	defer func() { _ = f.Close() }()

	return intoPathErr("truncate", name, f.Truncate(size))
}
