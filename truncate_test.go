package fsx_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestTruncate(t *testing.T) {
	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTruncateFS := mockfs.NewMockTruncateFS(ctrl)
		mockTruncateFS.EXPECT().Truncate("foo", int64(100)).Return(nil).Times(1)

		err := fsx.Truncate(mockTruncateFS, "foo", 100)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}
	})

	t.Run("Fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockWriterFS(ctrl)
		mockFile := mockfs.NewMockFile(ctrl)

		mockFS.EXPECT().OpenFile("foo", os.O_WRONLY, fs.FileMode(0)).Return(mockFile, nil).Times(1)
		mockFile.EXPECT().Truncate(int64(100)).Return(nil).Times(1)
		mockFile.EXPECT().Close().Return(nil).Times(1)

		err := fsx.Truncate(mockFS, "foo", 100)
		if err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		m := fstest.MapFS{}
		err := fsx.Truncate(m, "foo", 100)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})

	t.Run("Fail OpenFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockWriterFS(ctrl)
		mockFS.EXPECT().OpenFile("foo", os.O_WRONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist).Times(1)

		err := fsx.Truncate(mockFS, "foo", 100)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})
}
