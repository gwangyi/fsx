package contextual_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestToContextual_Open(t *testing.T) {
	mapFS := fstest.MapFS{
		"testfile": {Data: []byte("hello")},
	}

	ctxFS := contextual.ToContextual(mapFS)

	// Test successful Open
	f, err := ctxFS.Open(t.Context(), "testfile")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if stat.Name() != "testfile" {
		t.Errorf("Expected name 'testfile', got %s", stat.Name())
	}

	// Test Open error
	_, err = ctxFS.Open(t.Context(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Expected NotExist error, got %v", err)
	}
}

func TestToContextual_WriterFS(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "newfile"
		m.EXPECT().Create(name).Return(nil, nil)
		_, err := ctxFS.Create(t.Context(), name)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
	})

	t.Run("OpenFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "openfile"
		flag := 0
		mode := fs.FileMode(0644)
		m.EXPECT().OpenFile(name, flag, mode).Return(nil, nil)
		_, err := ctxFS.OpenFile(t.Context(), name, flag, mode)
		if err != nil {
			t.Errorf("OpenFile failed: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "oldfile"
		m.EXPECT().Remove(name).Return(nil)
		err := ctxFS.Remove(t.Context(), name)
		if err != nil {
			t.Errorf("Remove failed: %v", err)
		}
	})
}

func TestToContextual_RO(t *testing.T) {
	mapFS := fstest.MapFS{}
	ctxFS := contextual.ToContextual(mapFS).(contextual.WriterFS)

	_, err := ctxFS.Create(t.Context(), "foo")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Expected ErrUnsupported for Create on RO FS, got %v", err)
	}

	err = ctxFS.Remove(t.Context(), "foo")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Expected ErrUnsupported for Remove on RO FS, got %v", err)
	}
}
