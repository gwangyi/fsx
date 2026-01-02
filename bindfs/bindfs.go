package bindfs

import (
	"context"
	"io/fs"
	"os"
	"path"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/contextual"
)

func Static[T any](val T) func(string) T {
	return func(string) T {
		return val
	}
}

// Config defines the override configuration for bindfs.
type Config struct {
	GrantPerm  func(name string) fs.FileMode
	RevokePerm func(name string) fs.FileMode
	Owner      func(name string) string
	Group      func(name string) string
}

type filesystem struct {
	Config
	fs contextual.FS
}

// New creates a new bindfs that delegates all operations to fsys but
// overrides metadata according to config.
func New(fsys contextual.FS, config Config) contextual.FileSystem {
	f := &filesystem{
		Config: config,
		fs:     fsys,
	}
	return f
}

type fileInfo struct {
	contextual.FileInfo
	name string
	fs   *filesystem
}

func (fi *fileInfo) Owner() string {
	if fi.fs.Owner != nil {
		return fi.fs.Owner(fi.name)
	}
	return fi.FileInfo.Owner()
}

func (fi *fileInfo) Group() string {
	if fi.fs.Group != nil {
		return fi.fs.Group(fi.name)
	}
	return fi.FileInfo.Group()
}

func (fi *fileInfo) Mode() fs.FileMode {
	mode := fi.FileInfo.Mode()
	if fi.fs.GrantPerm != nil {
		mode |= fi.fs.GrantPerm(fi.name).Perm()
	}
	if fi.fs.RevokePerm != nil {
		mode &= ^fi.fs.RevokePerm(fi.name).Perm()
	}
	return mode
}

type dirEntry struct {
	fs.DirEntry
	name string
	fs   *filesystem
}

func (d *dirEntry) Info() (fs.FileInfo, error) {
	fi, err := d.DirEntry.Info()
	if err != nil {
		return nil, err
	}
	return d.fs.wrapFileInfo(d.name, fi), nil
}

func (f *filesystem) wrapFileInfo(name string, fi fs.FileInfo) fs.FileInfo {
	if fi == nil {
		return nil
	}
	return &fileInfo{
		FileInfo: contextual.ExtendFileInfo(fi),
		name:     name,
		fs:       f,
	}
}

func (f *filesystem) wrapDirEntry(parent string, de fs.DirEntry) fs.DirEntry {
	if de == nil {
		return nil
	}
	return &dirEntry{
		DirEntry: de,
		name:     path.Join(parent, de.Name()),
		fs:       f,
	}
}

type fileWrapper struct {
	fsx.File
	name string
	fs   *filesystem
}

func (f *fileWrapper) Stat() (fs.FileInfo, error) {
	fi, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	return f.fs.wrapFileInfo(f.name, fi), nil
}

func (f *filesystem) Open(ctx context.Context, name string) (fs.File, error) {
	file, err := contextual.OpenFile(ctx, f.fs, name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: file, name: name, fs: f}, nil
}

func (f *filesystem) Create(ctx context.Context, name string) (fsx.File, error) {
	file, err := contextual.Create(ctx, f.fs, name)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: file, name: name, fs: f}, nil
}

func (f *filesystem) OpenFile(ctx context.Context, name string, flag int, mode fs.FileMode) (fsx.File, error) {
	file, err := contextual.OpenFile(ctx, f.fs, name, flag, mode)
	if err != nil {
		return nil, err
	}
	return &fileWrapper{File: file, name: name, fs: f}, nil
}

func (f *filesystem) Remove(ctx context.Context, name string) error {
	return contextual.Remove(ctx, f.fs, name)
}

func (f *filesystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return contextual.ReadFile(ctx, f.fs, name)
}

func (f *filesystem) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	fi, err := contextual.Stat(ctx, f.fs, name)
	if err != nil {
		return nil, err
	}
	return f.wrapFileInfo(name, fi), nil
}

func (f *filesystem) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	entries, err := contextual.ReadDir(ctx, f.fs, name)
	if err != nil {
		return nil, err
	}
	wrapped := make([]fs.DirEntry, len(entries))
	for i, e := range entries {
		wrapped[i] = f.wrapDirEntry(name, e)
	}
	return wrapped, nil
}

func (f *filesystem) Mkdir(ctx context.Context, name string, perm fs.FileMode) error {
	return contextual.Mkdir(ctx, f.fs, name, perm)
}

func (f *filesystem) MkdirAll(ctx context.Context, name string, perm fs.FileMode) error {
	return contextual.MkdirAll(ctx, f.fs, name, perm)
}

func (f *filesystem) RemoveAll(ctx context.Context, name string) error {
	return contextual.RemoveAll(ctx, f.fs, name)
}

func (f *filesystem) Rename(ctx context.Context, oldname, newname string) error {
	return contextual.Rename(ctx, f.fs, oldname, newname)
}

func (f *filesystem) Symlink(ctx context.Context, oldname, newname string) error {
	return contextual.Symlink(ctx, f.fs, oldname, newname)
}

func (f *filesystem) ReadLink(ctx context.Context, name string) (string, error) {
	return contextual.ReadLink(ctx, f.fs, name)
}

func (f *filesystem) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	fi, err := contextual.Lstat(ctx, f.fs, name)
	if err != nil {
		return nil, err
	}
	return f.wrapFileInfo(name, fi), nil
}

func (f *filesystem) Lchown(ctx context.Context, name, owner, group string) error {
	return contextual.Lchown(ctx, f.fs, name, owner, group)
}

func (f *filesystem) Truncate(ctx context.Context, name string, size int64) error {
	return contextual.Truncate(ctx, f.fs, name, size)
}

func (f *filesystem) WriteFile(ctx context.Context, name string, data []byte, perm fs.FileMode) error {
	return contextual.WriteFile(ctx, f.fs, name, data, perm)
}

func (f *filesystem) Chown(ctx context.Context, name, owner, group string) error {
	return contextual.Chown(ctx, f.fs, name, owner, group)
}

func (f *filesystem) Chmod(ctx context.Context, name string, mode fs.FileMode) error {
	return contextual.Chmod(ctx, f.fs, name, mode)
}

func (f *filesystem) Chtimes(ctx context.Context, name string, atime, ctime time.Time) error {
	return contextual.Chtimes(ctx, f.fs, name, atime, ctime)
}

var _ contextual.FileSystem = &filesystem{}
