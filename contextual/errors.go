package contextual

import (
	"github.com/gwangyi/fsx/internal"
)

func intoPathErr(op, path string, err error) error {
	return internal.IntoPathErr(op, path, err)
}

func intoLinkErr(op, oldpath, newpath string, err error) error {
	return internal.IntoLinkErr(op, oldpath, newpath, err)
}
