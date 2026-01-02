package contextual

import (
	"context"
	"errors"
	"io/fs"
	"os"
)

// WriteFileFS is the interface implemented by a filesystem that provides
// an optimized WriteFile method.
type WriteFileFS interface {
	WriterFS

	// WriteFile writes data to the named file, creating it if necessary.
	WriteFile(ctx context.Context, name string, data []byte, perm fs.FileMode) error
}

// WriteFile writes data to the named file, creating it if necessary.
func WriteFile(ctx context.Context, fsys FS, name string, data []byte, perm fs.FileMode) error {
	if wfs, ok := fsys.(WriteFileFS); ok {
		if err := wfs.WriteFile(ctx, name, data, perm); !errors.Is(err, errors.ErrUnsupported) {
			return intoPathErr("writefile", name, err)
		}
	}

	f, err := OpenFile(ctx, fsys, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return intoPathErr("writefile", name, err)
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return intoPathErr("writefile", name, err)
}
