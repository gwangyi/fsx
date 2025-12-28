package fsx_test

import (
	"io/fs"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

// TestExtendFileInfo verifies that ExtendFileInfo correctly populates the
// extended fields of the FileInfo interface from the underlying OS file info.
func TestExtendFileInfo(t *testing.T) {
	// Create a temporary file to stat.
	f, err := os.CreateTemp("", "test_extend_fileinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_ = f.Close()

	// Get the standard os.FileInfo.
	fi, err := os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Extend it to fsx.FileInfo.
	xfi := fsx.ExtendFileInfo(fi)

	// Verify basic fields match.
	if xfi.Name() != fi.Name() {
		t.Errorf("Name mismatch: got %q, want %q", xfi.Name(), fi.Name())
	}

	if xfi.Size() != fi.Size() {
		t.Errorf("Size mismatch: got %d, want %d", xfi.Size(), fi.Size())
	}

	if xfi.Mode() != fi.Mode() {
		t.Errorf("Mode mismatch: got %v, want %v", xfi.Mode(), fi.Mode())
	}

	if !xfi.ModTime().Equal(fi.ModTime()) {
		t.Errorf("ModTime mismatch: got %v, want %v", xfi.ModTime(), fi.ModTime())
	}

	// Platform specific checks.
	if runtime.GOOS == "linux" {
		// Owner/Group should be populated (either name or numeric string).
		if xfi.Owner() == "" {
			t.Error("Expected Owner to be non-empty on Linux")
		}
		if xfi.Group() == "" {
			t.Error("Expected Group to be non-empty on Linux")
		}
	}
}

type mockBasicFileInfo struct {
	fs.FileInfo
}

// TestExtendFileInfo_Fallback verifies that ExtendFileInfo handles basic
// fs.FileInfo implementations (that don't provide Sys()) by returning
// default values for extended fields.
func TestExtendFileInfo_Fallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mfi := mockfs.NewMockFileInfo(ctrl)
	mfi.EXPECT().Name().Return("mock_file").AnyTimes()
	mfi.EXPECT().ModTime().Return(time.Now()).AnyTimes()
	mfi.EXPECT().Sys().Return(nil)

	// mockBasicFileInfo implements fs.FileInfo but returns nil for Sys().
	m := &mockBasicFileInfo{
		FileInfo: mfi,
	}

	xfi := fsx.ExtendFileInfo(m)

	if xfi.Name() != m.Name() {
		t.Errorf("Name mismatch: got %q, want %q", xfi.Name(), m.Name())
	}
	if xfi.Owner() != "" {
		t.Errorf("Expected Owner to be empty for fallback, got %q", xfi.Owner())
	}
	// AccessTime/ChangeTime should default to ModTime
	if !xfi.AccessTime().Equal(m.ModTime()) {
		t.Errorf("Expected AccessTime to equal ModTime, got %v", xfi.AccessTime())
	}
}

func TestExtendFileInfo_Noop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mfi := mockfs.NewMockFileInfo(ctrl)
	// When ExtendFileInfo is called with an fs.FileInfo that already implements
	// the fsx.FileInfo interface, it should return the original object directly
	// without wrapping it again.
	xfi := fsx.ExtendFileInfo(mfi)
	if xfi != mfi {
		t.Errorf("ExtendFileInfo should return the original fsx.FileInfo if it already implements it; got %v, want %v", xfi, mfi)
	}
}

func TestExtendFileInfo_Nil(t *testing.T) {
	if xfi := fsx.ExtendFileInfo(nil); xfi != nil {
		t.Errorf("ExtendFileInfo(nil) = %v, want nil", xfi)
	}
}
