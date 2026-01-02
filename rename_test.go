package fsx_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestRename(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rfs := mockfs.NewMockRenameFS(ctrl)
		rfs.EXPECT().Rename("old", "new").Return(nil)

		if err := fsx.Rename(rfs, "old", "new"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("same file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		if err := fsx.Rename(m, "old", "old"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)
		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(newName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
			return 0, io.EOF
		})

		dst.EXPECT().Close().Return(nil)

		m.EXPECT().Remove(oldName).Return(nil)

		if err := fsx.Rename(m, oldName, newName); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("open error")

		m.EXPECT().Open(oldName).Return(nil, expectedErr)

		err := fsx.Rename(m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback stat error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedMode := fs.FileMode(0666) // Default mode if Stat fails

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)
		src.EXPECT().Stat().Return(nil, errors.New("stat error"))
		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(newName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, expectedMode).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(nil)
		m.EXPECT().Remove(oldName).Return(nil)

		if err := fsx.Rename(m, oldName, newName); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback dst create error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("create error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)

		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)

		src.EXPECT().Close().Return(nil) // Defer close

		m.EXPECT().OpenFile(newName, gomock.Any(), gomock.Any()).Return(nil, expectedErr)

		err := fsx.Rename(m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback copy error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("copy error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)

		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)

		src.EXPECT().Close().Return(nil) // Defer close

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(newName, gomock.Any(), gomock.Any()).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).Return(0, expectedErr)
		dst.EXPECT().Close().Return(nil)

		err := fsx.Rename(m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback dst close error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("close error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)

		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)

		src.EXPECT().Close().Return(nil) // Defer close

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(newName, gomock.Any(), gomock.Any()).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(expectedErr)

		err := fsx.Rename(m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback remove error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("remove error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(oldName).Return(src, nil)

		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)

		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(newName, gomock.Any(), gomock.Any()).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(nil)

		m.EXPECT().Remove(oldName).Return(expectedErr)

		err := fsx.Rename(m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
