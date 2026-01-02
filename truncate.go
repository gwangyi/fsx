package fsx

import (
	"errors"
	"io/fs"
	"os"

	"github.com/gwangyi/fsx/internal"
)

// TruncateFS is the interface implemented by a file system that supports
// truncating files by name.
type TruncateFS interface {
	FS
	// Truncate changes the size of the named file.
	// If the file is a symbolic link, it changes the size of the link's target.
	Truncate(name string, size int64) error
}

// Truncate changes the size of the named file.
//
// If fsys implements TruncateFS, it calls fsys.Truncate.
// Otherwise, it attempts to open the file with write permissions and call
// the Truncate method on the returned File object.
func Truncate(fsys fs.FS, name string, size int64) error {
	// Try the optimized/direct TruncateFS implementation first.
	if fsys, ok := fsys.(TruncateFS); ok {
		if err := fsys.Truncate(name, size); !errors.Is(err, errors.ErrUnsupported) {
			return internal.IntoPathErr("truncate", name, err)
		}
	}

	// Fallback: Open the file and call Truncate on the file handle.
	// For Read-only file system, OpenFile(O_WRONLY) will fail with ErrUnsupported.
	f, err := OpenFile(fsys, name, os.O_WRONLY, 0)
	if err != nil {
		return internal.IntoPathErr("truncate", name, err)
	}
	defer func() { _ = f.Close() }()
	return internal.IntoPathErr("truncate", name, f.Truncate(size))
}
