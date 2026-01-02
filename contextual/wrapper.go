package contextual

import (
	"context"
	"io/fs"
	"time"

	"github.com/gwangyi/fsx"
)

// ToContextual converts a non-contextual fs.FS to a contextual FS.
// The returned FS ignores the context.
func ToContextual(fsys fs.FS) FS {
	return &contextualFS{fsys: fsys}
}

type contextualFS struct {
	fsys fs.FS
}

func (c *contextualFS) Open(ctx context.Context, name string) (fs.File, error) {
	return c.fsys.Open(name)
}

func (c *contextualFS) Create(ctx context.Context, name string) (File, error) {
	return fsx.Create(c.fsys, name)
}

func (c *contextualFS) OpenFile(ctx context.Context, name string, flag int, mode fs.FileMode) (File, error) {
	return fsx.OpenFile(c.fsys, name, flag, mode)
}

func (c *contextualFS) Remove(ctx context.Context, name string) error {
	return fsx.Remove(c.fsys, name)
}

func (c *contextualFS) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return fs.ReadFile(c.fsys, name)
}

func (c *contextualFS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	return fs.Stat(c.fsys, name)
}

func (c *contextualFS) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(c.fsys, name)
}

func (c *contextualFS) Mkdir(ctx context.Context, name string, perm fs.FileMode) error {
	return fsx.Mkdir(c.fsys, name, perm)
}

func (c *contextualFS) MkdirAll(ctx context.Context, name string, perm fs.FileMode) error {
	return fsx.MkdirAll(c.fsys, name, perm)
}

func (c *contextualFS) RemoveAll(ctx context.Context, name string) error {
	return fsx.RemoveAll(c.fsys, name)
}

func (c *contextualFS) Rename(ctx context.Context, oldname, newname string) error {
	return fsx.Rename(c.fsys, oldname, newname)
}

func (c *contextualFS) Symlink(ctx context.Context, oldname, newname string) error {
	return fsx.Symlink(c.fsys, oldname, newname)
}

func (c *contextualFS) ReadLink(ctx context.Context, name string) (string, error) {
	return fs.ReadLink(c.fsys, name)
}

func (c *contextualFS) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	return fs.Lstat(c.fsys, name)
}

func (c *contextualFS) Lchown(ctx context.Context, name, owner, group string) error {
	return fsx.Lchown(c.fsys, name, owner, group)
}

func (c *contextualFS) Truncate(ctx context.Context, name string, size int64) error {
	return fsx.Truncate(c.fsys, name, size)
}

func (c *contextualFS) WriteFile(ctx context.Context, name string, data []byte, perm fs.FileMode) error {
	return fsx.WriteFile(c.fsys, name, data, perm)
}

func (c *contextualFS) Chown(ctx context.Context, name, owner, group string) error {
	return fsx.Chown(c.fsys, name, owner, group)
}

func (c *contextualFS) Chmod(ctx context.Context, name string, mode fs.FileMode) error {
	return fsx.Chmod(c.fsys, name, mode)
}

func (c *contextualFS) Chtimes(ctx context.Context, name string, atime, ctime time.Time) error {
	return fsx.Chtimes(c.fsys, name, atime, ctime)
}
