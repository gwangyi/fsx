package bindfs_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/bindfs"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestBindFS(t *testing.T) {
	ctx := context.Background()

	config := bindfs.Config{
		Owner:      bindfs.Static("alice"),
		Group:      bindfs.Static("users"),
		GrantPerm:  bindfs.Static(fs.FileMode(0100)),
		RevokePerm: bindfs.Static(fs.FileMode(0002)),
	}

	t.Run("Open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFS.EXPECT().OpenFile(ctx, "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(mockFile, nil)
		f, err := fsys.Open(ctx, "test.txt")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		if f == nil {
			t.Fatal("Open returned nil file")
		}

		mockFS.EXPECT().OpenFile(ctx, "error.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		_, err = fsys.Open(ctx, "error.txt")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Expected ErrNotExist, got %v", err)
		}
	})

	t.Run("Create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFS.EXPECT().Create(ctx, "new.txt").Return(mockFile, nil)
		f, err := fsys.Create(ctx, "new.txt")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if f == nil {
			t.Fatal("Create returned nil file")
		}

		mockFS.EXPECT().Create(ctx, "error.txt").Return(nil, fs.ErrPermission)
		_, err = fsys.Create(ctx, "error.txt")
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("Expected ErrPermission, got %v", err)
		}
	})

	t.Run("OpenFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFS.EXPECT().OpenFile(ctx, "open.txt", os.O_RDWR, fs.FileMode(0644)).Return(mockFile, nil)
		f, err := fsys.OpenFile(ctx, "open.txt", os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf("OpenFile failed: %v", err)
		}
		if f == nil {
			t.Fatal("OpenFile returned nil file")
		}

		mockFS.EXPECT().OpenFile(ctx, "error.txt", os.O_RDWR, fs.FileMode(0644)).Return(nil, fs.ErrPermission)
		_, err = fsys.OpenFile(ctx, "error.txt", os.O_RDWR, 0644)
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("Expected ErrPermission, got %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Remove(ctx, "remove.txt").Return(nil)
		err := fsys.Remove(ctx, "remove.txt")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	})

	t.Run("ReadFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().ReadFile(ctx, "read.txt").Return([]byte("hello"), nil)
		data, err := fsys.ReadFile(ctx, "read.txt")
		if err != nil {
			t.Fatalf("ReadFile failed: %v", err)
		}
		if string(data) != "hello" {
			t.Errorf("Expected hello, got %s", string(data))
		}
	})

	t.Run("Stat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFI := mockfs.NewMockFileInfo(ctrl)
		mockFI.EXPECT().Mode().Return(fs.FileMode(0666))
		mockFS.EXPECT().Stat(ctx, "stat.txt").Return(mockFI, nil)

		fi, err := fsys.Stat(ctx, "stat.txt")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		xfi := fi.(fsx.FileInfo)
		if xfi.Owner() != "alice" {
			t.Errorf("Expected owner alice, got %s", xfi.Owner())
		}
		if xfi.Group() != "users" {
			t.Errorf("Expected group users, got %s", xfi.Group())
		}
		if xfi.Mode() != 0764 {
			t.Errorf("Expected mode 0764, got %v", xfi.Mode())
		}

		mockFS.EXPECT().Stat(ctx, "error.txt").Return(nil, fs.ErrNotExist)
		_, err = fsys.Stat(ctx, "error.txt")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Expected ErrNotExist, got %v", err)
		}

		// Cover wrapFileInfo(nil)
		mockFS.EXPECT().Stat(ctx, "nil.txt").Return(nil, nil)
		fi, err = fsys.Stat(ctx, "nil.txt")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		if fi != nil {
			t.Errorf("Expected nil fi, got %v", fi)
		}
	})

	t.Run("Lstat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFI := mockfs.NewMockFileInfo(ctrl)
		mockFI.EXPECT().Mode().Return(fs.FileMode(0666))
		mockFS.EXPECT().Lstat(ctx, "link.txt").Return(mockFI, nil)

		fi, err := fsys.Lstat(ctx, "link.txt")
		if err != nil {
			t.Fatalf("Lstat failed: %v", err)
		}
		xfi := fi.(fsx.FileInfo)
		if xfi.Owner() != "alice" {
			t.Errorf("Expected owner alice, got %s", xfi.Owner())
		}
		if xfi.Group() != "users" {
			t.Errorf("Expected group users, got %s", xfi.Group())
		}
		if xfi.Mode() != 0764 {
			t.Errorf("Expected mode 0764, got %v", xfi.Mode())
		}

		mockFS.EXPECT().Lstat(ctx, "error.txt").Return(nil, fs.ErrNotExist)
		_, err = fsys.Lstat(ctx, "error.txt")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Expected ErrNotExist, got %v", err)
		}
	})

	t.Run("ReadDir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockDE := mockfs.NewMockDirEntry(ctrl)
		mockDE.EXPECT().Name().Return("entry").AnyTimes()
		mockFS.EXPECT().ReadDir(ctx, "dir").Return([]fs.DirEntry{mockDE}, nil)

		entries, err := fsys.ReadDir(ctx, "dir")
		if err != nil {
			t.Fatalf("ReadDir failed: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}
		if entries[0].Name() != "entry" {
			t.Errorf("Expected entry, got %s", entries[0].Name())
		}

		mockFI := mockfs.NewMockFileInfo(ctrl)
		mockFI.EXPECT().Mode().Return(fs.FileMode(0644))
		mockDE.EXPECT().Info().Return(mockFI, nil)

		fi, err := entries[0].Info()
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}
		xfi := fi.(fsx.FileInfo)
		if xfi.Owner() != "alice" {
			t.Errorf("Expected owner alice, got %s", xfi.Owner())
		}
		if xfi.Mode() != 0644|0100 {
			t.Errorf("Expected mode %v, got %v", 0644|0100, xfi.Mode())
		}

		mockDE.EXPECT().Info().Return(nil, fs.ErrPermission)
		_, err = entries[0].Info()
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("Expected ErrPermission, got %v", err)
		}

		mockFS.EXPECT().ReadDir(ctx, "error").Return(nil, fs.ErrNotExist)
		_, err = fsys.ReadDir(ctx, "error")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Expected ErrNotExist, got %v", err)
		}

		// Cover wrapDirEntry(nil)
		mockFS.EXPECT().ReadDir(ctx, "nil").Return([]fs.DirEntry{nil}, nil)
		entries, err = fsys.ReadDir(ctx, "nil")
		if err != nil {
			t.Fatalf("ReadDir failed: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("Expected 1 entry, got %d", len(entries))
		}
		if entries[0] != nil {
			t.Errorf("Expected nil entry, got %v", entries[0])
		}
	})

	t.Run("Mkdir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Mkdir(ctx, "dir", fs.FileMode(0755)).Return(nil)
		err := fsys.Mkdir(ctx, "dir", 0755)
		if err != nil {
			t.Fatalf("Mkdir failed: %v", err)
		}
	})

	t.Run("MkdirAll", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().MkdirAll(ctx, "a/b/c", fs.FileMode(0755)).Return(nil)
		err := fsys.MkdirAll(ctx, "a/b/c", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().RemoveAll(ctx, "dir").Return(nil)
		err := fsys.RemoveAll(ctx, "dir")
		if err != nil {
			t.Fatalf("RemoveAll failed: %v", err)
		}
	})

	t.Run("Rename", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Rename(ctx, "old", "new").Return(nil)
		err := fsys.Rename(ctx, "old", "new")
		if err != nil {
			t.Fatalf("Rename failed: %v", err)
		}
	})

	t.Run("Symlink", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Symlink(ctx, "target", "link").Return(nil)
		err := fsys.Symlink(ctx, "target", "link")
		if err != nil {
			t.Fatalf("Symlink failed: %v", err)
		}
	})

	t.Run("ReadLink", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().ReadLink(ctx, "link").Return("target", nil)
		target, err := fsys.ReadLink(ctx, "link")
		if err != nil {
			t.Fatalf("ReadLink failed: %v", err)
		}
		if target != "target" {
			t.Errorf("Expected target, got %s", target)
		}
	})

	t.Run("Lchown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Lchown(ctx, "link", "owner", "group").Return(nil)
		err := fsys.Lchown(ctx, "link", "owner", "group")
		if err != nil {
			t.Fatalf("Lchown failed: %v", err)
		}
	})

	t.Run("Truncate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Truncate(ctx, "file", int64(123)).Return(nil)
		err := fsys.Truncate(ctx, "file", 123)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().WriteFile(ctx, "file", []byte("data"), fs.FileMode(0644)).Return(nil)
		err := fsys.WriteFile(ctx, "file", []byte("data"), 0644)
		if err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	})

	t.Run("Chown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Chown(ctx, "file", "owner", "group").Return(nil)
		err := fsys.Chown(ctx, "file", "owner", "group")
		if err != nil {
			t.Fatalf("Chown failed: %v", err)
		}
	})

	t.Run("Chmod", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		mockFS.EXPECT().Chmod(ctx, "file", fs.FileMode(0644)).Return(nil)
		err := fsys.Chmod(ctx, "file", 0644)
		if err != nil {
			t.Fatalf("Chmod failed: %v", err)
		}
	})

	t.Run("Chtimes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)
		fsys := bindfs.New(mockFS, config)

		now := time.Now()
		mockFS.EXPECT().Chtimes(ctx, "file", now, now).Return(nil)
		err := fsys.Chtimes(ctx, "file", now, now)
		if err != nil {
			t.Fatalf("Chtimes failed: %v", err)
		}
	})

	t.Run("Context", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockFS := cmockfs.NewMockFileSystem(ctrl)

		type ctxKey struct{}
		myCtx := context.WithValue(context.Background(), ctxKey{}, "val")

		config := bindfs.Config{
			Owner: func(ctx context.Context, name string) string {
				if v := ctx.Value(ctxKey{}); v != nil {
					return v.(string)
				}
				return "nobody"
			},
		}
		fsys := bindfs.New(mockFS, config)

		mockFI := mockfs.NewMockFileInfo(ctrl)
		mockFS.EXPECT().Stat(myCtx, "test.txt").Return(mockFI, nil)

		fi, err := fsys.Stat(myCtx, "test.txt")
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		xfi := fi.(fsx.FileInfo)
		if xfi.Owner() != "val" {
			t.Errorf("Expected owner val, got %s", xfi.Owner())
		}
	})
}

func TestFileWrapper_Stat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFS := cmockfs.NewMockFileSystem(ctrl)
	mockFile := mockfs.NewMockFile(ctrl)
	mockFI := mockfs.NewMockFileInfo(ctrl)
	ctx := context.Background()

	config := bindfs.Config{
		Owner: bindfs.Static("alice"),
	}
	fsys := bindfs.New(mockFS, config)

	mockFS.EXPECT().OpenFile(ctx, "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(mockFile, nil)
	f, err := fsys.Open(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	mockFile.EXPECT().Stat().Return(mockFI, nil)
	mockFI.EXPECT().Mode().Return(fs.FileMode(0644))

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	xfi := fi.(fsx.FileInfo)
	if xfi.Owner() != "alice" {
		t.Errorf("Expected owner alice, got %s", xfi.Owner())
	}
	if xfi.Mode() != 0644 {
		t.Errorf("Expected mode 0644, got %v", xfi.Mode())
	}

	mockFile.EXPECT().Stat().Return(nil, fs.ErrPermission)
	_, err = f.Stat()
	if !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Expected ErrPermission, got %v", err)
	}
}

func TestFileInfo_NoOverrides(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFS := cmockfs.NewMockFileSystem(ctrl)
	mockFI := mockfs.NewMockFileInfo(ctrl)
	ctx := context.Background()

	fsys := bindfs.New(mockFS, bindfs.Config{})

	mockFS.EXPECT().Stat(ctx, "test.txt").Return(mockFI, nil)
	fi, err := fsys.Stat(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	xfi := fi.(fsx.FileInfo)
	mockFI.EXPECT().Owner().Return("bob")
	if xfi.Owner() != "bob" {
		t.Errorf("Expected owner bob, got %s", xfi.Owner())
	}

	mockFI.EXPECT().Group().Return("staff")
	if xfi.Group() != "staff" {
		t.Errorf("Expected group staff, got %s", xfi.Group())
	}

	mockFI.EXPECT().Mode().Return(fs.FileMode(0644))
	if xfi.Mode() != 0644 {
		t.Errorf("Expected mode 0644, got %v", xfi.Mode())
	}
}
