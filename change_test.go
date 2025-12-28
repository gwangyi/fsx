package fsx_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

// TestChown verifies the behavior of the fsx.Chown helper function.
func TestChown(t *testing.T) {
	// Test case: The filesystem supports ChangeFS and the operation succeeds.
	t.Run("supported", func(t *testing.T) {

		nameArg := "foo"
		ownerArg := "owner"
		groupArg := "group"

		// Create a mock filesystem that expects specific arguments.
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chown(nameArg, ownerArg, groupArg).Return(nil)
		// Call the helper function.
		err := fsx.Chown(m, nameArg, ownerArg, groupArg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

	})

	// Test case: The filesystem supports ChangeFS but the operation returns an error.
	t.Run("supported with error", func(t *testing.T) {
		expectedErr := errors.New("chown error")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chown("foo", "owner", "group").Return(expectedErr)
		err := fsx.Chown(m, "foo", "owner", "group")
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	// Test case: The filesystem does not support ChangeFS (does not implement the interface).
	t.Run("unsupported", func(t *testing.T) {
		mapFS := fstest.MapFS{} // fstest.MapFS does not implement ChangeFS
		err := fsx.Chown(mapFS, "foo", "owner", "group")
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

// TestChmod verifies the behavior of the fsx.Chmod helper function.
func TestChmod(t *testing.T) {
	// Test case: The filesystem supports ChangeFS and the operation succeeds.
	t.Run("supported", func(t *testing.T) {

		nameArg := "foo"
		modeArg := fs.FileMode(0644)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chmod(nameArg, modeArg).Return(nil)
		err := fsx.Chmod(m, nameArg, modeArg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Test case: The filesystem supports ChangeFS but the operation returns an error.
	t.Run("supported with error", func(t *testing.T) {
		expectedErr := errors.New("chmod error")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chmod("foo", fs.FileMode(0644)).Return(expectedErr)
		err := fsx.Chmod(m, "foo", 0644)
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	// Test case: The filesystem does not support ChangeFS.
	t.Run("unsupported", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		err := fsx.Chmod(mapFS, "foo", 0644)
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}

// TestChtimes verifies the behavior of the fsx.Chtimes helper function.
func TestChtimes(t *testing.T) {
	// Test case: The filesystem supports ChangeFS and the operation succeeds.
	t.Run("supported", func(t *testing.T) {

		nameArg := "foo"
		atimeArg := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		ctimeArg := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chtimes(nameArg, atimeArg, ctimeArg).Return(nil)
		err := fsx.Chtimes(m, nameArg, atimeArg, ctimeArg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Test case: The filesystem supports ChangeFS but the operation returns an error.
	t.Run("supported with error", func(t *testing.T) {
		expectedErr := errors.New("chtimes error")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mockfs.NewMockChangeFS(ctrl)
		m.EXPECT().Chtimes("foo", gomock.Any(), gomock.Any()).Return(expectedErr)
		err := fsx.Chtimes(m, "foo", time.Now(), time.Now())
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	// Test case: The filesystem does not support ChangeFS.
	t.Run("unsupported", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		err := fsx.Chtimes(mapFS, "foo", time.Now(), time.Now())
		if !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("expected ErrUnsupported, got %v", err)
		}
	})
}
