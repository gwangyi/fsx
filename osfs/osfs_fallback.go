//go:build !unix

package osfs

import (
	"errors"
)

// Chown is not implemented very well in non-unix system.
func (fsys filesystem) Chown(name, owner, group string) error {
	return errors.ErrUnsupported
}

// Lchown is not implemented very well in non-unix system.
func (fsys filesystem) Lchown(name, owner, group string) error {
	return errors.ErrUnsupported
}
