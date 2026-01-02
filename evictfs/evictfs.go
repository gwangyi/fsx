// Package evictfs provides a contextual filesystem wrapper that automatically evicts files
// based on configurable limits such as maximum file count or total size.
package evictfs

import (
	"container/heap"
	"context"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gwangyi/fsx/contextual"
)

// Metadata represents the eviction-related metadata for a file.
// It allows custom eviction policies beyond simple LRU.
type Metadata interface {
	// Less returns true if this metadata has lower priority (should be evicted sooner)
	// than the other metadata.
	Less(other Metadata) bool
	// Update updates the metadata with new FileInfo when the file is accessed.
	Update(fi contextual.FileInfo)
	// Size returns the size of the file as tracked by this metadata.
	Size() int64
	// AccessTime returns the time when the file was last accessed.
	AccessTime() time.Time
}

// Config specifies the configuration for evictfs.
type Config struct {
	// MaxFiles is the maximum number of files to keep in the filesystem.
	// If 0, no limit is enforced based on file count.
	MaxFiles int
	// MaxSize is the maximum total size (in bytes) of all files in the filesystem.
	// If 0, no limit is enforced based on total size.
	MaxSize int64
	// MaxAge is the maximum age of a file in the filesystem.
	// Files older than this threshold (based on AccessTime) will be deleted on access.
	// If 0, no limit is enforced based on age.
	MaxAge time.Duration

	// Metadata is a factory function that creates a new Metadata instance
	// for a file when it is first discovered or created.
	// If nil, it defaults to an LRU policy.
	Metadata func(fi contextual.FileInfo) Metadata
}

// filesystem is a contextual filesystem that evicts files based on a threshold.
// It tracks file metadata in memory to determine which files should be removed
// when limits are reached.
type filesystem struct {
	fsys   contextual.FS
	config Config

	mu sync.Mutex
	// files maps file paths to their corresponding priority queue items.
	// TODO: consider replacing this map with a sorted map like a btree
	// to improve performance for prefix-based removals (e.g., in RemoveAll).
	files       map[string]*item
	pq          *priorityQueue
	currentSize int64

	evictSignal chan struct{}
}

// New creates a new evictfs instance wrapping the provided fsys.
// It initializes the internal state by walking the existing files in fsys.
func New(ctx context.Context, fsys contextual.FS, config Config) (contextual.FS, error) {
	if config.Metadata == nil {
		// Default to LRU if no priority function is provided.
		config.Metadata = newLRU
	}

	e := &filesystem{
		fsys:        fsys,
		config:      config,
		files:       make(map[string]*item),
		pq:          &priorityQueue{},
		evictSignal: make(chan struct{}, 1),
	}

	if err := e.init(ctx); err != nil {
		return nil, err
	}

	go e.evictLoop()

	return e, nil
}

// init scans the entire filesystem to build the initial priority queue and size tracking.
func (e *filesystem) init(ctx context.Context) error {
	fsys := contextual.FromContextual(e.fsys, ctx)
	return fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		extInfo := contextual.ExtendFileInfo(info)
		e.mu.Lock()
		e.addFileLocked(name, e.config.Metadata(extInfo))
		e.mu.Unlock()
		return nil
	})
}

// addFileLocked adds a file to the internal tracking state.
// It must be called with e.mu held.
func (e *filesystem) addFileLocked(name string, metadata Metadata) {
	it := &item{name: name, metadata: metadata}
	e.files[name] = it
	heap.Push(e.pq, it)
	e.currentSize += metadata.Size()
}

// removeFileLocked removes a file from the internal tracking state.
// It must be called with e.mu held.
func (e *filesystem) removeFileLocked(it *item) {
	heap.Remove(e.pq, it.index)
	delete(e.files, it.name)
	e.currentSize -= it.metadata.Size()
}

// touch updates the priority of a file because it was accessed or modified.
// If the file was not previously tracked, it is added.
// This method also triggers eviction if limits are exceeded.
func (e *filesystem) touch(ctx context.Context, name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	info, err := contextual.Stat(ctx, e.fsys, name)
	if err != nil {
		// If the file no longer exists or can't be stated, stop tracking it.
		if it, ok := e.files[name]; ok {
			e.removeFileLocked(it)
		}
		return
	}

	if info.IsDir() {
		return
	}

	if it, ok := e.files[name]; ok {
		// Update existing item.
		e.currentSize -= it.metadata.Size()
		it.metadata.Update(info)
		e.currentSize += it.metadata.Size()
		heap.Fix(e.pq, it.index)
	} else {
		// Add new item.
		md := e.config.Metadata(info)
		e.addFileLocked(name, md)
	}

	select {
	case e.evictSignal <- struct{}{}:
	default:
	}
}

// evictLoop runs in the background and processes eviction signals.
func (e *filesystem) evictLoop() {
	ctx := context.Background()
	for range e.evictSignal {
		for {
			var name string
			var metadata Metadata

			e.mu.Lock()
			if (e.config.MaxFiles > 0 && len(e.files) > e.config.MaxFiles) ||
				(e.config.MaxSize > 0 && e.currentSize > e.config.MaxSize) {
				// We expect the PQ to never be empty here because the loop condition
				// is based on tracked files.
				it := heap.Pop(e.pq).(*item)
				delete(e.files, it.name)
				e.currentSize -= it.metadata.Size()
				name = it.name
				metadata = it.metadata
			}
			e.mu.Unlock()

			if name == "" {
				break
			}

			_ = contextual.Remove(ctx, e.fsys, name)
			_ = metadata // metadata is popped, but we could use it if needed
		}
	}
}

// checkExpired checks if a file is expired and deletes it if it is.
func (e *filesystem) checkExpired(ctx context.Context, name string) error {
	if e.config.MaxAge <= 0 {
		return nil
	}
	e.mu.Lock()
	it, ok := e.files[name]
	if !ok || time.Since(it.metadata.AccessTime()) <= e.config.MaxAge {
		e.mu.Unlock()
		return nil
	}
	e.removeFileLocked(it)
	e.mu.Unlock()
	_ = contextual.Remove(ctx, e.fsys, name)
	return fs.ErrNotExist
}

// Open opens the named file for reading.
func (e *filesystem) Open(ctx context.Context, name string) (fs.File, error) {
	if err := e.checkExpired(ctx, name); err != nil {
		return nil, err
	}
	return e.OpenFile(ctx, name, os.O_RDONLY, 0)
}

// Create creates or truncates the named file.
func (e *filesystem) Create(ctx context.Context, name string) (contextual.File, error) {
	return e.OpenFile(ctx, name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// OpenFile is the generalized open call.
func (e *filesystem) OpenFile(ctx context.Context, name string, flag int, mode fs.FileMode) (contextual.File, error) {
	// If O_CREATE is not set, we should check expiration.
	// If O_CREATE is set, it might be an access to existing file or creating a new one.
	if flag&os.O_CREATE == 0 {
		if err := e.checkExpired(ctx, name); err != nil {
			return nil, err
		}
	}
	f, err := contextual.OpenFile(ctx, e.fsys, name, flag, mode)
	if err != nil {
		return nil, err
	}
	e.touch(ctx, name)
	return &evictFile{File: f, fs: e, name: name}, nil
}

// Remove removes the named file or (empty) directory.
func (e *filesystem) Remove(ctx context.Context, name string) error {
	err := contextual.Remove(ctx, e.fsys, name)
	if err == nil {
		e.mu.Lock()
		if it, ok := e.files[name]; ok {
			e.removeFileLocked(it)
		}
		e.mu.Unlock()
	}
	return err
}

// ReadFile reads the named file and returns its contents.
func (e *filesystem) ReadFile(ctx context.Context, name string) ([]byte, error) {
	if err := e.checkExpired(ctx, name); err != nil {
		return nil, err
	}
	data, err := contextual.ReadFile(ctx, e.fsys, name)
	if err == nil {
		e.touch(ctx, name)
	}
	return data, err
}

// Stat returns a FileInfo describing the named file.
func (e *filesystem) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	if err := e.checkExpired(ctx, name); err != nil {
		return nil, err
	}
	fi, err := contextual.Stat(ctx, e.fsys, name)
	if err == nil {
		e.touch(ctx, name)
	}
	return fi, err
}

// ReadDir reads the named directory and returns a list of directory entries.
func (e *filesystem) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	return contextual.ReadDir(ctx, e.fsys, name)
}

// Mkdir creates a new directory.
func (e *filesystem) Mkdir(ctx context.Context, name string, perm fs.FileMode) error {
	return contextual.Mkdir(ctx, e.fsys, name, perm)
}

// MkdirAll creates a directory and all necessary parents.
func (e *filesystem) MkdirAll(ctx context.Context, name string, perm fs.FileMode) error {
	return contextual.MkdirAll(ctx, e.fsys, name, perm)
}

// RemoveAll removes path and any children it contains.
func (e *filesystem) RemoveAll(ctx context.Context, name string) error {
	err := contextual.RemoveAll(ctx, e.fsys, name)
	if err == nil {
		e.mu.Lock()
		for p, it := range e.files {
			if p == name || strings.HasPrefix(p, name+"/") {
				e.removeFileLocked(it)
			}
		}
		e.mu.Unlock()
	}
	return err
}

// Rename renames a file.
func (e *filesystem) Rename(ctx context.Context, oldname, newname string) error {
	if err := e.checkExpired(ctx, oldname); err != nil {
		return err
	}
	err := contextual.Rename(ctx, e.fsys, oldname, newname)
	if err == nil {
		e.mu.Lock()
		if it, ok := e.files[oldname]; ok {
			e.removeFileLocked(it)
		}
		e.mu.Unlock()
		e.touch(ctx, newname)
	}
	return err
}

// Symlink creates a symbolic link.
func (e *filesystem) Symlink(ctx context.Context, oldname, newname string) error {
	err := contextual.Symlink(ctx, e.fsys, oldname, newname)
	if err == nil {
		e.touch(ctx, newname)
	}
	return err
}

// ReadLink returns the destination of the named symbolic link.
func (e *filesystem) ReadLink(ctx context.Context, name string) (string, error) {
	return contextual.ReadLink(ctx, e.fsys, name)
}

// Lstat returns a FileInfo describing the named file, without following links.
func (e *filesystem) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	if err := e.checkExpired(ctx, name); err != nil {
		return nil, err
	}
	fi, err := contextual.Lstat(ctx, e.fsys, name)
	if err == nil {
		e.touch(ctx, name)
	}
	return fi, err
}

// Lchown changes the owner and group of the named file, without following links.
func (e *filesystem) Lchown(ctx context.Context, name, owner, group string) error {
	if err := e.checkExpired(ctx, name); err != nil {
		return err
	}
	err := contextual.Lchown(ctx, e.fsys, name, owner, group)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// Truncate changes the size of the named file.
func (e *filesystem) Truncate(ctx context.Context, name string, size int64) error {
	if err := e.checkExpired(ctx, name); err != nil {
		return err
	}
	err := contextual.Truncate(ctx, e.fsys, name, size)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// WriteFile writes data to the named file.
func (e *filesystem) WriteFile(ctx context.Context, name string, data []byte, perm fs.FileMode) error {
	err := contextual.WriteFile(ctx, e.fsys, name, data, perm)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// Chown changes the owner and group of the named file.
func (e *filesystem) Chown(ctx context.Context, name, owner, group string) error {
	if err := e.checkExpired(ctx, name); err != nil {
		return err
	}
	err := contextual.Chown(ctx, e.fsys, name, owner, group)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// Chmod changes the mode of the named file.
func (e *filesystem) Chmod(ctx context.Context, name string, mode fs.FileMode) error {
	if err := e.checkExpired(ctx, name); err != nil {
		return err
	}
	err := contextual.Chmod(ctx, e.fsys, name, mode)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// Chtimes changes the access and modification times of the named file.
func (e *filesystem) Chtimes(ctx context.Context, name string, atime, ctime time.Time) error {
	if err := e.checkExpired(ctx, name); err != nil {
		return err
	}
	err := contextual.Chtimes(ctx, e.fsys, name, atime, ctime)
	if err == nil {
		e.touch(ctx, name)
	}
	return err
}

// evictFile wraps a contextual.File to track write and truncate operations.
type evictFile struct {
	contextual.File
	fs   *filesystem
	name string
}

// Write writes p to the file and touches it to update its eviction priority.
func (f *evictFile) Write(p []byte) (int, error) {
	n, err := f.File.Write(p)
	if n > 0 {
		f.fs.touch(context.Background(), f.name)
	}
	return n, err
}

// Truncate changes the size of the file and touches it.
func (f *evictFile) Truncate(size int64) error {
	err := f.File.Truncate(size)
	if err == nil {
		f.fs.touch(context.Background(), f.name)
	}
	return err
}

// item represents a tracked file in the priority queue.
type item struct {
	name     string
	metadata Metadata
	index    int // index in the priority queue (maintained by heap.Interface).
}

// priorityQueue implements heap.Interface to manage file eviction priority.
type priorityQueue struct {
	items []*item
}

func (pq *priorityQueue) Len() int           { return len(pq.items) }
func (pq *priorityQueue) Less(i, j int) bool { return pq.items[i].metadata.Less(pq.items[j].metadata) }
func (pq *priorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}
func (pq *priorityQueue) Push(x any) {
	it := x.(*item)
	it.index = len(pq.items)
	pq.items = append(pq.items, it)
}
func (pq *priorityQueue) Pop() any {
	n := len(pq.items)
	it := pq.items[n-1]
	it.index = -1
	pq.items = pq.items[:n-1]
	return it
}

var _ contextual.FileSystem = &filesystem{}
