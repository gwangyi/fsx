package contextual_test

import (
	"context"
	"errors"
	"io/fs"
	"syscall"
	"testing"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestMkdir(t *testing.T) {
	ctx := context.Background()

	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)
		mockDir.EXPECT().Mkdir(ctx, "foo", fs.FileMode(0755)).Return(nil).Times(1)

		err := contextual.Mkdir(ctx, mockDir, "foo", 0755)
		if err != nil {
			t.Fatalf("Mkdir failed: %v", err)
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := cmockfs.NewMockFS(ctrl)
		err := contextual.Mkdir(ctx, mockFS, "foo", 0755)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestMkdirAll(t *testing.T) {
	ctx := context.Background()

	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMkdirAll := cmockfs.NewMockMkdirAllFS(ctrl)
		mockMkdirAll.EXPECT().MkdirAll(ctx, "foo/bar", fs.FileMode(0755)).Return(nil).Times(1)

		err := contextual.MkdirAll(ctx, mockMkdirAll, "foo/bar", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithDirFS", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Fallback involves Stat (via Open if not StatFS) and Mkdir.
		// MkdirAll implementation checks Stat(name).
		// If not exist, checks parent.

		// Let's use a mock that implements DirFS but NOT MkdirAllFS.
		// And we need to support Stat. The Stat implementation in `stat.go` uses Open if StatFS is not implemented.
		// MkdirAll uses Stat.

		mockDir := cmockfs.NewMockDirFS(ctrl)

		// 1. Stat("foo/bar") -> Open("foo/bar") -> ErrNotExist
		mockDir.EXPECT().Open(ctx, "foo/bar").Return(nil, fs.ErrNotExist)

		// 2. Stat("foo") -> Open("foo") -> ErrNotExist
		mockDir.EXPECT().Open(ctx, "foo").Return(nil, fs.ErrNotExist)

		// 3. Stat(".") (from path.Dir("foo") -> ".") -> exists (implicit or explicit depending on impl)
		// Current implementation: parent := path.Dir(name). if parent != "." && parent != name ...
		// path.Dir("foo") is ".". So recursion stops.

		// 4. Mkdir("foo")
		mockDir.EXPECT().Mkdir(ctx, "foo", fs.FileMode(0755)).Return(nil)

		// 5. Mkdir("foo/bar")
		mockDir.EXPECT().Mkdir(ctx, "foo/bar", fs.FileMode(0755)).Return(nil)

		err := contextual.MkdirAll(ctx, mockDir, "foo/bar", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithExistingDir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)
		mockFileInfo := mockfs.NewMockFileInfo(ctrl)
		mockFileInfo.EXPECT().IsDir().Return(true).Times(1)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFile.EXPECT().Stat().Return(mockFileInfo, nil).Times(1)
		mockFile.EXPECT().Close().Return(nil).Times(1)

		// Stat("foo") via Open
		mockDir.EXPECT().Open(ctx, "foo").Return(mockFile, nil).Times(1)

		err := contextual.MkdirAll(ctx, mockDir, "foo", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithExistingFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)
		mockFileInfo := mockfs.NewMockFileInfo(ctrl)
		mockFileInfo.EXPECT().IsDir().Return(false).Times(1)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFile.EXPECT().Stat().Return(mockFileInfo, nil).Times(1)
		mockFile.EXPECT().Close().Return(nil).Times(1)

		mockDir.EXPECT().Open(ctx, "foo").Return(mockFile, nil).Times(1)

		err := contextual.MkdirAll(ctx, mockDir, "foo", 0755)
		// Check for wrapped syscall.ENOTDIR
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pathErr *fs.PathError
		if !errors.As(err, &pathErr) || pathErr.Err != syscall.ENOTDIR {
			t.Errorf("expected ENOTDIR, got %v", err)
		}
	})

	t.Run("Recursive creation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)

		mockDir.EXPECT().Open(ctx, "a/b").Return(nil, fs.ErrNotExist)
		mockDir.EXPECT().Open(ctx, "a").Return(nil, fs.ErrNotExist)

		mockDir.EXPECT().Mkdir(ctx, "a", fs.FileMode(0755)).Return(errors.New("mkdir fail"))

		if err := contextual.MkdirAll(ctx, mockDir, "a/b", 0755); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := cmockfs.NewMockFS(ctrl)
		// MkdirAll will try to Stat -> Open
		mockFS.EXPECT().Open(ctx, "foo").Return(nil, fs.ErrNotExist)

		err := contextual.MkdirAll(ctx, mockFS, "foo", 0755)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestReadDir(t *testing.T) {
	ctx := context.Background()

	t.Run("ReadDirFS supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockReadDirFS(ctrl)
		mfs.EXPECT().ReadDir(ctx, ".").Return([]fs.DirEntry{}, nil)

		if _, err := contextual.ReadDir(ctx, mfs, "."); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("fallback to Open success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockFS(ctrl)
		rdf := mockfs.NewMockReadDirFile(ctrl)
		rdf.EXPECT().Close().Return(nil)
		rdf.EXPECT().ReadDir(-1).Return([]fs.DirEntry{}, nil)

		mfs.EXPECT().Open(ctx, ".").Return(rdf, nil)

		_, err := contextual.ReadDir(ctx, mfs, ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fallback to Open success sorting", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockFS(ctrl)
		rdf := mockfs.NewMockReadDirFile(ctrl)
		rdf.EXPECT().Close().Return(nil)

		entryB := mockfs.NewMockDirEntry(ctrl)
		entryB.EXPECT().Name().Return("b").AnyTimes()
		entryA := mockfs.NewMockDirEntry(ctrl)
		entryA.EXPECT().Name().Return("a").AnyTimes()

		rdf.EXPECT().ReadDir(-1).Return([]fs.DirEntry{entryB, entryA}, nil)

		mfs.EXPECT().Open(ctx, ".").Return(rdf, nil)

		entries, err := contextual.ReadDir(ctx, mfs, ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 || entries[0].Name() != "a" || entries[1].Name() != "b" {
			t.Errorf("unexpected sorting: %v", entries)
		}
	})

	t.Run("fallback to Open not a directory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockFS(ctrl)
		mockFile := mockfs.NewMockFile(ctrl)
		mockFile.EXPECT().Close().Return(nil)

		mfs.EXPECT().Open(ctx, "file").Return(mockFile, nil)

		_, err := contextual.ReadDir(ctx, mfs, "file")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pErr *fs.PathError
		if !errors.As(err, &pErr) || pErr.Op != "readdir" || !errors.Is(pErr.Err, errors.ErrUnsupported) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback to Open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockFS(ctrl)
		mfs.EXPECT().Open(ctx, "missing").Return(nil, fs.ErrNotExist)

		_, err := contextual.ReadDir(ctx, mfs, "missing")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})
}
