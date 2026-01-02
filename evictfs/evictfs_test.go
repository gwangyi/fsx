package evictfs_test

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/evictfs"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func newMockFileInfo(ctrl *gomock.Controller, name string, size int64, atime time.Time) *mockfs.MockFileInfo {
	info := mockfs.NewMockFileInfo(ctrl)
	info.EXPECT().Name().Return(name).AnyTimes()
	info.EXPECT().Size().Return(size).AnyTimes()
	info.EXPECT().AccessTime().Return(atime).AnyTimes()
	info.EXPECT().IsDir().Return(false).AnyTimes()
	return info
}

func setupExpiredFile(ctrl *gomock.Controller, m *cmockfs.MockFileSystem, ctx context.Context, fsys contextual.FS, name string) {
	oldTime := time.Now().Add(-2 * time.Hour)
	info := newMockFileInfo(ctrl, name, 10, oldTime)
	m.EXPECT().OpenFile(gomock.Any(), name, gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), name).Return(info, nil)
	_, _ = contextual.Create(ctx, fsys, name)
}

func TestFilesystem_EvictMaxFiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	// Initial walk: empty
	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)

	fsys, err := evictfs.New(ctx, m, evictfs.Config{
		MaxFiles: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add file 1
	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)

	_, err = contextual.Create(ctx, fsys, "file1")
	if err != nil {
		t.Fatal(err)
	}

	// Add file 2
	info2 := newMockFileInfo(ctrl, "file2", 10, time.Now().Add(time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file2", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file2").Return(info2, nil)

	_, err = contextual.Create(ctx, fsys, "file2")
	if err != nil {
		t.Fatal(err)
	}

	// Add file 3, should evict file 1
	info3 := newMockFileInfo(ctrl, "file3", 10, time.Now().Add(2*time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file3", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file3").Return(info3, nil)
	m.EXPECT().Remove(gomock.Any(), "file1").Return(nil)

	_, err = contextual.Create(ctx, fsys, "file3")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestFilesystem_Touch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	// Initial walk: empty
	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)

	fsys, err := evictfs.New(ctx, m, evictfs.Config{
		MaxFiles: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add file 1
	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "file1")

	// Add file 2
	info2 := newMockFileInfo(ctrl, "file2", 10, time.Now().Add(time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file2", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file2").Return(info2, nil)
	_, _ = contextual.Create(ctx, fsys, "file2")

	// Touch file 1 (update atime)
	info1Updated := newMockFileInfo(ctrl, "file1", 10, time.Now().Add(2*time.Second))
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1Updated, nil).Times(2)
	_, _ = contextual.Stat(ctx, fsys, "file1")

	// Add file 3, should evict file 2 because file 1 was touched
	info3 := newMockFileInfo(ctrl, "file3", 10, time.Now().Add(3*time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file3", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file3").Return(info3, nil)
	m.EXPECT().Remove(gomock.Any(), "file2").Return(nil)

	_, err = contextual.Create(ctx, fsys, "file3")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestFilesystem_EvictMaxSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	// Initial walk: empty
	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)

	fsys, err := evictfs.New(ctx, m, evictfs.Config{
		MaxSize: 20,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add file 1 (15 bytes)
	info1 := newMockFileInfo(ctrl, "file1", 15, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)

	_, err = contextual.Create(ctx, fsys, "file1")
	if err != nil {
		t.Fatal(err)
	}

	// Add file 2 (10 bytes), should evict file 1
	info2 := newMockFileInfo(ctrl, "file2", 10, time.Now().Add(time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file2", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file2").Return(info2, nil)
	m.EXPECT().Remove(gomock.Any(), "file1").Return(nil)

	_, err = contextual.Create(ctx, fsys, "file2")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestFilesystem_Init(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	// Initial walk: 2 files
	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	info2 := newMockFileInfo(ctrl, "file2", 10, time.Now().Add(time.Second))

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	de1 := mockfs.NewMockDirEntry(ctrl)
	de1.EXPECT().Name().Return("file1").AnyTimes()
	de1.EXPECT().IsDir().Return(false).AnyTimes()
	de1.EXPECT().Info().Return(info1, nil).AnyTimes()
	de2 := mockfs.NewMockDirEntry(ctrl)
	de2.EXPECT().Name().Return("file2").AnyTimes()
	de2.EXPECT().IsDir().Return(false).AnyTimes()
	de2.EXPECT().Info().Return(info2, nil).AnyTimes()

	m.EXPECT().ReadDir(gomock.Any(), ".").Return([]fs.DirEntry{de1, de2}, nil)

	fsys, err := evictfs.New(ctx, m, evictfs.Config{
		MaxFiles: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add file 3, should evict file 1
	info3 := newMockFileInfo(ctrl, "file3", 10, time.Now().Add(2*time.Second))
	m.EXPECT().OpenFile(gomock.Any(), "file3", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file3").Return(info3, nil)
	m.EXPECT().Remove(gomock.Any(), "file1").Return(nil)

	_, err = contextual.Create(ctx, fsys, "file3")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
}

func TestFilesystem_Delegation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	m.EXPECT().Mkdir(gomock.Any(), "dir", fs.FileMode(0755)).Return(nil)
	_ = contextual.Mkdir(ctx, fsys, "dir", 0755)

	m.EXPECT().MkdirAll(gomock.Any(), "a/b/c", fs.FileMode(0755)).Return(nil)
	_ = contextual.MkdirAll(ctx, fsys, "a/b/c", 0755)

	m.EXPECT().RemoveAll(gomock.Any(), "dir").Return(nil)
	_ = contextual.RemoveAll(ctx, fsys, "dir")

	m.EXPECT().ReadDir(gomock.Any(), "dir").Return(nil, nil)
	_, _ = contextual.ReadDir(ctx, fsys, "dir")

	m.EXPECT().ReadLink(gomock.Any(), "link").Return("target", nil)
	_, _ = contextual.ReadLink(ctx, fsys, "link")

	m.EXPECT().Lstat(gomock.Any(), "link").Return(newMockFileInfo(ctrl, "link", 0, time.Now()), nil)
	m.EXPECT().Stat(gomock.Any(), "link").Return(newMockFileInfo(ctrl, "link", 0, time.Now()), nil)
	_, _ = contextual.Lstat(ctx, fsys, "link")

	m.EXPECT().ReadFile(gomock.Any(), "file").Return([]byte("data"), nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_, _ = contextual.ReadFile(ctx, fsys, "file")

	m.EXPECT().WriteFile(gomock.Any(), "file", []byte("data"), fs.FileMode(0644)).Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_ = contextual.WriteFile(ctx, fsys, "file", []byte("data"), 0644)

	m.EXPECT().Rename(gomock.Any(), "old", "new").Return(nil)
	m.EXPECT().Stat(gomock.Any(), "new").Return(newMockFileInfo(ctrl, "new", 10, time.Now()), nil)
	_ = contextual.Rename(ctx, fsys, "old", "new")

	m.EXPECT().RemoveAll(gomock.Any(), "dir").Return(nil)
	_ = contextual.RemoveAll(ctx, fsys, "dir")

	m.EXPECT().Symlink(gomock.Any(), "old", "new").Return(nil)
	m.EXPECT().Stat(gomock.Any(), "new").Return(newMockFileInfo(ctrl, "new", 0, time.Now()), nil)
	_ = contextual.Symlink(ctx, fsys, "old", "new")

	m.EXPECT().Lchown(gomock.Any(), "file", "owner", "group").Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_ = contextual.Lchown(ctx, fsys, "file", "owner", "group")

	m.EXPECT().Truncate(gomock.Any(), "file", int64(5)).Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 5, time.Now()), nil)
	_ = contextual.Truncate(ctx, fsys, "file", 5)

	m.EXPECT().Chown(gomock.Any(), "file", "owner", "group").Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_ = contextual.Chown(ctx, fsys, "file", "owner", "group")

	m.EXPECT().Chmod(gomock.Any(), "file", fs.FileMode(0644)).Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_ = contextual.Chmod(ctx, fsys, "file", 0644)

	m.EXPECT().Chtimes(gomock.Any(), "file", gomock.Any(), gomock.Any()).Return(nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_ = contextual.Chtimes(ctx, fsys, "file", time.Now(), time.Now())

	m.EXPECT().OpenFile(gomock.Any(), "file", os.O_RDONLY, fs.FileMode(0)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file").Return(newMockFileInfo(ctrl, "file", 10, time.Now()), nil)
	_, _ = contextual.Open(ctx, fsys, "file")
}

func TestFilesystem_Touch_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	// Add file 1
	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "file1")

	// Touch with error (stat returns error)
	// contextual.Stat calls m.Stat, which fails.
	// Then it falls back to Open, which also fails.
	m.EXPECT().Stat(gomock.Any(), "file1").Return(nil, fs.ErrNotExist).AnyTimes()
	m.EXPECT().Open(gomock.Any(), "file1").Return(nil, fs.ErrNotExist).AnyTimes()
	_, _ = contextual.Stat(ctx, fsys, "file1")

	// Touch a directory (should skip tracking)
	dirInfo := mockfs.NewMockFileInfo(ctrl)
	dirInfo.EXPECT().IsDir().Return(true).AnyTimes()
	// contextual.Stat calls m.Stat, then touch calls contextual.Stat which calls m.Stat again.
	m.EXPECT().Stat(gomock.Any(), "dir").Return(dirInfo, nil).AnyTimes()
	_, _ = contextual.Stat(ctx, fsys, "dir")
}

func TestFilesystem_RemoveAll_Recursive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	// Add file in sub dir
	info1 := newMockFileInfo(ctrl, "dir/file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "dir/file1", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "dir/file1").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "dir/file1")

	m.EXPECT().RemoveAll(gomock.Any(), "dir").Return(nil)
	_ = contextual.RemoveAll(ctx, fsys, "dir")
}

func TestFilesystem_Init_Extra(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)

	dirDe := mockfs.NewMockDirEntry(ctrl)
	dirDe.EXPECT().Name().Return("dir").AnyTimes()
	dirDe.EXPECT().IsDir().Return(true).AnyTimes()

	m.EXPECT().ReadDir(gomock.Any(), ".").Return([]fs.DirEntry{dirDe}, nil)
	m.EXPECT().ReadDir(gomock.Any(), "dir").Return(nil, nil)

	_, err := evictfs.New(ctx, m, evictfs.Config{
		MaxSize: 1, // small size to trigger evict if anything added
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_Errors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	// New error (init fails)
	m.EXPECT().Stat(gomock.Any(), ".").Return(nil, fs.ErrPermission)
	_, err := evictfs.New(ctx, m, evictfs.Config{})
	if err == nil {
		t.Error("expected error during New")
	}

	// Setup for subsequent tests
	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	// OpenFile error
	m.EXPECT().OpenFile(gomock.Any(), "err", gomock.Any(), gomock.Any()).Return(nil, fs.ErrPermission)
	_, err = contextual.OpenFile(ctx, fsys, "err", os.O_RDWR, 0666)
	if err == nil {
		t.Error("expected error during OpenFile")
	}

	// Rename untracked file
	m.EXPECT().Rename(gomock.Any(), "old", "new").Return(nil)
	// touch("new") calls contextual.Stat which calls m.Stat.
	// Since "new" is not tracked by evictfs yet, touch("new") will just add it.
	m.EXPECT().Stat(gomock.Any(), "new").Return(newMockFileInfo(ctrl, "new", 10, time.Now()), nil)
	_ = contextual.Rename(ctx, fsys, "old", "new")
}

func TestFilesystem_OnAccessExpiration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)

	// Threshold: 1 hour. We'll add a file with access time 2 hours ago.
	threshold := time.Hour
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{
		MaxAge: threshold,
	})

	// Stat
	setupExpiredFile(ctrl, m, ctx, fsys, "expired")
	m.EXPECT().Remove(gomock.Any(), "expired").Return(nil)
	_, err := contextual.Stat(ctx, fsys, "expired")
	if !os.IsNotExist(err) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}

	// Open
	setupExpiredFile(ctrl, m, ctx, fsys, "open")
	m.EXPECT().Remove(gomock.Any(), "open").Return(nil)
	_, err = contextual.Open(ctx, fsys, "open")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Open")
	}

	// Create (should succeed regardless of age)
	setupExpiredFile(ctrl, m, ctx, fsys, "create")
	m.EXPECT().OpenFile(gomock.Any(), "create", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "create").Return(newMockFileInfo(ctrl, "create", 10, time.Now()), nil)
	_, err = contextual.Create(ctx, fsys, "create")
	if err != nil {
		t.Errorf("expected success for Create on expired file, got %v", err)
	}

	// ReadFile
	setupExpiredFile(ctrl, m, ctx, fsys, "readfile")
	m.EXPECT().Remove(gomock.Any(), "readfile").Return(nil)
	_, err = contextual.ReadFile(ctx, fsys, "readfile")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for ReadFile")
	}

	// Rename
	setupExpiredFile(ctrl, m, ctx, fsys, "rename")
	m.EXPECT().Remove(gomock.Any(), "rename").Return(nil)
	err = contextual.Rename(ctx, fsys, "rename", "new")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Rename")
	}

	// Lstat
	setupExpiredFile(ctrl, m, ctx, fsys, "lstat")
	m.EXPECT().Remove(gomock.Any(), "lstat").Return(nil)
	_, err = contextual.Lstat(ctx, fsys, "lstat")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Lstat")
	}

	// Lchown
	setupExpiredFile(ctrl, m, ctx, fsys, "lchown")
	m.EXPECT().Remove(gomock.Any(), "lchown").Return(nil)
	err = contextual.Lchown(ctx, fsys, "lchown", "o", "g")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Lchown")
	}

	// Truncate
	setupExpiredFile(ctrl, m, ctx, fsys, "truncate")
	m.EXPECT().Remove(gomock.Any(), "truncate").Return(nil)
	err = contextual.Truncate(ctx, fsys, "truncate", 0)
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Truncate")
	}

	// WriteFile (should succeed regardless of age)
	setupExpiredFile(ctrl, m, ctx, fsys, "writefile")
	m.EXPECT().WriteFile(gomock.Any(), "writefile", gomock.Any(), gomock.Any()).Return(nil)
	m.EXPECT().Stat(gomock.Any(), "writefile").Return(newMockFileInfo(ctrl, "writefile", 10, time.Now()), nil)
	err = contextual.WriteFile(ctx, fsys, "writefile", nil, 0)
	if err != nil {
		t.Errorf("expected success for WriteFile on expired file, got %v", err)
	}

	// Chown
	setupExpiredFile(ctrl, m, ctx, fsys, "chown")
	m.EXPECT().Remove(gomock.Any(), "chown").Return(nil)
	err = contextual.Chown(ctx, fsys, "chown", "o", "g")
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Chown")
	}

	// Chmod
	setupExpiredFile(ctrl, m, ctx, fsys, "chmod")
	m.EXPECT().Remove(gomock.Any(), "chmod").Return(nil)
	err = contextual.Chmod(ctx, fsys, "chmod", 0)
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Chmod")
	}

	// Chtimes
	setupExpiredFile(ctrl, m, ctx, fsys, "chtimes")
	m.EXPECT().Remove(gomock.Any(), "chtimes").Return(nil)
	err = contextual.Chtimes(ctx, fsys, "chtimes", time.Now(), time.Now())
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for Chtimes")
	}
}

func TestFilesystem_checkExpired_Coverage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil).AnyTimes()
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil).AnyTimes()

	fsys, _ := evictfs.New(ctx, m, evictfs.Config{
		MaxAge: time.Hour,
	})

	// Case 1: checkExpired with untracked file (!ok branch)
	m.EXPECT().Stat(gomock.Any(), "untracked").Return(newMockFileInfo(ctrl, "untracked", 10, time.Now()), nil).AnyTimes()
	_, _ = contextual.Stat(ctx, fsys, "untracked")

	// Case 2: checkExpired with tracked but NOT expired file
	info := newMockFileInfo(ctrl, "fresh", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "fresh", gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), "fresh").Return(info, nil).AnyTimes()
	_, _ = contextual.Create(ctx, fsys, "fresh")

	_, _ = contextual.Stat(ctx, fsys, "fresh")
}

func TestFilesystem_OpenFile_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)

	threshold := time.Hour
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{MaxAge: threshold})

	// Case 1: OpenFile with O_CREATE on a new file (branch flag&os.O_CREATE == 0 is false)
	info := newMockFileInfo(ctrl, "newfile", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "newfile", os.O_RDWR|os.O_CREATE, fs.FileMode(0666)).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "newfile").Return(info, nil)
	_, _ = contextual.OpenFile(ctx, fsys, "newfile", os.O_RDWR|os.O_CREATE, 0666)

	// Case 2: OpenFile WITHOUT O_CREATE on a non-existent/expired file (branch flag&os.O_CREATE == 0 is true)
	// (Already covered by TestFilesystem_OnAccessExpiration, but let's be explicit here if needed)
	setupExpiredFile(ctrl, m, ctx, fsys, "notcreate")
	m.EXPECT().Remove(gomock.Any(), "notcreate").Return(nil)
	_, err := contextual.OpenFile(ctx, fsys, "notcreate", os.O_RDWR, 0666)
	if !os.IsNotExist(err) {
		t.Error("expected ErrNotExist for OpenFile without O_CREATE on expired file")
	}
}

func TestFilesystem_Coverage_Init_InfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)

	de := mockfs.NewMockDirEntry(ctrl)
	de.EXPECT().Name().Return("file").AnyTimes()
	de.EXPECT().IsDir().Return(false).AnyTimes()
	de.EXPECT().Info().Return(nil, fs.ErrPermission).AnyTimes()

	m.EXPECT().ReadDir(gomock.Any(), ".").Return([]fs.DirEntry{de}, nil)

	_, err := evictfs.New(ctx, m, evictfs.Config{})
	if err == nil {
		t.Error("expected error from New when d.Info() fails")
	}
}

func TestFilesystem_Coverage_Touch_TrackedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	// Add file
	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "file1")

	// ReadFile success but touch fails due to Stat error
	m.EXPECT().ReadFile(gomock.Any(), "file1").Return([]byte("data"), nil)
	// touch calls Stat
	m.EXPECT().Stat(gomock.Any(), "file1").Return(nil, fs.ErrNotExist)
	_, _ = contextual.ReadFile(ctx, fsys, "file1")
}

func TestFilesystem_Coverage_RemoveAll_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	m.EXPECT().RemoveAll(gomock.Any(), "dir").Return(fs.ErrPermission)
	err := contextual.RemoveAll(ctx, fsys, "dir")
	if err == nil {
		t.Error("expected error from RemoveAll")
	}
}

func TestFilesystem_Coverage_Rename_Tracked(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{})

	// Add old file
	info1 := newMockFileInfo(ctrl, "old", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "old", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "old").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "old")

	// Rename tracked file
	m.EXPECT().Rename(gomock.Any(), "old", "new").Return(nil)
	// touch("new")
	m.EXPECT().Stat(gomock.Any(), "new").Return(newMockFileInfo(ctrl, "new", 10, time.Now()), nil)
	_ = contextual.Rename(ctx, fsys, "old", "new")
}

func TestFilesystem_Remove(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{MaxFiles: 1})

	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file1", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)
	_, _ = contextual.Create(ctx, fsys, "file1")

	m.EXPECT().Remove(gomock.Any(), "file1").Return(nil)
	_ = contextual.Remove(ctx, fsys, "file1")

	// Add another file, file 1 should not be evicted (already removed)
	info2 := newMockFileInfo(ctrl, "file2", 10, time.Now())
	m.EXPECT().OpenFile(gomock.Any(), "file2", gomock.Any(), gomock.Any()).Return(nil, nil)
	m.EXPECT().Stat(gomock.Any(), "file2").Return(info2, nil)
	_, _ = contextual.Create(ctx, fsys, "file2")
}

func TestFilesystem_FileOps(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFileSystem(ctrl)
	ctx := context.Background()

	dot := mockfs.NewMockFileInfo(ctrl)
	dot.EXPECT().IsDir().Return(true).AnyTimes()
	m.EXPECT().Stat(gomock.Any(), ".").Return(dot, nil)
	m.EXPECT().ReadDir(gomock.Any(), ".").Return(nil, nil)
	fsys, _ := evictfs.New(ctx, m, evictfs.Config{MaxFiles: 1})

	info1 := newMockFileInfo(ctrl, "file1", 10, time.Now())
	mf1 := mockfs.NewMockFile(ctrl)
	m.EXPECT().OpenFile(gomock.Any(), "file1", gomock.Any(), gomock.Any()).Return(mf1, nil)
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1, nil)
	f, _ := contextual.Create(ctx, fsys, "file1")

	// Write should touch
	info1Updated := newMockFileInfo(ctrl, "file1", 20, time.Now().Add(time.Second))
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1Updated, nil)
	mf1.EXPECT().Write(gomock.Any()).Return(5, nil)
	_, _ = f.Write([]byte("hello"))

	// Truncate should touch
	info1Updated2 := newMockFileInfo(ctrl, "file1", 5, time.Now().Add(2*time.Second))
	m.EXPECT().Stat(gomock.Any(), "file1").Return(info1Updated2, nil)
	mf1.EXPECT().Truncate(gomock.Any()).Return(nil)
	_ = f.Truncate(5)
}
