// Package contextualfs provides extended filesystem interfaces that support write operations.
package contextual

//go:generate mockgen -destination ../mockfs/contextual/mockfs.go -package cmockfs . FS,ReadFileFS,WriterFS,ChangeFS,ReadDirFS,DirFS,MkdirAllFS,RemoveAllFS,RenameFS,StatFS,ReadLinkFS,SymlinkFS,LchownFS,TruncateFS,WriteFileFS,FileSystem

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/gwangyi/fsx/internal"
)

// File is an open file that supports reading, writing, and truncation.
type File = internal.File

// FileInfo extends the standard fs.FileInfo interface with additional metadata
// like ownership, access time, and change time.
type FileInfo = internal.FileInfo

// ExtendFileInfo returns a FileInfo that wraps the provided fs.FileInfo,
// attempting to extract extended system-specific information.
func ExtendFileInfo(fi fs.FileInfo) FileInfo {
	return internal.ExtendFileInfo(fi)
}

// FS is the interface implemented by a file system that supports
// context-aware Open.
//
// It mirrors io/fs.FS but with a context parameter.
type FS interface {
	// Open opens the named file.
	//
	// When Open returns an error, it should be of type *fs.PathError
	// with the Op field set to "open", the Path field set to name,
	// and the Err field describing the problem.
	//
	// Open should reject attempts to open names that do not satisfy
	// fs.ValidPath(name), returning a *fs.PathError with Err set to
	// fs.ErrInvalid or fs.ErrNotExist.
	Open(ctx context.Context, name string) (fs.File, error)
}

// ReadFileFS is the interface implemented by a file system that supports
// context-aware ReadFile.
type ReadFileFS interface {
	FS
	// ReadFile reads the named file and returns its contents.
	// A successful call returns a nil error, not io.EOF.
	// (Because ReadFile reads the whole file, the expected behavior
	// is to return the size of the file.)
	ReadFile(ctx context.Context, name string) ([]byte, error)
}

// WriterFS is the interface implemented by a file system that supports
// write operations (Create, OpenFile, Remove) in addition to Open.
//
// It corresponds to fsx.WriterFS but with context parameters.
type WriterFS interface {
	FS

	// Create creates or truncates the named file. If the file already exists,
	// it is truncated. If the file does not exist, it is created with mode 0666
	// (before umask). If successful, methods on the returned File can
	// be used for I/O; the associated file descriptor has mode O_RDWR.
	// If there is an error, it will be of type *fs.PathError.
	Create(ctx context.Context, name string) (File, error)

	// OpenFile is the generalized open call; most users will use Open
	// or Create instead. It opens the named file with specified flag
	// (O_RDONLY etc.) and perm (0666 etc.) if applicable. If successful,
	// methods on the returned File can be used for I/O.
	// If there is an error, it will be of type *fs.PathError.
	OpenFile(ctx context.Context, name string, flag int, mode fs.FileMode) (File, error)

	// Remove removes the named file or (empty) directory.
	// If there is an error, it will be of type *fs.PathError.
	Remove(ctx context.Context, name string) error
}

// Open opens the named file in the given filesystem.
func Open(ctx context.Context, fsys FS, name string) (fs.File, error) {
	return fsys.Open(ctx, name)
}

// Create creates or truncates the named file in the given filesystem.
// If fsys implements WriterFS, it calls fsys.Create(ctx, name).
// Otherwise, it returns errors.ErrUnsupported.
func Create(ctx context.Context, fsys FS, name string) (File, error) {
	if xfs, ok := fsys.(WriterFS); ok {
		f, err := xfs.Create(ctx, name)
		return f, intoPathErr("open", name, err)
	}

	return nil, errors.ErrUnsupported
}

// OpenFile opens the named file with specified flag and mode in the given filesystem.
// If fsys implements WriterFS, it calls fsys.OpenFile(ctx, name, flag, mode).
// Otherwise, it attempts a fallback for read-only access.
func OpenFile(ctx context.Context, fsys FS, name string, flag int, mode fs.FileMode) (File, error) {
	if xfs, ok := fsys.(WriterFS); ok {
		if f, err := xfs.OpenFile(ctx, name, flag, mode); !errors.Is(err, errors.ErrUnsupported) {
			return f, intoPathErr("open", name, err)
		}
	}

	if flag&internal.O_ACCMODE == os.O_RDONLY {
		f, err := fsys.Open(ctx, name)
		if err != nil {
			return nil, intoPathErr("open", name, err)
		}
		return internal.ReadOnlyFile{File: f}, nil
	}

	return nil, errors.ErrUnsupported
}

// Remove removes the named file or (empty) directory from the filesystem.
func Remove(ctx context.Context, fsys FS, name string) error {
	if xfs, ok := fsys.(WriterFS); ok {
		return intoPathErr("remove", name, xfs.Remove(ctx, name))
	}

	return errors.ErrUnsupported
}

// ReadFile reads the named file from the given filesystem and returns its contents.
func ReadFile(ctx context.Context, fsys FS, name string) ([]byte, error) {
	if fsys, ok := fsys.(ReadFileFS); ok {
		return fsys.ReadFile(ctx, name)
	}

	f, err := fsys.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return io.ReadAll(f)
}

type FileSystem interface {
	FS
	ChangeFS
	DirFS
	LchownFS
	MkdirAllFS
	ReadDirFS
	ReadFileFS
	ReadLinkFS
	RemoveAllFS
	RenameFS
	StatFS
	SymlinkFS
	TruncateFS
	WriteFileFS
	WriterFS
}
