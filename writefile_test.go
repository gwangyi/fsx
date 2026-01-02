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

func TestWriteFile(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		m.EXPECT().WriteFile(name, data, perm).Return(nil)

		err := fsx.WriteFile(m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		expectedErr := errors.New("write error")
		m.EXPECT().WriteFile(name, data, perm).Return(expectedErr)

		err := fsx.WriteFile(m, name, data, perm)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("supported ErrUnsupported fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriteFileFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		m.EXPECT().WriteFile(name, data, perm).Return(errors.ErrUnsupported)

		// Fallback to OpenFile
		f := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(f, nil)
		f.EXPECT().Write(data).Return(len(data), nil)
		f.EXPECT().Close().Return(nil)

		err := fsx.WriteFile(m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fs.FS fallback success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)

		f := mockfs.NewMockFile(ctrl)
		m.EXPECT().OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(f, nil)
		f.EXPECT().Write(data).Return(len(data), nil)
		f.EXPECT().Close().Return(nil)

		err := fsx.WriteFile(m, name, data, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fs.FS fallback readonly", func(t *testing.T) {
		m := fstest.MapFS{}

		err := fsx.WriteFile(m, "foo", nil, 0644)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected error ErrUnsupported, got %v", err)
		}
	})

	t.Run("fs.FS fallback open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		data := []byte("bar")
		perm := fs.FileMode(0644)
		expectedErr := errors.New("open error")

		m.EXPECT().OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm).Return(nil, expectedErr)

		err := fsx.WriteFile(m, name, data, perm)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
