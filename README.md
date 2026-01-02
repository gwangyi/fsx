# fsx

`fsx` is a Go library that extends the standard `io/fs` package to support **write operations** and creates a rich ecosystem of composable filesystem abstractions.

While Go's `io/fs` provides an excellent abstraction for read-only filesystems, `fsx` defines standard interfaces for modifying filesystems—including creating files, writing data, removing entries, and changing metadata. It also provides a suite of robust filesystem implementations like UnionFS, EvictFS (cache), and secure OS-backed filesystems.

## Features

- **Standard Write Interfaces**: Extends `fs.FS` with `WriterFS` for `Create`, `OpenFile`, and `Remove`.
- **Context Awareness**: The `contextual` package mirrors filesystem operations with `context.Context` support for timeout and cancellation.
- **Composable implementations**:
  - **`osfs`**: A secure, confined filesystem rooted in a specific OS directory (leveraging Go 1.24+ `os.Root`).
  - **`unionfs`**: A Copy-on-Write (CoW) union filesystem merging a read-write layer with multiple read-only layers.
  - **`evictfs`**: A self-cleaning filesystem that evicts files based on LRU, total size, or file age (perfect for caches).
  - **`bindfs`**: A wrapper that can override file permissions and ownership dynamically.
- **Testable**: Includes `mockfs` generated with `mockgen` for easy unit testing of filesystem interactions.

## Installation

```bash
go get github.com/gwangyi/fsx
```

## Usage

### Basic Filesystem Operations

Use the top-level `fsx` helper functions to perform write operations on any compatible filesystem.

```go
package main

import (
	"log"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/osfs"
)

func main() {
	// Create a secure filesystem confined to a local directory
	fsys, err := osfs.New("./workspace")
	if err != nil {
		log.Fatal(err)
	}

	// Create a new file
	file, err := fsx.Create(fsys, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	
	_, err = file.Write([]byte("Hello, fsx!"))
	file.Close()

	if err != nil {
		log.Fatal(err)
	}

	// Remove the file
	if err := fsx.Remove(fsys, "example.txt"); err != nil {
		log.Fatal(err)
	}
}
```

### Union Filesystem (Overlay)

Create a filesystem that merges a read-only base layer with a writable upper layer. Modifications are written to the upper layer (Copy-on-Write), leaving the base intact.

```go
import (
	"github.com/gwangyi/fsx/unionfs"
	"github.com/gwangyi/fsx/osfs"
	"github.com/gwangyi/fsx/contextual"
)

// ...

// Base layer (Read-Only)
base, _ := osfs.New("./base_data")
// Upper layer (Read-Write)
upper, _ := osfs.New("./user_data")

// Create a unified view
// We use contextual.ToContextual to adapt standard fs.FS to contextual.FS
ufs := unionfs.New(contextual.ToContextual(upper), contextual.ToContextual(base))

// Reads check 'upper' then 'base'
// Writes always go to 'upper'
```

### Eviction Filesystem (Cache)

Wrap a filesystem to automatically enforce limits on size or file count.

```go
import (
	"time"
	"github.com/gwangyi/fsx/evictfs"
)

// ...

config := evictfs.Config{
    MaxFiles: 1000,
    MaxSize:  100 * 1024 * 1024, // 100 MB
    MaxAge:   24 * time.Hour,
}

// wrappedFS will now automatically delete old/least-used files 
// to stay within limits
cacheFS, err := evictfs.New(ctx, underlyingFS, config)
```

## Packages

| Package | Description |
| :--- | :--- |
| `fsx` | Core interfaces (`WriterFS`, `FileSystem`) and helper functions. |
| `osfs` | OS-backed filesystem confined to a root directory. |
| `contextual` | `context.Context`-aware interfaces and adapters. |
| `unionfs` | Union (Overlay) filesystem implementation. |
| `evictfs` | LRU/Size/Time-based eviction filesystem. |
| `bindfs` | Bind filesystem for remapping permissions/owners. |
| `mockfs` | Generated mocks for testing. |

## Requirements

- Go 1.25 or higher.

## License

MIT License
