package contextual

import (
	"context"
	"io/fs"

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
