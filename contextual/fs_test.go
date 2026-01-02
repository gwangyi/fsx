package contextual_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
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

func TestExtendFileInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mockfs.NewMockFileInfo(ctrl)
	m.EXPECT().Name().Return("foo").AnyTimes()
	m.EXPECT().ModTime().Return(time.Now()).AnyTimes()
	m.EXPECT().Sys().Return(nil).AnyTimes()

	xfi := contextual.ExtendFileInfo(m)
	if xfi.Name() != "foo" {
		t.Errorf("expected name foo, got %q", xfi.Name())
	}
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := cmockfs.NewMockFS(ctrl)
	name := "foo"
	m.EXPECT().Open(ctx, name).Return(nil, nil)

	_, err := contextual.Open(ctx, m, name)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		m.EXPECT().Create(ctx, name).Return(nil, nil)

		_, err := contextual.Create(ctx, m, name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("create error")
		m.EXPECT().Create(ctx, name).Return(nil, expectedErr)

		_, err := contextual.Create(ctx, m, name)
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

		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("create error")
		m.EXPECT().Create(ctx, name).Return(nil, &fs.PathError{Err: expectedErr})

		_, err := contextual.Create(ctx, m, name)
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockFS(ctrl)
		_, err := contextual.Create(ctx, m, "foo")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestOpenFile(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		flag := os.O_RDWR
		perm := fs.FileMode(0644)

		m.EXPECT().OpenFile(ctx, name, flag, perm).Return(nil, nil)

		_, err := contextual.OpenFile(ctx, m, name, flag, perm)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		flag := os.O_RDWR
		perm := fs.FileMode(0644)
		expectedErr := errors.New("open error")

		m.EXPECT().OpenFile(ctx, name, flag, perm).Return(nil, expectedErr)

		_, err := contextual.OpenFile(ctx, m, name, flag, perm)
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		// Use mockfs.MockFS which simulates a basic FS (like fstest.MapFS but mocked)
		m := cmockfs.NewMockFS(ctrl)
		file := mockfs.NewMockFile(ctrl)
		file.EXPECT().Close().Return(nil)

		m.EXPECT().Open(ctx, "foo").Return(file, nil)

		f, err := contextual.OpenFile(ctx, m, "foo", os.O_RDONLY, 0)
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		m.EXPECT().Open(ctx, "foo").Return(nil, os.ErrNotExist)

		f, err := contextual.OpenFile(ctx, m, "foo", os.O_RDONLY, 0)
		if err == nil {
			_ = f.Close()
			t.Error("OpenFile expected to fail, but succeeded")
		} else if !os.IsNotExist(err) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("unsupported fs.FS with write flag", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		// We expect OpenFile to NOT be called on FS if it doesn't implement WriterFS,
		// and fallthrough to check flags. Since flags are RDWR, it should fail.

		_, err := contextual.OpenFile(ctx, m, "foo", os.O_RDWR, 0)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestRemove(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		m.EXPECT().Remove(ctx, name).Return(nil)

		err := contextual.Remove(ctx, m, name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockWriterFS(ctrl)
		name := "foo"
		expectedErr := errors.New("remove error")
		m.EXPECT().Remove(ctx, name).Return(expectedErr)

		err := contextual.Remove(ctx, m, name)
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockFS(ctrl)

		err := contextual.Remove(ctx, m, "foo")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

// Additional tests for other functions in contextualfs

func TestReadFile(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockReadFileFS(ctrl)
		name := "foo"
		content := []byte("hello")
		m.EXPECT().ReadFile(ctx, name).Return(content, nil)

		got, err := contextual.ReadFile(ctx, m, name)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("expected %q, got %q", content, got)
		}
	})

	t.Run("fallback to Open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		file := mockfs.NewMockFile(ctrl)
		content := []byte("hello")

		// ReadAll reads until EOF
		file.EXPECT().Read(gomock.Any()).DoAndReturn(func(p []byte) (int, error) {
			copy(p, content)
			return len(content), nil
		})
		file.EXPECT().Read(gomock.Any()).Return(0, io.EOF) // Return EOF on subsequent call
		file.EXPECT().Close().Return(nil)

		m.EXPECT().Open(ctx, "foo").Return(file, nil)

		got, err := contextual.ReadFile(ctx, m, "foo")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("expected %q, got %q", content, got)
		}
	})

	t.Run("fallback Open error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		m.EXPECT().Open(ctx, "missing").Return(nil, fs.ErrNotExist)

		_, err := contextual.ReadFile(ctx, m, "missing")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})
}
