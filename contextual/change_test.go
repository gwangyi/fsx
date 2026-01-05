package contextual_test

import (
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/gwangyi/fsx/contextual"
	cmockfs "github.com/gwangyi/fsx/mockfs/contextual"
	"go.uber.org/mock/gomock"
)

func TestChown(t *testing.T) {
	ctx := t.Context()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chown(ctx, "foo", "owner", "group").Return(nil)

		if err := contextual.Chown(ctx, m, "foo", "owner", "group"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("chown error")
		m := cmockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chown(ctx, "foo", "owner", "group").Return(expectedErr)

		if err := contextual.Chown(ctx, m, "foo", "owner", "group"); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockFS(ctrl)
		if err := contextual.Chown(ctx, m, "foo", "owner", "group"); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestChmod(t *testing.T) {
	ctx := t.Context()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chmod(ctx, "foo", fs.FileMode(0644)).Return(nil)

		if err := contextual.Chmod(ctx, m, "foo", 0644); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("chmod error")
		m := cmockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chmod(ctx, "foo", fs.FileMode(0644)).Return(expectedErr)

		if err := contextual.Chmod(ctx, m, "foo", 0644); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockFS(ctrl)
		if err := contextual.Chmod(ctx, m, "foo", 0644); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

func TestChtimes(t *testing.T) {
	ctx := t.Context()

	t.Run("supported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := cmockfs.NewMockChangeFS(ctrl)
		now := time.Now()
		m.EXPECT().Chtimes(ctx, "foo", now, now).Return(nil)

		if err := contextual.Chtimes(ctx, m, "foo", now, now); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("supported with error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedErr := errors.New("chtimes error")
		m := cmockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chtimes(ctx, "foo", gomock.Any(), gomock.Any()).Return(expectedErr)

		if err := contextual.Chtimes(ctx, m, "foo", time.Now(), time.Now()); !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := cmockfs.NewMockFS(ctrl)
		if err := contextual.Chtimes(ctx, m, "foo", time.Now(), time.Now()); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
