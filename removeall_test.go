package fsx_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestRemoveAll(t *testing.T) {
	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockRemoveAllFS(ctrl)
		m.EXPECT().RemoveAll("foo").Return(nil)

		err := fsx.RemoveAll(m, "foo")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackFile", func(t *testing.T) {
		// RemoveAll on a file just calls Remove and succeeds
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockFS(ctrl)
		// Expectation: Remove("foo") -> nil
		m.EXPECT().Remove("foo").Return(nil)

		err := fsx.RemoveAll(m, "foo")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackNonEmptyDir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)

		// Create mock DirEntries
		file1 := mockfs.NewMockDirEntry(ctrl)
		file1.EXPECT().Name().Return("file1").AnyTimes()
		// RemoveAll checks IsDir? No, it calls RemoveAll recursively.
		// removeall.go:
		// entries, _ := fs.ReadDir(fsys, name)
		// for _, entry := range entries {
		//    childPath := path.Join(name, entry.Name())
		//    if err := RemoveAll(fsys, childPath); ...
		// }
		// So IsDir is not strictly called by RemoveAll logic, but it's good practice.
		// Actually, I don't need to mock IsDir unless RemoveAll uses it.
		// The current implementation of RemoveAll in removeall.go DOES NOT use IsDir from DirEntry.
		// It just constructs path and calls RemoveAll.

		subDir := mockfs.NewMockDirEntry(ctrl)
		subDir.EXPECT().Name().Return("sub").AnyTimes()

		// Sequence:
		// 1. Remove("dir") -> fails (e.g. not empty)
		mockDir.EXPECT().Remove("dir").Return(errors.New("not empty"))

		// 2. ReadDir("dir") -> [file1, sub]
		mockDir.EXPECT().ReadDir("dir").Return([]fs.DirEntry{file1, subDir}, nil)

		// 3. Loop:
		//    a. RemoveAll("dir/file1")
		//       - Remove("dir/file1") -> nil (success)
		mockDir.EXPECT().Remove("dir/file1").Return(nil)

		//    b. RemoveAll("dir/sub")
		//       - Remove("dir/sub") -> fail (simulate directory)
		mockDir.EXPECT().Remove("dir/sub").Return(errors.New("not empty"))

		//       - ReadDir("dir/sub") -> []
		mockDir.EXPECT().ReadDir("dir/sub").Return([]fs.DirEntry{}, nil)

		//       - Remove("dir/sub") -> nil (success empty dir)
		mockDir.EXPECT().Remove("dir/sub").Return(nil)

		// 4. Remove("dir") -> nil (success)
		mockDir.EXPECT().Remove("dir").Return(nil)

		err := fsx.RemoveAll(mockDir, "dir")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("FallbackReadDirError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)

		expectedErr := errors.New("initial remove error")
		mockDir.EXPECT().Remove("foo").Return(expectedErr)
		mockDir.EXPECT().ReadDir("foo").Return(nil, errors.New("readdir failed"))

		err := fsx.RemoveAll(mockDir, "foo")
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("FallbackRemoveError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)
		expectedErr := errors.New("generic remove error")

		// 1. First Remove fails
		mockDir.EXPECT().Remove("foo").Return(fs.ErrExist)

		// 2. ReadDir succeeds with entries
		mockEntry := mockfs.NewMockDirEntry(ctrl)
		mockEntry.EXPECT().Name().Return("bar").AnyTimes()
		mockDir.EXPECT().ReadDir("foo").Return([]fs.DirEntry{mockEntry}, nil)

		// 3. Remove child fails
		mockDir.EXPECT().Remove("foo/bar").Return(expectedErr)
		// 4. ReadDir on child fails, stopping recursion
		mockDir.EXPECT().ReadDir("foo/bar").Return(nil, errors.New("readdir failed"))

		err := fsx.RemoveAll(mockDir, "foo")
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("FallbackUnsupported", func(t *testing.T) {
		m := fstest.MapFS{}
		err := fsx.RemoveAll(m, "somepath")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
