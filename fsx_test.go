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

func TestFlags(t *testing.T) {
	nonRWFlags := os.O_APPEND | os.O_CREATE | os.O_EXCL | os.O_SYNC | os.O_TRUNC

	for _, tt := range []struct {
		name string
		flag int
	}{
		{name: "O_RDONLY", flag: os.O_RDONLY},
		{name: "O_WRONLY", flag: os.O_WRONLY},
		{name: "O_RDWR", flag: os.O_RDWR},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.flag|nonRWFlags)&fsx.O_ACCMODE != tt.flag {
				t.Errorf("Flag %s 0x%x is not compatible with O_ACCMODE 0x%x", tt.name, tt.flag, fsx.O_ACCMODE)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		m.EXPECT().Create(name).Return(nil, nil)

		_, err := fsx.Create(m, name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("create error")
		m.EXPECT().Create(name).Return(nil, expectedErr)

		_, err := fsx.Create(m, name)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pErr *fs.PathError
		if !errors.As(err, &pErr) {
			t.Errorf("expected *fs.PathError, got %T", err)
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error to be %v", expectedErr)
		}
		if pErr.Op != "open" || pErr.Path != name {
			t.Errorf("unexpected PathError content: %v", pErr)
		}
	})

	t.Run("supported with path error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("create error")
		m.EXPECT().Create(name).Return(nil, &fs.PathError{Err: expectedErr})

		_, err := fsx.Create(m, name)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pErr *fs.PathError
		if !errors.As(err, &pErr) {
			t.Errorf("expected *fs.PathError, got %T", err)
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error to be %v", expectedErr)
		}
		if pErr.Op != "open" || pErr.Path != name {
			t.Errorf("unexpected PathError content: %v", pErr)
		}
	})

	t.Run("unsupported fs.FS", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		_, err := fsx.Create(mapFS, "foo")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestOpenFile(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		flag := os.O_RDWR
		perm := fs.FileMode(0644)

		m.EXPECT().OpenFile(name, flag, perm).Return(nil, nil)

		_, err := fsx.OpenFile(m, name, flag, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		flag := os.O_RDWR
		perm := fs.FileMode(0644)
		expectedErr := errors.New("open error")

		m.EXPECT().OpenFile(name, flag, perm).Return(nil, expectedErr)

		_, err := fsx.OpenFile(m, name, flag, perm)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pErr *fs.PathError
		if !errors.As(err, &pErr) {
			t.Errorf("expected *fs.PathError, got %T", err)
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error to be %v", expectedErr)
		}
	})

	t.Run("unsupported fs.FS read-only fallback", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"foo": &fstest.MapFile{Data: []byte("test")},
		}

		f, err := fsx.OpenFile(mapFS, "foo", os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		defer func() { _ = f.Close() }()

		// Verify it is a read-only file
		if f != nil {
			_, err := f.Write([]byte("test"))
			if !errors.Is(err, fsx.ErrBadFileDescriptor) {
				t.Errorf("expected ErrBadFileDescriptor, got %v", err)
			}
			err = f.Truncate(0)
			if !errors.Is(err, fsx.ErrBadFileDescriptor) {
				t.Errorf("expected ErrBadFileDescriptor, got %v", err)
			}
		}
	})

	t.Run("unsupported fs.FS read-only fallback with error", func(t *testing.T) {
		mapFS := fstest.MapFS{}

		f, err := fsx.OpenFile(mapFS, "foo", os.O_RDONLY, 0)
		if err == nil {
			_ = f.Close()
			t.Error("OpenFile expected to fail, but succeeded")
		} else if !os.IsNotExist(err) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("unsupported fs.FS with write flag", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		_, err := fsx.OpenFile(mapFS, "foo", os.O_RDWR, 0)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		m.EXPECT().Remove(name).Return(nil)

		err := fsx.Remove(m, name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("remove error")
		m.EXPECT().Remove(name).Return(expectedErr)

		err := fsx.Remove(m, name)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pErr *fs.PathError
		if !errors.As(err, &pErr) {
			t.Errorf("expected *fs.PathError, got %T", err)
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected wrapped error to be %v", expectedErr)
		}
		if pErr.Op != "remove" {
			t.Errorf("expected Op to be 'remove', got %s", pErr.Op)
		}
	})

	t.Run("unsupported fs.FS", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		err := fsx.Remove(mapFS, "foo")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
