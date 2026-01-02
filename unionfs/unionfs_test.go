// Package unionfs_test contains integration tests for the unionfs package.
// It uses mock filesystems to verify the behavior of the union filesystem
// across various scenarios including read-write/read-only layer interactions,
// copy-on-write, whiteouts, and merged directory listings.
package unionfs_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"github.com/gwangyi/fsx/unionfs"
	"go.uber.org/mock/gomock"
)

func TestFS_Open(t *testing.T) {
	t.Run("found in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		mockFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(mockFile, nil)

		file, err := f.Open(t.Context(), "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if file != mockFile {
			t.Errorf("expected mockFile, got %v", file)
		}
	})

	t.Run("found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		mockFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(mockFile, nil)

		file, err := f.Open(t.Context(), "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Since ro is MockFS, contextual.OpenFile wraps it
		if file == mockFile {
			t.Errorf("expected wrapped file, got mockFile")
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		_, err := f.Open(t.Context(), "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := f.Open(t.Context(), "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error (due to whiteout), got %v", err)
		}
	})
}

func TestFS_OpenFile(t *testing.T) {
	t.Run("read-only found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		mockFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(mockFile, nil)

		file, err := contextual.OpenFile(t.Context(), f, "test.txt", os.O_RDONLY, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Since ro is MockFS, contextual.OpenFile wraps it
		if file == mockFile {
			t.Errorf("expected wrapped file, got mockFile")
		}
	})

	t.Run("write found in RO (copy-on-write)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		// Find in RO: Stat on RO
		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockInfo, nil)

		// Copy file: Open RO
		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		roFile.EXPECT().Close().Return(nil)

		// Create in RW (copyToRW calls OpenFile)
		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(0644)).Return(rwFile, nil)
		rwFile.EXPECT().Close().Return(nil)

		// Remove whiteout
		rw.EXPECT().Remove(t.Context(), ".wh.test.txt").Return(fs.ErrNotExist)

		// Finally OpenFile in RW
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDWR, fs.FileMode(0)).Return(rwFile, nil)

		_, err := contextual.OpenFile(t.Context(), f, "test.txt", os.O_RDWR, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Remove(t *testing.T) {
	t.Run("in RW only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Remove(t.Context(), "test.txt").Return(nil)
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		err := contextual.Remove(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("in RO (whiteout)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Remove(t.Context(), "test.txt").Return(fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// createWhiteout uses WriteFile
		rw.EXPECT().WriteFile(t.Context(), ".wh.test.txt", nil, fs.FileMode(0644)).Return(nil)

		err := contextual.Remove(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("in RO subdir (whiteout)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Remove(t.Context(), "subdir/test.txt").Return(fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "subdir/test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// createWhiteout uses MkdirAll first
		rw.EXPECT().MkdirAll(t.Context(), "subdir/", fs.FileMode(0755)).Return(nil)

		// Then WriteFile
		rw.EXPECT().WriteFile(t.Context(), "subdir/.wh.test.txt", nil, fs.FileMode(0644)).Return(nil)

		err := contextual.Remove(t.Context(), f, "subdir/test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("in RO subdir (whiteout) with failing mkdir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("expected")
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Remove(t.Context(), "subdir/test.txt").Return(fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "subdir/test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// createWhiteout uses MkdirAll first
		rw.EXPECT().MkdirAll(t.Context(), "subdir/", fs.FileMode(0755)).Return(expectedErr)

		err := contextual.Remove(t.Context(), f, "subdir/test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_ReadDir(t *testing.T) {
	t.Run("merge entries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadDirFS(ctrl)
		f := unionfs.New(rw, ro)

		de1 := mockfs.NewMockDirEntry(ctrl)
		de1.EXPECT().Name().Return("a.txt").AnyTimes()
		rw.EXPECT().ReadDir(t.Context(), "dir").Return([]fs.DirEntry{de1}, nil)

		de2 := mockfs.NewMockDirEntry(ctrl)
		de2.EXPECT().Name().Return("b.txt").AnyTimes()
		de1ro := mockfs.NewMockDirEntry(ctrl)
		de1ro.EXPECT().Name().Return("a.txt").AnyTimes()
		ro.EXPECT().ReadDir(t.Context(), "dir").Return([]fs.DirEntry{de1ro, de2}, nil)

		entries, err := contextual.ReadDir(t.Context(), f, "dir")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
		if entries[0].Name() != "a.txt" || entries[1].Name() != "b.txt" {
			t.Errorf("unexpected entry names")
		}
	})

	t.Run("whiteout entries", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadDirFS(ctrl)
		f := unionfs.New(rw, ro)

		wh := mockfs.NewMockDirEntry(ctrl)
		wh.EXPECT().Name().Return(".wh.b.txt").AnyTimes()
		rw.EXPECT().ReadDir(t.Context(), "dir").Return([]fs.DirEntry{wh}, nil)

		de1 := mockfs.NewMockDirEntry(ctrl)
		de1.EXPECT().Name().Return("a.txt").AnyTimes()
		de2 := mockfs.NewMockDirEntry(ctrl)
		de2.EXPECT().Name().Return("b.txt").AnyTimes()
		ro.EXPECT().ReadDir(t.Context(), "dir").Return([]fs.DirEntry{de1, de2}, nil)

		entries, err := contextual.ReadDir(t.Context(), f, "dir")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Name() != "a.txt" {
			t.Errorf("expected a.txt, got %s", entries[0].Name())
		}
	})
}

func TestFS_Mkdir(t *testing.T) {
	t.Run("mkdir removes whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Mkdir(t.Context(), "newdir", fs.FileMode(0755)).Return(nil)
		rw.EXPECT().Remove(t.Context(), ".wh.newdir").Return(nil)

		err := contextual.Mkdir(t.Context(), f, "newdir", 0755)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Rename(t *testing.T) {
	t.Run("rename in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// Stat old.txt on RW
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// inRO check
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)

		// copyToRW (already in RW, so Stat again)
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// Rename
		rw.EXPECT().Rename(t.Context(), "old.txt", "new.txt").Return(nil)

		err := contextual.Rename(t.Context(), f, "old.txt", "new.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rename in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// f.Stat(old.txt)
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.old.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// inRO check
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// copyToRW
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)
		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(mockInfo, nil)
		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "old.txt").Return(roFile, nil)
		roFile.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		roFile.EXPECT().Close().Return(nil)
		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "old.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(0644)).Return(rwFile, nil)
		rwFile.EXPECT().Close().Return(nil)
		rw.EXPECT().Remove(t.Context(), ".wh.old.txt").Return(nil)

		// Rename
		rw.EXPECT().Rename(t.Context(), "old.txt", "new.txt").Return(nil)

		// whiteout old.txt
		rw.EXPECT().WriteFile(t.Context(), ".wh.old.txt", nil, fs.FileMode(0644)).Return(nil)

		err := contextual.Rename(t.Context(), f, "old.txt", "new.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("rename oldname not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.old.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)

		err := contextual.Rename(t.Context(), f, "old.txt", "new.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("rename copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		// f.Stat(old.txt)
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.old.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// inRO check
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// copyToRW fails
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, expectedErr)

		err := contextual.Rename(t.Context(), f, "old.txt", "new.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})

	t.Run("rename copyToRW failure other", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// f.Stat(old.txt)
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(nil, nil)

		// inRO check
		ro.EXPECT().Stat(t.Context(), "old.txt").Return(nil, fs.ErrNotExist)

		// copyToRW (already in RW, so Stat again)
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "old.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// Rename
		rw.EXPECT().Rename(t.Context(), "old.txt", "new.txt").Return(expectedErr)

		err := contextual.Rename(t.Context(), f, "old.txt", "new.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_Stat(t *testing.T) {
	t.Run("found in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Stat(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Stat(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Stat(t.Context(), f, "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		_, err := contextual.Stat(t.Context(), f, "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})
}

func TestFS_Lstat(t *testing.T) {
	t.Run("found in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Lstat(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		ro.EXPECT().Lstat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Lstat(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.Lstat(t.Context(), f, "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		_, err := contextual.Lstat(t.Context(), f, "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("RW error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrPermission)

		_, err := f.Lstat(t.Context(), "test.txt")
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected ErrPermission, got %v", err)
		}
	})

	t.Run("RO error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Lstat(t.Context(), "test.txt").Return(nil, fs.ErrPermission)

		_, err := f.Lstat(t.Context(), "test.txt")
		if !errors.Is(err, fs.ErrPermission) {
			t.Errorf("expected ErrPermission, got %v", err)
		}
	})
}

func TestFS_ReadLink(t *testing.T) {
	t.Run("found in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadLink(t.Context(), "link").Return("target", nil)

		link, err := contextual.ReadLink(t.Context(), f, "link")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if link != "target" {
			t.Errorf("expected target, got %s", link)
		}
	})

	t.Run("found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadLink(t.Context(), "link").Return("", fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.link").Return(nil, fs.ErrNotExist)

		ro.EXPECT().ReadLink(t.Context(), "link").Return("target", nil)

		link, err := contextual.ReadLink(t.Context(), f, "link")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if link != "target" {
			t.Errorf("expected target, got %s", link)
		}
	})

	t.Run("whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadLink(t.Context(), "link").Return("", fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.link").Return(mockfs.NewMockFileInfo(ctrl), nil)

		_, err := contextual.ReadLink(t.Context(), f, "link")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadLink(t.Context(), "link").Return("", fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.link").Return(nil, fs.ErrNotExist)
		ro.EXPECT().ReadLink(t.Context(), "link").Return("", fs.ErrNotExist)

		_, err := contextual.ReadLink(t.Context(), f, "link")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})
}

func TestFS_Symlink(t *testing.T) {
	t.Run("symlink removes whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Symlink(t.Context(), "old", "new").Return(nil)
		rw.EXPECT().Remove(t.Context(), ".wh.new").Return(nil)

		err := contextual.Symlink(t.Context(), f, "old", "new")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Truncate(t *testing.T) {
	t.Run("truncate in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		rw.EXPECT().Truncate(t.Context(), "test.txt", int64(10)).Return(nil)

		err := contextual.Truncate(t.Context(), f, "test.txt", 10)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("truncate copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW fails on first Stat on RW
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		err := contextual.Truncate(t.Context(), f, "test.txt", 10)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_Chown(t *testing.T) {
	t.Run("chown in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		rw.EXPECT().Chown(t.Context(), "test.txt", "user", "group").Return(nil)

		err := contextual.Chown(t.Context(), f, "test.txt", "user", "group")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("chown copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW fails on first Stat on RW
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		err := contextual.Chown(t.Context(), f, "test.txt", "user", "group")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_Chmod(t *testing.T) {
	t.Run("chmod in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		rw.EXPECT().Chmod(t.Context(), "test.txt", fs.FileMode(0644)).Return(nil)

		err := contextual.Chmod(t.Context(), f, "test.txt", 0644)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("chmod copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW fails on first Stat on RW
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		err := contextual.Chmod(t.Context(), f, "test.txt", 0644)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_Chtimes(t *testing.T) {
	t.Run("chtimes in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		now := time.Now()
		rw.EXPECT().Chtimes(t.Context(), "test.txt", now, now).Return(nil)

		err := contextual.Chtimes(t.Context(), f, "test.txt", now, now)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("chtimes copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW fails on first Stat on RW
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		now := time.Now()
		err := contextual.Chtimes(t.Context(), f, "test.txt", now, now)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_Lchown(t *testing.T) {
	t.Run("lchown in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(mockfs.NewMockFileInfo(ctrl), nil)

		rw.EXPECT().Lchown(t.Context(), "test.txt", "user", "group").Return(nil)

		err := contextual.Lchown(t.Context(), f, "test.txt", "user", "group")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("lchown copyToRW fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW fails on first Stat on RW
		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		err := contextual.Lchown(t.Context(), f, "test.txt", "user", "group")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_MkdirAll(t *testing.T) {
	t.Run("mkdirall removes whiteout", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().MkdirAll(t.Context(), "newdir", fs.FileMode(0755)).Return(nil)
		rw.EXPECT().Remove(t.Context(), ".wh.newdir").Return(nil)

		err := contextual.MkdirAll(t.Context(), f, "newdir", 0755)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_RemoveAll(t *testing.T) {
	t.Run("removeall in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().RemoveAll(t.Context(), "test").Return(nil)
		ro.EXPECT().Stat(t.Context(), "test").Return(nil, fs.ErrNotExist)

		err := contextual.RemoveAll(t.Context(), f, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("removeall in RO (whiteout)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().RemoveAll(t.Context(), "test").Return(nil)
		ro.EXPECT().Stat(t.Context(), "test").Return(mockfs.NewMockFileInfo(ctrl), nil)

		// createWhiteout
		rw.EXPECT().WriteFile(t.Context(), ".wh.test", nil, fs.FileMode(0644)).Return(nil)

		err := contextual.RemoveAll(t.Context(), f, "test")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Create(t *testing.T) {
	t.Run("create calls openfile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		// Find in RO
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		// OpenFile
		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, fs.FileMode(0666)).Return(rwFile, nil)

		file, err := contextual.Create(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if file != rwFile {
			t.Errorf("expected rwFile, got %v", file)
		}
	})
}

func TestFS_WriteFile(t *testing.T) {
	t.Run("writefile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		data := []byte("hello")
		rw.EXPECT().WriteFile(t.Context(), "test.txt", data, fs.FileMode(0644)).Return(nil)

		err := contextual.WriteFile(t.Context(), f, "test.txt", data, 0644)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_ReadFile(t *testing.T) {
	t.Run("found in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)

		data := []byte("hello")
		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(data, nil)

		res, err := contextual.ReadFile(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(res) != string(data) {
			t.Errorf("expected %s, got %s", data, res)
		}
	})

	t.Run("found in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		data := []byte("hello")
		ro.EXPECT().ReadFile(t.Context(), "test.txt").Return(data, nil)

		res, err := contextual.ReadFile(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(res) != string(data) {
			t.Errorf("expected %s, got %s", data, res)
		}
	})

	t.Run("copy on read", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)
		unionfs.SetCopyOnRead(f, true)

		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		data := []byte("hello")
		ro.EXPECT().ReadFile(t.Context(), "test.txt").Return(data, nil)

		rw.EXPECT().WriteFile(t.Context(), "test.txt", data, fs.FileMode(0666)).Return(nil)

		res, err := contextual.ReadFile(t.Context(), f, "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if string(res) != string(data) {
			t.Errorf("expected %s, got %s", data, res)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		_, err := contextual.ReadFile(t.Context(), f, "test.txt")
		if !os.IsNotExist(err) {
			t.Errorf("expected NotExist error, got %v", err)
		}
	})
}

func TestFS_CopyOnRead(t *testing.T) {
	t.Run("copy on read", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)
		unionfs.SetCopyOnRead(f, true)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		// isWhiteout check
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Close().Return(nil)

		// copyToRW calls Stat on RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		// Find in RO: Stat on RO
		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockInfo, nil)

		// Copy file: Open RO
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Read(gomock.Any()).Return(0, io.EOF)
		roFile.EXPECT().Close().Return(nil)

		// Create in RW
		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(0644)).Return(rwFile, nil)
		rwFile.EXPECT().Close().Return(nil)

		// Remove whiteout
		rw.EXPECT().Remove(t.Context(), ".wh.test.txt").Return(fs.ErrNotExist)

		// Finally Open in RW
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(rwFile, nil)

		_, err := f.Open(t.Context(), "test.txt")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy on read directory", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)
		unionfs.SetCopyOnRead(f, true)

		rw.EXPECT().OpenFile(t.Context(), "dir", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.dir").Return(nil, fs.ErrNotExist)

		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "dir").Return(roFile, nil)
		roFile.EXPECT().Close().Return(nil)

		// copyToRW calls
		rw.EXPECT().Stat(t.Context(), "dir").Return(nil, fs.ErrNotExist)
		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(true).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0755 | fs.ModeDir)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "dir").Return(mockInfo, nil)

		rw.EXPECT().MkdirAll(t.Context(), "dir", fs.FileMode(0755)).Return(nil)

		// Finally Open in RW
		rw.EXPECT().OpenFile(t.Context(), "dir", os.O_RDONLY, fs.FileMode(0)).Return(roFile, nil)

		_, err := f.Open(t.Context(), "dir")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_OpenFile_Errors(t *testing.T) {
	t.Run("write new file (copyToRW returns NotExist)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW calls
		rw.EXPECT().Stat(t.Context(), "new.txt").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Stat(t.Context(), "new.txt").Return(nil, fs.ErrNotExist)

		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "new.txt", os.O_RDWR|os.O_CREATE, fs.FileMode(0644)).Return(rwFile, nil)

		_, err := f.OpenFile(t.Context(), "new.txt", os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("read-only other error in RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, expectedErr)

		_, err := f.OpenFile(t.Context(), "test.txt", os.O_RDONLY, 0)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("read-only other error in RO", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		expectedErr := errors.New("expected")
		ro.EXPECT().Open(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.OpenFile(t.Context(), "test.txt", os.O_RDONLY, 0)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy on read fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)
		unionfs.SetCopyOnRead(f, true)

		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_RDONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Close().Return(nil)

		expectedErr := errors.New("copy failed")
		// copyToRW fails
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.OpenFile(t.Context(), "test.txt", os.O_RDONLY, 0)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("write append found in RO (copy-on-write) copy error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		// copyToRW
		// Check if already in RW
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		// Find in RO
		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockInfo, nil)

		// Open source from RO
		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Close().Return(nil)

		// Open destination in RW
		rwFile := mockfs.NewMockFile(ctrl)
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(0644)).Return(rwFile, nil)
		rwFile.EXPECT().Close().Return(nil)

		// Fail copy (read from RO file fails)
		expectedErr := errors.New("read error")
		roFile.EXPECT().Read(gomock.Any()).Return(0, expectedErr)

		_, err := f.OpenFile(t.Context(), "test.txt", os.O_RDWR|os.O_APPEND, 0)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v, want %v", err, expectedErr)
		}
	})
}

func TestFS_ReadDir_Errors(t *testing.T) {
	t.Run("RW error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().ReadDir(t.Context(), "dir").Return(nil, expectedErr)

		_, err := f.ReadDir(t.Context(), "dir")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("RO error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadDirFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadDir(t.Context(), "dir").Return(nil, fs.ErrNotExist)

		expectedErr := errors.New("expected")
		ro.EXPECT().ReadDir(t.Context(), "dir").Return(nil, expectedErr)

		_, err := f.ReadDir(t.Context(), "dir")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty but error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadDir(t.Context(), "dir").Return(nil, fs.ErrNotExist)
		ro.EXPECT().Open(t.Context(), "dir").Return(nil, fs.ErrNotExist)

		_, err := f.ReadDir(t.Context(), "dir")
		if !os.IsNotExist(err) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Mkdir_Errors(t *testing.T) {
	t.Run("mkdir fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().Mkdir(t.Context(), "dir", fs.FileMode(0755)).Return(expectedErr)

		err := f.Mkdir(t.Context(), "dir", 0755)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_MkdirAll_Errors(t *testing.T) {
	t.Run("mkdirall fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().MkdirAll(t.Context(), "dir", fs.FileMode(0755)).Return(expectedErr)

		err := f.MkdirAll(t.Context(), "dir", 0755)
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Symlink_Errors(t *testing.T) {
	t.Run("symlink fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().Symlink(t.Context(), "old", "new").Return(expectedErr)

		err := f.Symlink(t.Context(), "old", "new")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_RemoveAll_Errors(t *testing.T) {
	t.Run("removeall fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().RemoveAll(t.Context(), "test").Return(expectedErr)

		err := f.RemoveAll(t.Context(), "test")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Remove_Errors(t *testing.T) {
	t.Run("RW other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().Remove(t.Context(), "test.txt").Return(expectedErr)

		err := f.Remove(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_Stat_Errors(t *testing.T) {
	t.Run("RW other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.Stat(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("RO other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.test.txt").Return(nil, fs.ErrNotExist)

		expectedErr := errors.New("expected")
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.Stat(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_ReadLink_Errors(t *testing.T) {
	t.Run("RW other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().ReadLink(t.Context(), "link").Return("", expectedErr)

		_, err := f.ReadLink(t.Context(), "link")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("RO other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadLinkFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadLink(t.Context(), "link").Return("", fs.ErrNotExist)
		rw.EXPECT().Stat(t.Context(), ".wh.link").Return(nil, fs.ErrNotExist)

		expectedErr := errors.New("expected")
		ro.EXPECT().ReadLink(t.Context(), "link").Return("", expectedErr)

		_, err := f.ReadLink(t.Context(), "link")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFS_ReadFile_Errors(t *testing.T) {
	t.Run("RW other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)

		expectedErr := errors.New("expected")
		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.ReadFile(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("RO other error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)

		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		expectedErr := errors.New("expected")
		ro.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, expectedErr)

		_, err := f.ReadFile(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy on read write error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockReadFileFS(ctrl)
		f := unionfs.New(rw, ro)
		unionfs.SetCopyOnRead(f, true)

		rw.EXPECT().ReadFile(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		data := []byte("hello")
		ro.EXPECT().ReadFile(t.Context(), "test.txt").Return(data, nil)

		expectedErr := errors.New("write error")
		rw.EXPECT().WriteFile(t.Context(), "test.txt", data, fs.FileMode(0666)).Return(expectedErr)

		_, err := f.ReadFile(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
