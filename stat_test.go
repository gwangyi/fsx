package fsx_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gwangyi/fsx"
)

// TestStat verifies that fsx.Stat correctly wraps fs.Stat and returns
// extended file information.
func TestStat(t *testing.T) {
	// Create a temporary directory for the test.
	tempDir, err := os.MkdirTemp("", "fsx_stat_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test file within the temporary directory.
	filename := "testfile.txt"
	filePath := filepath.Join(tempDir, filename)
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a filesystem from the temporary directory.
	fsys := os.DirFS(tempDir)

	// Call fsx.Stat.
	fi, err := fsx.Stat(fsys, filename)
	if err != nil {
		t.Fatalf("fsx.Stat failed: %v", err)
	}

	// Verify that the returned FileInfo is valid and extended.
	if fi.Name() != filename {
		t.Errorf("expected name %q, got %q", filename, fi.Name())
	}
	if fi.Size() != 7 {
		t.Errorf("expected size 7, got %d", fi.Size())
	}

	// Check extended fields (platform dependent).
	if runtime.GOOS == "linux" {
		if fi.Owner() == "" {
			t.Error("expected owner to be set on Linux")
		}
		if fi.Group() == "" {
			t.Error("expected group to be set on Linux")
		}
	}
}

// TestLstat verifies that fsx.Lstat correctly returns file information
// for a symbolic link itself, not its target.
func TestLstat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Symbolic links behave differently on Windows; skipping TestLstat.")
	}

	// Create a temporary directory.
	tempDir, err := os.MkdirTemp("", "fsx_lstat_test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a target file.
	targetName := "target.txt"
	targetPath := filepath.Join(tempDir, targetName)
	if err := os.WriteFile(targetPath, []byte("some data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symbolic link to the target file.
	linkName := "link.txt"
	linkPath := filepath.Join(tempDir, linkName)
	if err := os.Symlink(targetName, linkPath); err != nil { // Symlink expects oldname, newname
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create a filesystem from the temporary directory.
	fsys := os.DirFS(tempDir)

	// Call fsx.Lstat on the symbolic link.
	fi, err := fsx.Lstat(fsys, linkName)
	if err != nil {
		t.Fatalf("fsx.Lstat failed: %v", err)
	}

	// Verify that the returned FileInfo is for the symbolic link.
	if fi.Name() != linkName {
		t.Errorf("expected name %q, got %q", linkName, fi.Name())
	}
	if fi.Mode()&fs.ModeSymlink == 0 {
		t.Error("expected file mode to indicate a symbolic link")
	}

	// For a symlink, Size() typically returns the length of the link target path.
	// We check if it's non-zero.
	if fi.Size() == 0 {
		t.Error("expected symlink size to be non-zero (length of target path)")
	}
}
