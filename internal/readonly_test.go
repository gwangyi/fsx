package internal_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/gwangyi/fsx/internal"
)

// mockFSFile implements fs.File for testing purposes.
type mockFSFile struct {
	statFunc  func() (fs.FileInfo, error)
	readFunc  func([]byte) (int, error)
	closeFunc func() error
}

func (m *mockFSFile) Stat() (fs.FileInfo, error) {
	if m.statFunc != nil {
		return m.statFunc()
	}
	return nil, nil
}

func (m *mockFSFile) Read(b []byte) (int, error) {
	if m.readFunc != nil {
		return m.readFunc(b)
	}
	return 0, io.EOF
}

func (m *mockFSFile) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestReadOnlyFile_Write(t *testing.T) {
	f := internal.ReadOnlyFile{File: &mockFSFile{}}
	_, err := f.Write([]byte("test"))
	if !errors.Is(err, internal.ErrBadFileDescriptor) {
		t.Errorf("expected ErrBadFileDescriptor, got %v", err)
	}
}

func TestReadOnlyFile_Truncate(t *testing.T) {
	f := internal.ReadOnlyFile{File: &mockFSFile{}}
	err := f.Truncate(10)
	if !errors.Is(err, internal.ErrBadFileDescriptor) {
		t.Errorf("expected ErrBadFileDescriptor, got %v", err)
	}
}

func TestReadOnlyFile_Delegation(t *testing.T) {
	readCalled := false
	closeCalled := false
	statCalled := false

	mock := &mockFSFile{
		readFunc: func(b []byte) (int, error) {
			readCalled = true
			return 0, io.EOF
		},
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
		statFunc: func() (fs.FileInfo, error) {
			statCalled = true
			return nil, nil
		},
	}

	f := internal.ReadOnlyFile{File: mock}

	// Test Read delegation
	_, _ = f.Read(make([]byte, 10))
	if !readCalled {
		t.Error("Read was not delegated")
	}

	// Test Stat delegation
	_, _ = f.Stat()
	if !statCalled {
		t.Error("Stat was not delegated")
	}

	// Test Close delegation
	_ = f.Close()
	if !closeCalled {
		t.Error("Close was not delegated")
	}
}
