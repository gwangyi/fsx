package contextual_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestRename(t *testing.T) {
	ctx := t.Context()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rfs := cmockfs.NewMockRenameFS(ctrl)
		rfs.EXPECT().Rename(ctx, "old", "new").Return(nil)

		if err := contextual.Rename(ctx, rfs, "old", "new"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("same file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		if err := contextual.Rename(ctx, m, "old", "old"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)
		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, newName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
			return 0, io.EOF
		})

		dst.EXPECT().Close().Return(nil)

		m.EXPECT().Remove(ctx, oldName).Return(nil)

		if err := contextual.Rename(ctx, m, oldName, newName); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("open error")

		m.EXPECT().Open(ctx, oldName).Return(nil, expectedErr)

		err := contextual.Rename(ctx, m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback stat error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedMode := fs.FileMode(0666)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		src.EXPECT().Stat().Return(nil, errors.New("stat error"))
		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, newName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, expectedMode).Return(dst, nil)

		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(nil)
		m.EXPECT().Remove(ctx, oldName).Return(nil)

		if err := contextual.Rename(ctx, m, oldName, newName); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fallback copy error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("copy error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)
		src.EXPECT().Close().Return(nil)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, newName, gomock.Any(), gomock.Any()).Return(dst, nil)
		src.EXPECT().Read(gomock.Any()).Return(0, expectedErr)
		dst.EXPECT().Close().Return(nil)

		err := contextual.Rename(ctx, m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback dst close error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("close error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)
		src.EXPECT().Close().Return(nil)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, newName, gomock.Any(), gomock.Any()).Return(dst, nil)
		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(expectedErr)

		err := contextual.Rename(ctx, m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback remove error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		oldName := "old"
		newName := "new"
		expectedErr := errors.New("remove error")
		mode := fs.FileMode(0644)

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Mode().Return(mode)
		src.EXPECT().Stat().Return(info, nil)
		src.EXPECT().Close().Return(nil).Times(2)

		dst := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, newName, gomock.Any(), gomock.Any()).Return(dst, nil)
		src.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		dst.EXPECT().Close().Return(nil)

		m.EXPECT().Remove(ctx, oldName).Return(expectedErr)

		err := contextual.Rename(ctx, m, oldName, newName)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("fallback OpenFile unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		oldName := "old"
		newName := "new"

		src := mockfs.NewMockFile(ctrl)
		m.EXPECT().Open(ctx, oldName).Return(src, nil)
		src.EXPECT().Stat().Return(nil, errors.New("stat error"))
		src.EXPECT().Close().Return(nil)

		err := contextual.Rename(ctx, m, oldName, newName)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
		var lErr *os.LinkError
		if !errors.As(err, &lErr) {
			t.Errorf("expected *os.LinkError, got %T", err)
		}
	})
}
