package fsx

import (
	"io"
	"io/fs"
	"os"

	"github.com/gwangyi/fsx/internal"
)

// RenameFS is the interface implemented by a file system that supports
// renaming files.
type RenameFS interface {
	WriterFS

	// Rename moves oldname to newname.
	// If newname already exists and is not a directory, Rename replaces it.
	// OS-specific restrictions may apply when oldname and newname are in different directories.
	Rename(oldname, newname string) error
}

// Rename moves oldname to newname.
//
// If fsys implements RenameFS, it calls fsys.Rename.
// Otherwise, it attempts to simulate the rename operation by copying the source
// to the destination and then removing the source. This fallback is not atomic.
func Rename(fsys fs.FS, oldname, newname string) error {
	if rfs, ok := fsys.(RenameFS); ok {
		return internal.IntoLinkErr("rename", oldname, newname, rfs.Rename(oldname, newname))
	}

	if oldname == newname {
		return nil
	}

	// Fallback: Copy and Remove.
	src, err := fsys.Open(oldname)
	if err != nil {
		return internal.IntoLinkErr("rename", oldname, newname, err)
	}
	defer func() { _ = src.Close() }()

	mode := fs.FileMode(0666)
	if info, err := src.Stat(); err == nil {
		mode = info.Mode()
	}

	// Create destination file with the same mode.
	dst, err := OpenFile(fsys, newname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return internal.IntoLinkErr("rename", oldname, newname, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return internal.IntoLinkErr("rename", oldname, newname, err)
	}

	if err := dst.Close(); err != nil {
		return internal.IntoLinkErr("rename", oldname, newname, err)
	}

	// Close src before removing to ensure no lock is held (e.g. on Windows)
	_ = src.Close()

	if err := Remove(fsys, oldname); err != nil {
		return internal.IntoLinkErr("rename", oldname, newname, err)
	}

	return nil
}
