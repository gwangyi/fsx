package fsx_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestMkdir(t *testing.T) {
	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)
		mockDir.EXPECT().Mkdir("foo", fs.FileMode(0755)).Return(nil).Times(1)

		err := fsx.Mkdir(mockDir, "foo", 0755)
		if err != nil {
			t.Fatalf("Mkdir failed: %v", err)
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockWriterFS(ctrl) // This mock does not implement DirFS
		err := fsx.Mkdir(mockFS, "foo", 0755)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestMkdirAll(t *testing.T) {
	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMkdirAll := mockfs.NewMockMkdirAllFS(ctrl)
		mockMkdirAll.EXPECT().MkdirAll("foo/bar", fs.FileMode(0755)).Return(nil).Times(1)

		err := fsx.MkdirAll(mockMkdirAll, "foo/bar", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithDirFS", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)
		// Expectations for MkdirAll fallback logic:
		// 1. Stat(name) -> ErrNotExist (foo/bar)
		// 2. Stat(parent) -> ErrNotExist (foo)
		// 3. Mkdir(parent) (foo)
		// 4. Mkdir(name) (foo/bar)

		// Note: The specific order and calls might vary slightly depending on the exact implementation details of MkdirAll
		// and how it handles recursion. The current implementation in dir.go:
		// 1. Stat(name) (foo/bar)
		// 2. Recurse MkdirAll(parent) (foo)
		//    a. Stat(foo)
		//    b. Recurse MkdirAll(".") -> returns nil
		//    c. Mkdir(foo)
		// 3. Mkdir(foo/bar)

		// Mocking Stat via Open:
		mockDir.EXPECT().Open("foo/bar").Return(nil, fs.ErrNotExist).Times(1) // Check if foo/bar exists
		mockDir.EXPECT().Open("foo").Return(nil, fs.ErrNotExist).Times(1)     // Check if foo exists (recursion)

		gomock.InOrder(
			mockDir.EXPECT().Mkdir("foo", fs.FileMode(0755)).Return(nil),
			mockDir.EXPECT().Mkdir("foo/bar", fs.FileMode(0755)).Return(nil),
		)

		err := fsx.MkdirAll(mockDir, "foo/bar", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithExistingDir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)
		mockFileInfo := mockfs.NewMockFileInfo(ctrl)
		mockFileInfo.EXPECT().IsDir().Return(true).Times(1)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFile.EXPECT().Stat().Return(mockFileInfo, nil).Times(1)
		mockFile.EXPECT().Close().Return(nil).Times(1)

		mockDir.EXPECT().Open("foo").Return(mockFile, nil).Times(1)

		err := fsx.MkdirAll(mockDir, "foo", 0755)
		if err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
	})

	t.Run("FallbackWithExistingFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockDir := mockfs.NewMockDirFS(ctrl)
		mockFileInfo := mockfs.NewMockFileInfo(ctrl)
		mockFileInfo.EXPECT().IsDir().Return(false).Times(1)

		mockFile := mockfs.NewMockFile(ctrl)
		mockFile.EXPECT().Stat().Return(mockFileInfo, nil).Times(1)
		mockFile.EXPECT().Close().Return(nil).Times(1)

		mockDir.EXPECT().Open("foo").Return(mockFile, nil).Times(1)

		err := fsx.MkdirAll(mockDir, "foo", 0755)
		if !errors.Is(err, fsx.ErrNotDir) {
			t.Errorf("expected ErrNotDir, got %v", err)
		}
	})

	t.Run("FallbackWithFailingMkdir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("mkdir error")

		mockDir := mockfs.NewMockDirFS(ctrl)
		mockDir.EXPECT().Open("foo").Return(nil, fs.ErrNotExist).Times(1)
		mockDir.EXPECT().Open("foo/bar").Return(nil, fs.ErrNotExist).Times(1)
		mockDir.EXPECT().Mkdir("foo", fs.FileMode(0755)).Return(expectedErr).Times(1)

		err := fsx.MkdirAll(mockDir, "foo/bar", 0755)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected expectedErr, got %v", err)
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockWriterFS(ctrl) // This mock does not implement DirFS
		// MkdirAll will try to Stat the path to see if it exists
		mockFS.EXPECT().Open("foo").Return(nil, fs.ErrNotExist)

		err := fsx.MkdirAll(mockFS, "foo", 0755)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
