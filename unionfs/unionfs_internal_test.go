// Package unionfs contains internal tests for the unionfs package.
// These tests focus on private methods and internal logic like copyToRW
// that are not exposed through the public API.
package unionfs

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestFS_copyToRW(t *testing.T) {
	t.Run("copy directory to RW", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := New(rw, ro)
		// Open() with CopyOnRead calls copyToRW
		SetCopyOnRead(f, true)

		rw.EXPECT().Stat(t.Context(), "dir").Return(nil, fs.ErrNotExist)

		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(true).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0755 | fs.ModeDir)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "dir").Return(mockInfo, nil)

		rw.EXPECT().MkdirAll(t.Context(), "dir", fs.FileMode(0755)).Return(nil)

		err := f.copyToRW(t.Context(), "dir")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy file fails on parent mkdirall", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := New(rw, ro)
		// Open() with CopyOnRead calls copyToRW
		SetCopyOnRead(f, true)

		rw.EXPECT().Stat(t.Context(), "dir/test.txt").Return(nil, fs.ErrNotExist)

		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "dir/test.txt").Return(mockInfo, nil)

		expectedErr := errors.New("mkdirall failed")
		rw.EXPECT().MkdirAll(t.Context(), "dir", fs.FileMode(0755)).Return(expectedErr)

		err := f.copyToRW(t.Context(), "dir/test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy file fails on src open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := New(rw, ro)
		// Open() with CopyOnRead calls copyToRW
		SetCopyOnRead(f, true)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockInfo, nil)

		expectedErr := errors.New("open failed")
		ro.EXPECT().Open(t.Context(), "test.txt").Return(nil, expectedErr)

		err := f.copyToRW(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("copy file fails on dst openfile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		rw := cmockfs.NewMockFileSystem(ctrl)
		ro := cmockfs.NewMockStatFS(ctrl)
		f := New(rw, ro)
		// Open() with CopyOnRead calls copyToRW
		SetCopyOnRead(f, true)

		rw.EXPECT().Stat(t.Context(), "test.txt").Return(nil, fs.ErrNotExist)

		mockInfo := mockfs.NewMockFileInfo(ctrl)
		mockInfo.EXPECT().IsDir().Return(false).AnyTimes()
		mockInfo.EXPECT().Mode().Return(fs.FileMode(0644)).AnyTimes()
		ro.EXPECT().Stat(t.Context(), "test.txt").Return(mockInfo, nil)

		roFile := mockfs.NewMockFile(ctrl)
		ro.EXPECT().Open(t.Context(), "test.txt").Return(roFile, nil)
		roFile.EXPECT().Close().Return(nil)

		expectedErr := errors.New("openfile failed")
		rw.EXPECT().OpenFile(t.Context(), "test.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(0644)).Return(nil, expectedErr)

		err := f.copyToRW(t.Context(), "test.txt")
		if !errors.Is(err, expectedErr) {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
