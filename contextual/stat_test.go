package contextual_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestStat(t *testing.T) {
	ctx := context.Background()

	t.Run("StatFS supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		info := mockfs.NewMockFileInfo(ctrl)
		// ExtendFileInfo expectations
		info.EXPECT().Name().Return("foo").AnyTimes()
		info.EXPECT().Size().Return(int64(0)).AnyTimes()
		info.EXPECT().Mode().Return(fs.FileMode(0)).AnyTimes()
		info.EXPECT().ModTime().Return(time.Time{}).AnyTimes()
		info.EXPECT().IsDir().Return(false).AnyTimes()
		info.EXPECT().Sys().Return(nil).AnyTimes()

		mfs := cmockfs.NewMockStatFS(ctrl)
		mfs.EXPECT().Stat(ctx, "foo").Return(info, nil)

		got, err := contextual.Stat(ctx, mfs, "foo")
		if err != nil {
			t.Fatal(err)
		}
		if got.Name() != "foo" {
			t.Error("Name mismatch")
		}
	})

	t.Run("Fallback to Open", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		f := mockfs.NewMockFile(ctrl)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Name().Return("foo").AnyTimes()
		info.EXPECT().Size().Return(int64(0)).AnyTimes()
		info.EXPECT().Mode().Return(fs.FileMode(0)).AnyTimes()
		info.EXPECT().ModTime().Return(time.Time{}).AnyTimes()
		info.EXPECT().IsDir().Return(false).AnyTimes()
		info.EXPECT().Sys().Return(nil).AnyTimes()

		f.EXPECT().Stat().Return(info, nil)
		f.EXPECT().Close().Return(nil)

		mfs := cmockfs.NewMockFS(ctrl)
		mfs.EXPECT().Open(ctx, "foo").Return(f, nil)

		got, err := contextual.Stat(ctx, mfs, "foo")
		if err != nil {
			t.Fatal(err)
		}
		if got.Name() != "foo" {
			t.Error("Name mismatch")
		}
	})

	t.Run("Fallback Open fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockFS(ctrl)
		mfs.EXPECT().Open(ctx, "foo").Return(nil, errors.New("open error"))

		_, err := contextual.Stat(ctx, mfs, "foo")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLstat(t *testing.T) {
	ctx := context.Background()

	t.Run("ReadLinkFS supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Name().Return("foo").AnyTimes()
		info.EXPECT().Size().Return(int64(0)).AnyTimes()
		info.EXPECT().Mode().Return(os.ModeSymlink).AnyTimes()
		info.EXPECT().ModTime().Return(time.Time{}).AnyTimes()
		info.EXPECT().IsDir().Return(false).AnyTimes()
		info.EXPECT().Sys().Return(nil).AnyTimes()

		mfs := cmockfs.NewMockReadLinkFS(ctrl)
		mfs.EXPECT().Lstat(ctx, "foo").Return(info, nil)

		got, err := contextual.Lstat(ctx, mfs, "foo")
		if err != nil {
			t.Fatal(err)
		}
		if got.Mode()&os.ModeSymlink == 0 {
			t.Error("expected symlink mode")
		}
	})

	t.Run("ReadLinkFS returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockReadLinkFS(ctrl)
		mfs.EXPECT().Lstat(ctx, "foo").Return(nil, errors.New("lstat error"))

		_, err := contextual.Lstat(ctx, mfs, "foo")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("Fallback to Stat (which falls back to Open)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		f := mockfs.NewMockFile(ctrl)
		info := mockfs.NewMockFileInfo(ctrl)
		info.EXPECT().Name().Return("foo").AnyTimes()
		info.EXPECT().Size().Return(int64(0)).AnyTimes()
		info.EXPECT().Mode().Return(os.ModeSymlink).AnyTimes()
		info.EXPECT().ModTime().Return(time.Time{}).AnyTimes()
		info.EXPECT().IsDir().Return(false).AnyTimes()
		info.EXPECT().Sys().Return(nil).AnyTimes()

		f.EXPECT().Stat().Return(info, nil)
		f.EXPECT().Close().Return(nil)

		mfs := cmockfs.NewMockFS(ctrl)
		mfs.EXPECT().Open(ctx, "foo").Return(f, nil)

		got, err := contextual.Lstat(ctx, mfs, "foo")
		if err != nil {
			t.Fatal(err)
		}
		if got.Mode()&os.ModeSymlink == 0 {
			t.Error("expected symlink mode")
		}
	})
}
