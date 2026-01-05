package contextual_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestTruncate(t *testing.T) {
	ctx := t.Context()

	t.Run("OptimizedPath", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockTruncateFS(ctrl)
		m.EXPECT().Truncate(ctx, "foo", int64(100)).Return(nil)

		if err := contextual.Truncate(ctx, m, "foo", 100); err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}
	})

	t.Run("Fallback", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		f := mockfs.NewMockFile(ctrl)

		m.EXPECT().OpenFile(ctx, "foo", os.O_WRONLY, fs.FileMode(0)).Return(f, nil)
		f.EXPECT().Truncate(int64(100)).Return(nil)
		f.EXPECT().Close().Return(nil)

		if err := contextual.Truncate(ctx, m, "foo", 100); err != nil {
			t.Fatalf("Truncate failed: %v", err)
		}
	})

	t.Run("Unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockFS(ctrl)
		err := contextual.Truncate(ctx, m, "foo", 100)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})

	t.Run("Fail OpenFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockWriterFS(ctrl)
		m.EXPECT().OpenFile(ctx, "foo", os.O_WRONLY, fs.FileMode(0)).Return(nil, fs.ErrNotExist)

		err := contextual.Truncate(ctx, m, "foo", 100)
		if !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("expected ErrNotExist, got %v", err)
		}
	})

	t.Run("TruncateFS returns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockTruncateFS(ctrl)
		expectedErr := errors.New("truncate error")
		m.EXPECT().Truncate(ctx, "foo", int64(100)).Return(expectedErr)

		if err := contextual.Truncate(ctx, m, "foo", 100); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
