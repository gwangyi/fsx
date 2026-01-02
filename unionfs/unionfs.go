// Package unionfs provides a union filesystem implementation that merges multiple
// filesystems into a single view. It supports one read-write (RW) layer and
// multiple read-only (RO) layers.
//
// When a file is modified, it is copied from a read-only layer to the read-write
// layer (Copy-on-Write). Deletions are handled using "whiteout" files (e.g., .wh.<filename>)
// created in the read-write layer to hide files present in the read-only layers.
package unionfs

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/contextual"
)

// filesystem is a union filesystem that has one read-write layer and multiple
// read-only layers. It implements the contextual.FileSystem interface.
type filesystem struct {
	rw         contextual.FS
	ro         []contextual.FS
	copyOnRead bool
}

// New creates a new union filesystem with a mandatory read-write layer (rw)
// and optional read-only layers (ro). The layers are searched in order:
// rw is searched first, then ro layers in the order they were provided.
func New(rw contextual.FS, ro ...contextual.FS) *filesystem {
	return &filesystem{
		rw: rw,
		ro: ro,
	}
}

// SetCopyOnRead enables or disables copy-on-read behavior for the given filesystem.
// If enabled, opening a file for reading from a read-only layer will trigger
// a copy of that file to the read-write layer.
func SetCopyOnRead(fs contextual.FS, enabled bool) {
	fs.(*filesystem).copyOnRead = enabled
}

// isWhiteout checks if a whiteout file exists in the read-write layer for the given name.
// A whiteout file is named ".wh.<original_filename>" and indicates that the
// file should be treated as non-existent, even if it exists in a read-only layer.
func (f *filesystem) isWhiteout(ctx context.Context, name string) bool {
	dir, file := path.Split(name)
	wh := path.Join(dir, ".wh."+file)
	_, err := contextual.Stat(ctx, f.rw, wh)
	return err == nil
}

// createWhiteout creates a whiteout file in the read-write layer for the given name.
// This is used to "delete" a file that exists in a read-only layer.
func (f *filesystem) createWhiteout(ctx context.Context, name string) error {
	dir, file := path.Split(name)
	wh := path.Join(dir, ".wh."+file)
	// Ensure parent exists in RW
	if dir != "" && dir != "." {
		if err := contextual.MkdirAll(ctx, f.rw, dir, 0755); err != nil {
			return err
		}
	}
	return contextual.WriteFile(ctx, f.rw, wh, nil, 0644)
}

// Open opens the named file for reading. It satisfies the contextual.FS interface.
func (f *filesystem) Open(ctx context.Context, name string) (fs.File, error) {
	return f.OpenFile(ctx, name, os.O_RDONLY, 0)
}

// copyToRW copies a file or directory from one of the read-only layers to
// the read-write layer. If the file already exists in the read-write layer,
// it does nothing and returns nil.
func (f *filesystem) copyToRW(ctx context.Context, name string) error {
	// Check if already in RW
	if _, err := contextual.Stat(ctx, f.rw, name); !os.IsNotExist(err) {
		return err
	}

	// Find in RO
	var src contextual.FS
	var info fs.FileInfo
	for _, ro := range f.ro {
		if i, err := contextual.Stat(ctx, ro, name); err == nil {
			src = ro
			info = i
			break
		}
	}

	if src == nil {
		return fs.ErrNotExist
	}

	if info.IsDir() {
		return contextual.MkdirAll(ctx, f.rw, name, info.Mode().Perm())
	}

	// Copy file
	// Ensure parent directories exist in RW
	parent := path.Dir(name)
	if parent != "." {
		if err := contextual.MkdirAll(ctx, f.rw, parent, 0755); err != nil {
			return err
		}
	}

	in, err := src.Open(ctx, name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := contextual.OpenFile(ctx, f.rw, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// If there was a whiteout, remove it since we now have the real file in RW
	dir, file := path.Split(name)
	wh := path.Join(dir, ".wh."+file)
	_ = contextual.Remove(ctx, f.rw, wh)

	return nil
}

// OpenFile is the generalized open call. It implements Copy-on-Write: if the
// file is opened for writing and only exists in a read-only layer, it is
// first copied to the read-write layer.
func (f *filesystem) OpenFile(ctx context.Context, name string, flag int, mode fs.FileMode) (fsx.File, error) {
	if flag&fsx.O_ACCMODE != os.O_RDONLY || flag&os.O_CREATE != 0 || flag&os.O_TRUNC != 0 || flag&os.O_APPEND != 0 {
		// Write operation
		if err := f.copyToRW(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		// If copyToRW returned ErrNotExist, it means it's a new file to be created in RW.
		// If there was a whiteout, copyToRW (via Stat/isWhiteout) would have found it if we implemented it there.
		// Actually copyToRW doesn't check whiteouts yet.
		return contextual.OpenFile(ctx, f.rw, name, flag, mode)
	}

	// Read-only open
	file, err := contextual.OpenFile(ctx, f.rw, name, flag, mode)
	if err == nil {
		return file, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if f.isWhiteout(ctx, name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	for _, ro := range f.ro {
		file, err := contextual.OpenFile(ctx, ro, name, flag, mode)
		if err == nil {
			if f.copyOnRead {
				_ = file.Close()
				if err := f.copyToRW(ctx, name); err != nil {
					return nil, err
				}
				return contextual.OpenFile(ctx, f.rw, name, flag, mode)
			}
			return file, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

// Create creates the named file in the read-write layer.
func (f *filesystem) Create(ctx context.Context, name string) (fsx.File, error) {
	return f.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Remove removes the named file or directory. If the file exists in the
// read-write layer, it is removed. If it also exists in a read-only layer,
// a whiteout file is created in the read-write layer to hide it.
func (f *filesystem) Remove(ctx context.Context, name string) error {
	// If it exists in RW, remove it.
	err := contextual.Remove(ctx, f.rw, name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	// Check if it exists in RO
	inRO := false
	for _, ro := range f.ro {
		if _, err := contextual.Stat(ctx, ro, name); err == nil {
			inRO = true
			break
		}
	}

	if inRO {
		return f.createWhiteout(ctx, name)
	}

	return err // Return original Remove error if not in RO
}

// Stat returns FileInfo describing the named file. It checks the read-write
// layer first, then considers whiteouts, and finally checks read-only layers.
func (f *filesystem) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := contextual.Stat(ctx, f.rw, name)
	if err == nil {
		return info, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if f.isWhiteout(ctx, name) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	for _, ro := range f.ro {
		info, err := contextual.Stat(ctx, ro, name)
		if err == nil {
			return info, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
}

// ReadDir reads the named directory and returns a list of directory entries
// sorted by name. It merges entries from all layers and filters out whiteouts.
func (f *filesystem) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	entries := make(map[string]fs.DirEntry)
	whiteouts := make(map[string]bool)

	rwEntries, err := contextual.ReadDir(ctx, f.rw, name)
	if err == nil {
		for _, e := range rwEntries {
			if after, found := strings.CutPrefix(e.Name(), ".wh."); found {
				whiteouts[after] = true
				continue
			}
			entries[e.Name()] = e
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	for _, ro := range f.ro {
		roEntries, err := contextual.ReadDir(ctx, ro, name)
		if err == nil {
			for _, e := range roEntries {
				if whiteouts[e.Name()] {
					continue
				}
				if _, ok := entries[e.Name()]; !ok {
					entries[e.Name()] = e
				}
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if len(entries) == 0 && len(whiteouts) == 0 && err != nil {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}

	var list []fs.DirEntry
	for _, e := range entries {
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

// Mkdir creates a new directory in the read-write layer.
func (f *filesystem) Mkdir(ctx context.Context, name string, perm fs.FileMode) error {
	if err := contextual.Mkdir(ctx, f.rw, name, perm); err != nil {
		return err
	}
	// Remove whiteout if any, since we've just created the directory
	dir, file := path.Split(name)
	wh := path.Join(dir, ".wh."+file)
	_ = contextual.Remove(ctx, f.rw, wh)
	return nil
}

// MkdirAll creates a directory and all necessary parents in the read-write layer.
func (f *filesystem) MkdirAll(ctx context.Context, name string, perm fs.FileMode) error {
	if err := contextual.MkdirAll(ctx, f.rw, name, perm); err != nil {
		return err
	}
	// Remove whiteout if any
	dir, file := path.Split(name)
	wh := path.Join(dir, ".wh."+file)
	_ = contextual.Remove(ctx, f.rw, wh)
	return nil
}

// RemoveAll removes path and any children it contains from the read-write layer.
// If the path exists in a read-only layer, a whiteout is created.
func (f *filesystem) RemoveAll(ctx context.Context, name string) error {
	// This is tricky for unionfs. For now, just remove from RW and whiteout if needed.
	// Properly removing all in unionfs usually requires whiteouting the directory itself.
	if err := contextual.RemoveAll(ctx, f.rw, name); err != nil {
		return err
	}

	inRO := false
	for _, ro := range f.ro {
		if _, err := contextual.Stat(ctx, ro, name); err == nil {
			inRO = true
			break
		}
	}
	if inRO {
		return f.createWhiteout(ctx, name)
	}
	return nil
}

// Rename renames a file. If the file exists in a read-only layer, it is first
// copied to the read-write layer, then renamed there, and a whiteout is
// created for the old name.
func (f *filesystem) Rename(ctx context.Context, oldname, newname string) error {
	// Check if oldname exists in union
	if _, err := f.Stat(ctx, oldname); err != nil {
		return err
	}

	// If oldname is in RO, we need a whiteout after rename
	inRO := false
	for _, ro := range f.ro {
		if _, err := contextual.Stat(ctx, ro, oldname); err == nil {
			inRO = true
			break
		}
	}

	if err := f.copyToRW(ctx, oldname); err != nil {
		return err
	}
	if err := contextual.Rename(ctx, f.rw, oldname, newname); err != nil {
		return err
	}

	if inRO {
		return f.createWhiteout(ctx, oldname)
	}
	return nil
}

// Symlink creates newname as a symbolic link to oldname in the read-write layer.
func (f *filesystem) Symlink(ctx context.Context, oldname, newname string) error {
	if err := contextual.Symlink(ctx, f.rw, oldname, newname); err != nil {
		return err
	}
	dir, file := path.Split(newname)
	wh := path.Join(dir, ".wh."+file)
	_ = contextual.Remove(ctx, f.rw, wh)
	return nil
}

// ReadLink returns the destination of the named symbolic link.
func (f *filesystem) ReadLink(ctx context.Context, name string) (string, error) {
	l, err := contextual.ReadLink(ctx, f.rw, name)
	if err == nil {
		return l, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	if f.isWhiteout(ctx, name) {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrNotExist}
	}

	for _, ro := range f.ro {
		l, err := contextual.ReadLink(ctx, ro, name)
		if err == nil {
			return l, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
	}

	return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrNotExist}
}

// Lstat returns FileInfo describing the named file. If the file is a
// symbolic link, the returned FileInfo describes the symbolic link.
func (f *filesystem) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := contextual.Lstat(ctx, f.rw, name)
	if err == nil {
		return info, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	if f.isWhiteout(ctx, name) {
		return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
	}

	for _, ro := range f.ro {
		info, err := contextual.Lstat(ctx, ro, name)
		if err == nil {
			return info, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
}

// Lchown changes the numeric uid and gid of the named file. If the file is
// in a read-only layer, it is first copied to the read-write layer.
func (f *filesystem) Lchown(ctx context.Context, name, owner, group string) error {
	if err := f.copyToRW(ctx, name); err != nil {
		return err
	}
	return contextual.Lchown(ctx, f.rw, name, owner, group)
}

// Truncate changes the size of the named file. If the file is in a
// read-only layer, it is first copied to the read-write layer.
func (f *filesystem) Truncate(ctx context.Context, name string, size int64) error {
	if err := f.copyToRW(ctx, name); err != nil {
		return err
	}
	return contextual.Truncate(ctx, f.rw, name, size)
}

// WriteFile writes data to a file in the read-write layer.
func (f *filesystem) WriteFile(ctx context.Context, name string, data []byte, perm fs.FileMode) error {
	return contextual.WriteFile(ctx, f.rw, name, data, perm)
}

// Chown changes the numeric uid and gid of the named file. If the file is
// in a read-only layer, it is first copied to the read-write layer.
func (f *filesystem) Chown(ctx context.Context, name, owner, group string) error {
	if err := f.copyToRW(ctx, name); err != nil {
		return err
	}
	return contextual.Chown(ctx, f.rw, name, owner, group)
}

// Chmod changes the mode of the named file. If the file is in a
// read-only layer, it is first copied to the read-write layer.
func (f *filesystem) Chmod(ctx context.Context, name string, mode fs.FileMode) error {
	if err := f.copyToRW(ctx, name); err != nil {
		return err
	}
	return contextual.Chmod(ctx, f.rw, name, mode)
}

// Chtimes changes the access and modification times of the named file.
// If the file is in a read-only layer, it is first copied to the read-write layer.
func (f *filesystem) Chtimes(ctx context.Context, name string, atime, ctime time.Time) error {
	if err := f.copyToRW(ctx, name); err != nil {
		return err
	}
	return contextual.Chtimes(ctx, f.rw, name, atime, ctime)
}

// ReadFile reads the named file and returns its contents. It checks the
// read-write layer first, then the read-only layers.
func (f *filesystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	data, err := contextual.ReadFile(ctx, f.rw, name)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	for _, ro := range f.ro {
		data, err := contextual.ReadFile(ctx, ro, name)
		if err == nil {
			if f.copyOnRead {
				if err := f.WriteFile(ctx, name, data, 0666); err != nil {
					return nil, err
				}
			}
			return data, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
}

// Compile-time interface checks
var _ contextual.FileSystem = &filesystem{}
