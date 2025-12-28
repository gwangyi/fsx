package fsx_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestSymlink(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockSymlinkFS(ctrl)
		mockFS.EXPECT().Symlink("old", "new").Return(nil).Times(1)
		if err := fsx.Symlink(mockFS, "old", "new"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("symlink error")

		mockFS := mockfs.NewMockSymlinkFS(ctrl)
		mockFS.EXPECT().Symlink("error", "new").Return(expectedErr).Times(1)
		err := fsx.Symlink(mockFS, "error", "new")
		if err == nil {
			t.Error("expected error, got nil")
		} else if !errors.Is(err, expectedErr) {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		m := fstest.MapFS{}
		err := fsx.Symlink(m, "old", "new")
		if err == nil {
			t.Error("expected error, got nil")
		} else if !fsx.IsUnsupported(err) {
			t.Errorf("expected unsupported error, got %v", err)
		}
	})
}

func TestLchown(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockFS := mockfs.NewMockLchownFS(ctrl)
		mockFS.EXPECT().Lchown("name", "owner", "group").Return(nil).Times(1)
		if err := fsx.Lchown(mockFS, "name", "owner", "group"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("lchown error")

		mockFS := mockfs.NewMockLchownFS(ctrl)
		mockFS.EXPECT().Lchown("name", "owner", "group").Return(expectedErr).Times(1)
		err := fsx.Lchown(mockFS, "name", "owner", "group")
		if err == nil {
			t.Error("expected error, got nil")
		} else if !errors.Is(err, expectedErr) {
			t.Errorf("expected error, got %v", err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		m := fstest.MapFS{}
		err := fsx.Lchown(m, "name", "owner", "group")
		if err == nil {
			t.Error("expected error, got nil")
		} else if !fsx.IsUnsupported(err) {
			t.Errorf("expected unsupported error, got %v", err)
		}
	})
}
