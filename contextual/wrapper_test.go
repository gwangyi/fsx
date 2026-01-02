package contextual_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/gwangyi/fsx/contextual"
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
