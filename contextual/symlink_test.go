package contextual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gwangyi/fsx/contextual"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestSymlink(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockSymlinkFS(ctrl)
		mfs.EXPECT().Symlink(ctx, "old", "new").Return(nil)

		if err := contextual.Symlink(ctx, mfs, "old", "new"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("supported error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("symlink error")
		mfs := cmockfs.NewMockSymlinkFS(ctrl)
		mfs.EXPECT().Symlink(ctx, "old", "new").Return(expectedErr)

		if err := contextual.Symlink(ctx, mfs, "old", "new"); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockWriterFS(ctrl)
		if err := contextual.Symlink(ctx, mfs, "old", "new"); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestLchown(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockLchownFS(ctrl)
		mfs.EXPECT().Lchown(ctx, "foo", "u", "g").Return(nil)

		if err := contextual.Lchown(ctx, mfs, "foo", "u", "g"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("supported error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("lchown error")
		mfs := cmockfs.NewMockLchownFS(ctrl)
		mfs.EXPECT().Lchown(ctx, "foo", "u", "g").Return(expectedErr)

		if err := contextual.Lchown(ctx, mfs, "foo", "u", "g"); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockWriterFS(ctrl)
		if err := contextual.Lchown(ctx, mfs, "foo", "u", "g"); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestReadLink(t *testing.T) {
	ctx := context.Background()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mfs := cmockfs.NewMockReadLinkFS(ctrl)
		mfs.EXPECT().ReadLink(ctx, "foo").Return("target", nil)

		got, err := contextual.ReadLink(ctx, mfs, "foo")
		if err != nil {
			t.Fatal(err)
		}
		if got != "target" {
			t.Errorf("expected target, got %q", got)
		}
	})

	t.Run("supported error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("readlink error")
		mfs := cmockfs.NewMockReadLinkFS(ctrl)
		mfs.EXPECT().ReadLink(ctx, "foo").Return("", expectedErr)

		if _, err := contextual.ReadLink(ctx, mfs, "foo"); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mfs := cmockfs.NewMockWriterFS(ctrl)
		if _, err := contextual.ReadLink(ctx, mfs, "foo"); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
