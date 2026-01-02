package contextual_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestRemoveAll(t *testing.T) {
	ctx := context.Background()

	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockRemoveAllFS(ctrl)
		m.EXPECT().RemoveAll(ctx, "foo").Return(nil)

		err := contextual.RemoveAll(ctx, m, "foo")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		m.EXPECT().Remove(ctx, "foo").Return(nil)

		err := contextual.RemoveAll(ctx, m, "foo")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackNonEmptyDir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)

		file1 := mockfs.NewMockDirEntry(ctrl)
		file1.EXPECT().Name().Return("file1").AnyTimes()

		subDir := mockfs.NewMockDirEntry(ctrl)
		subDir.EXPECT().Name().Return("sub").AnyTimes()

		// 1. Remove("dir") -> fails (e.g. not empty)
		mockDir.EXPECT().Remove(ctx, "dir").Return(errors.New("not empty"))

		// 2. ReadDir("dir") -> [file1, sub]
		mockDir.EXPECT().ReadDir(ctx, "dir").Return([]fs.DirEntry{file1, subDir}, nil)

		// 3. RemoveAll("dir/file1") -> Remove("dir/file1") -> nil
		mockDir.EXPECT().Remove(ctx, "dir/file1").Return(nil)

		// 4. RemoveAll("dir/sub")
		//    - Remove("dir/sub") -> fail
		mockDir.EXPECT().Remove(ctx, "dir/sub").Return(errors.New("not empty"))
		//    - ReadDir("dir/sub") -> []
		mockDir.EXPECT().ReadDir(ctx, "dir/sub").Return([]fs.DirEntry{}, nil)
		//    - Remove("dir/sub") -> nil
		mockDir.EXPECT().Remove(ctx, "dir/sub").Return(nil)

		// 5. Remove("dir") -> nil
		mockDir.EXPECT().Remove(ctx, "dir").Return(nil)

		err := contextual.RemoveAll(ctx, mockDir, "dir")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackReadDirError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := cmockfs.NewMockDirFS(ctrl)

		removeErr := errors.New("is dir")
		readDirErr := errors.New("readdir failed")

		// Initial remove fails (is directory)
		mockDir.EXPECT().Remove(ctx, "foo").Return(removeErr)
		// ReadDir fails -> implementation returns the original Remove error
		mockDir.EXPECT().ReadDir(ctx, "foo").Return(nil, readDirErr)

		err := contextual.RemoveAll(ctx, mockDir, "foo")
		// We expect the original remove error
		if !errors.Is(err, removeErr) {
			t.Errorf("expected %v, got %v", removeErr, err)
		}
	})

	t.Run("FallbackNotExists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		m.EXPECT().Remove(ctx, "foo").Return(os.ErrNotExist)

		err := contextual.RemoveAll(ctx, m, "foo")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("Recursive removal child fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockDirFS(ctrl)

		mfs.EXPECT().Remove(ctx, "a").Return(errors.New("is dir"))

		entryB := mockfs.NewMockDirEntry(ctrl)
		entryB.EXPECT().Name().Return("b")
		mfs.EXPECT().ReadDir(ctx, "a").Return([]fs.DirEntry{entryB}, nil)

		// RemoveAll("a/b") -> Remove("a/b") fails
		mfs.EXPECT().Remove(ctx, "a/b").Return(errors.New("remove child fail"))
		// Then it calls ReadDir("a/b")
		mfs.EXPECT().ReadDir(ctx, "a/b").Return(nil, errors.New("readdir fail"))

		if err := contextual.RemoveAll(ctx, mfs, "a"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
