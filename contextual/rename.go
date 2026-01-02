package contextual

import (
	"context"
	"io"
	"io/fs"
	"os"
)

// RenameFS is the interface implemented by a file system that supports
// renaming files.
type RenameFS interface {
	WriterFS

	// Rename moves oldname to newname.
	Rename(ctx context.Context, oldname, newname string) error
}

// Rename moves oldname to newname.
func Rename(ctx context.Context, fsys FS, oldname, newname string) error {
	if rfs, ok := fsys.(RenameFS); ok {
		return intoLinkErr("rename", oldname, newname, rfs.Rename(ctx, oldname, newname))
	}

	if oldname == newname {
		return nil
	}

	// Fallback: Copy and Remove.
	src, err := fsys.Open(ctx, oldname)
	if err != nil {
		return intoLinkErr("rename", oldname, newname, err)
	}
	defer func() { _ = src.Close() }()

	mode := fs.FileMode(0666)
	if info, err := src.Stat(); err == nil {
		mode = info.Mode()
	}

	// Create destination file with the same mode.
	dst, err := OpenFile(ctx, fsys, newname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return intoLinkErr("rename", oldname, newname, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return intoLinkErr("rename", oldname, newname, err)
	}

	if err := dst.Close(); err != nil {
		return intoLinkErr("rename", oldname, newname, err)
	}

	// Close src before removing to ensure no lock is held (e.g. on Windows)
	_ = src.Close()

	if err := Remove(ctx, fsys, oldname); err != nil {
		return intoLinkErr("rename", oldname, newname, err)
	}

	return nil
}
