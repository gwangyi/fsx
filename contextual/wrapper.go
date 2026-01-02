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

// FromContextual converts a contextual FS to a non-contextual fs.FS.
// The returned filesystem satisfies fsx.FileSystem and all standard io/fs interfaces
// by using the provided context for every operation.
//
// This is useful for integrating context-aware filesystems into existing
// non-contextual APIs or libraries that expect an fs.FS.
func FromContextual(fsys FS, ctx context.Context) fs.FS {
	return &nonContextualFS{fsys: fsys, ctx: ctx}
}

// nonContextualFS implements the non-contextual fsx.FileSystem interface
// by wrapping a contextual FS and a fixed context.Context.
// Every method call on this struct delegates to the corresponding
// package-level helper function (e.g., contextual.ReadFile, contextual.MkdirAll),
// ensuring that feature detection and fallbacks are handled consistently.
type nonContextualFS struct {
	fsys FS
	ctx  context.Context
}

// Open implements fs.FS.
func (n *nonContextualFS) Open(name string) (fs.File, error) {
	return n.fsys.Open(n.ctx, name)
}

// Create implements fsx.WriterFS.
func (n *nonContextualFS) Create(name string) (File, error) {
	return Create(n.ctx, n.fsys, name)
}

// OpenFile implements fsx.WriterFS.
func (n *nonContextualFS) OpenFile(name string, flag int, mode fs.FileMode) (File, error) {
	return OpenFile(n.ctx, n.fsys, name, flag, mode)
}

// Remove implements fsx.WriterFS.
func (n *nonContextualFS) Remove(name string) error {
	return Remove(n.ctx, n.fsys, name)
}

// ReadFile implements fs.ReadFileFS.
func (n *nonContextualFS) ReadFile(name string) ([]byte, error) {
	return ReadFile(n.ctx, n.fsys, name)
}

// Stat implements fs.StatFS.
func (n *nonContextualFS) Stat(name string) (fs.FileInfo, error) {
	return Stat(n.ctx, n.fsys, name)
}

// ReadDir implements fs.ReadDirFS.
func (n *nonContextualFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return ReadDir(n.ctx, n.fsys, name)
}

// Mkdir implements fsx.DirFS.
func (n *nonContextualFS) Mkdir(name string, perm fs.FileMode) error {
	return Mkdir(n.ctx, n.fsys, name, perm)
}

// MkdirAll implements fsx.MkdirAllFS.
func (n *nonContextualFS) MkdirAll(name string, perm fs.FileMode) error {
	return MkdirAll(n.ctx, n.fsys, name, perm)
}

// RemoveAll implements fsx.RemoveAllFS.
func (n *nonContextualFS) RemoveAll(name string) error {
	return RemoveAll(n.ctx, n.fsys, name)
}

// Rename implements fsx.RenameFS.
func (n *nonContextualFS) Rename(oldname, newname string) error {
	return Rename(n.ctx, n.fsys, oldname, newname)
}

// Symlink implements fsx.SymlinkFS.
func (n *nonContextualFS) Symlink(oldname, newname string) error {
	return Symlink(n.ctx, n.fsys, oldname, newname)
}

// ReadLink implements fs.ReadLinkFS.
func (n *nonContextualFS) ReadLink(name string) (string, error) {
	return ReadLink(n.ctx, n.fsys, name)
}

// Lstat implements fs.ReadLinkFS.
func (n *nonContextualFS) Lstat(name string) (fs.FileInfo, error) {
	return Lstat(n.ctx, n.fsys, name)
}

// Lchown implements fsx.LchownFS.
func (n *nonContextualFS) Lchown(name, owner, group string) error {
	return Lchown(n.ctx, n.fsys, name, owner, group)
}

// Truncate implements fsx.TruncateFS.
func (n *nonContextualFS) Truncate(name string, size int64) error {
	return Truncate(n.ctx, n.fsys, name, size)
}

// WriteFile implements fsx.WriteFileFS.
func (n *nonContextualFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	return WriteFile(n.ctx, n.fsys, name, data, perm)
}

// Chown implements fsx.ChangeFS.
func (n *nonContextualFS) Chown(name, owner, group string) error {
	return Chown(n.ctx, n.fsys, name, owner, group)
}

// Chmod implements fsx.ChangeFS.
func (n *nonContextualFS) Chmod(name string, mode fs.FileMode) error {
	return Chmod(n.ctx, n.fsys, name, mode)
}

// Chtimes implements fsx.ChangeFS.
func (n *nonContextualFS) Chtimes(name string, atime, ctime time.Time) error {
	return Chtimes(n.ctx, n.fsys, name, atime, ctime)
}

var _ fsx.FileSystem = &nonContextualFS{}
