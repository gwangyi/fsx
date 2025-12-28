// Package osfs provides a robust and secure implementation of the fsx.FS filesystem interface
// by leveraging the host's operating system files. It is designed to confine file operations
// within a specified root directory using Go's `os.Root` introduced since Go 1.24+.
// This confinement mechanism prevents directory traversal attacks (e.g., via ".." paths)
// and limits access to resources outside the designated root, making it ideal for
// sandboxed environments, serving static content from a restricted directory,
// or any scenario requiring strict path-based security.
package osfs

import (
	"io/fs"
	"os"

	"github.com/gwangyi/fsx"
)

// filesystem is the main implementation of the `fsx.FS` interface for the `osfs` package.
// It embeds `minimalFS` to inherit the `os.Root` functionality, thereby ensuring all
// file operations are securely confined to the designated root directory.
// This structure prevents unauthorized access outside the root, including attempts
// to use ".." for directory traversal or following symbolic links to external locations,
// making it suitable for secure, isolated file system interactions.
type filesystem struct {
	minimalFS
}

// minimalFS is a wrapper around `*os.Root`. It provides the core file system
// operations (`Create`, `Open`, `OpenFile`) that directly delegate to the
// underlying `os.Root` instance. This embedding allows `filesystem` to
// inherit these root-confined operations, ensuring that all access is
// restricted to the root directory established by `os.OpenRoot`.
type minimalFS struct {
	*os.Root
}

// New creates and returns a new `fs.FS` instance that is rooted at the specified directory `name`.
// This function uses `os.OpenRoot` to establish a secure boundary, ensuring that all subsequent
// file system operations performed through the returned `fs.FS` are strictly confined to `name`
// and its subdirectories. This prevents any access to files or directories outside of `name`.
//
// Parameters:
//
//	name: The path to the directory that will serve as the root of the new confined filesystem.
//
// Returns:
//
//	A new `fs.FS` instance representing the confined filesystem, or an error if `name`
//	cannot be opened or is not a valid directory.
func New(name string) (fs.FS, error) {
	r, err := os.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	return filesystem{minimalFS: minimalFS{Root: r}}, nil
}

// Create creates the named file within the filesystem's root.
// It delegates the call to the underlying `os.Root.Create` method, which ensures
// that the file is created relative to the confined root directory.
// The `name` parameter must be a path relative to the `osfs` instance's root.
// If the file already exists, it is truncated to zero length.
//
// Returns:
//
//	An `fsx.File` instance representing the newly created file, or an error if
//	the file cannot be created (e.g., due to invalid path or permissions).
func (fsys minimalFS) Create(name string) (fsx.File, error) {
	return fsys.Root.Create(name)
}

// Open opens the named file for reading within the filesystem's root.
// This method delegates to the `os.Root.Open` function, ensuring that access is
// restricted to files and directories contained within the `osfs` instance's
// root. The `name` parameter must specify a path relative to this root.
//
// Returns:
//
//	An `fs.File` instance for reading the file, or an error if the file
//	cannot be opened (e.g., file not found, permission denied, or `name`
//	attempts to access a path outside the confined root).
func (fsys minimalFS) Open(name string) (fs.File, error) {
	return fsys.Root.Open(name)
}

// OpenFile opens the named file within the filesystem's root with specified flags and mode.
// This method serves as a pass-through to `os.Root.OpenFile`, inheriting its behavior
// for opening files. The `name` parameter must be a path relative to the root
// of the `osfs` instance, and all operations are confined within this root.
//
// Parameters:
//
//	name: The path to the file to open, relative to the confined root.
//	flag: A bitmask of flags (e.g., `os.O_RDONLY`, `os.O_WRONLY`, `os.O_RDWR`, `os.O_CREATE`, `os.O_TRUNC`, `os.O_APPEND`)
//	      that specify the file's open mode.
//	mode: The file mode (permissions) to use if a new file is created.
//
// Returns:
//
//	An `fsx.File` instance that satisfies the requested flags, or an error if the
//	file cannot be opened (e.g., due to invalid path, permissions, or if `name`
//	attempts to access a path outside the confined root).
func (fsys minimalFS) OpenFile(name string, flag int, mode fs.FileMode) (fsx.File, error) {
	return fsys.Root.OpenFile(name, flag, mode)
}

// Ensure that `filesystem` correctly implements all expected filesystem interfaces.
// This compile-time check verifies that `filesystem` satisfies the contracts defined by:
// - `fsx.FS`: The primary filesystem interface.
// - `fs.ReadFileFS`: For efficiently reading entire files.
// - `fsx.TruncateFS`: For resizing files.
// - `fsx.WriteFileFS`: For writing to files.
var _ fsx.FS = filesystem{}
var _ fs.ReadFileFS = filesystem{}
var _ fsx.WriteFileFS = filesystem{}
var _ fsx.RenameFS = filesystem{}
