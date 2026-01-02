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

func TestWriteFile(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		m.EXPECT().WriteFile(ctx, name, data, perm).Return(nil)

		err := contextual.WriteFile(ctx, m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		expectedErr := errors.New("write error")
		m.EXPECT().WriteFile(ctx, name, data, perm).Return(expectedErr)

		err := contextual.WriteFile(ctx, m, name, data, perm)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("supported ErrUnsupported fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		m.EXPECT().WriteFile(ctx, name, data, perm).Return(errors.ErrUnsupported)

		// Fallback to OpenFile
		f := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(f, nil)
		f.EXPECT().Write(data).Return(len(data), nil)
		f.EXPECT().Close().Return(nil)

		err := contextual.WriteFile(ctx, m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fs.FS fallback success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)

		f := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(ctx, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(f, nil)
		f.EXPECT().Write(data).Return(len(data), nil)
		f.EXPECT().Close().Return(nil)

		err := contextual.WriteFile(ctx, m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fs.FS fallback unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl) // Doesn't implement WriterFS

		err := contextual.WriteFile(ctx, m, "foo", nil, 0644)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected error ErrUnsupported, got %v", err)
		}
	})

	t.Run("fs.FS fallback open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		expectedErr := errors.New("open error")

		m.EXPECT().OpenFile(ctx, name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(nil, expectedErr)

		err := contextual.WriteFile(ctx, m, name, data, perm)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
