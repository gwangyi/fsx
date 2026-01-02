package contextual

import (
	"context"
	"io/fs"
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
