package contextual_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gwangyi/fsx/contextual"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

func TestToContextual_Open(t *testing.T) {
	mapFS := fstest.MapFS{
		"testfile": {Data: []byte("hello")},
	}

	ctxFS := contextual.ToContextual(mapFS)

	// Test successful Open
	f, err := ctxFS.Open(t.Context(), "testfile")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if stat.Name() != "testfile" {
		t.Errorf("Expected name 'testfile', got %s", stat.Name())
	}

	// Test Open error
	_, err = ctxFS.Open(t.Context(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Expected NotExist error, got %v", err)
	}
}

func TestToContextual_WriterFS(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "newfile"
		m.EXPECT().Create(name).Return(nil, nil)
		_, err := ctxFS.Create(t.Context(), name)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
	})

	t.Run("OpenFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "openfile"
		flag := 0
		mode := fs.FileMode(0644)
		m.EXPECT().OpenFile(name, flag, mode).Return(nil, nil)
		_, err := ctxFS.OpenFile(t.Context(), name, flag, mode)
		if err != nil {
			t.Errorf("OpenFile failed: %v", err)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockFS(ctrl)
		ctxFS := contextual.ToContextual(m).(contextual.WriterFS)
		name := "oldfile"
		m.EXPECT().Remove(name).Return(nil)
		err := ctxFS.Remove(t.Context(), name)
		if err != nil {
			t.Errorf("Remove failed: %v", err)
		}
	})
}

func TestToContextual_RO(t *testing.T) {
	mapFS := fstest.MapFS{}
	ctxFS := contextual.ToContextual(mapFS).(contextual.WriterFS)

	_, err := ctxFS.Create(t.Context(), "foo")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Expected ErrUnsupported for Create on RO FS, got %v", err)
	}

	err = ctxFS.Remove(t.Context(), "foo")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Expected ErrUnsupported for Remove on RO FS, got %v", err)
	}
}

func TestToContextual_Interfaces(t *testing.T) {
	t.Run("ReadFile", func(t *testing.T) {
		mapFS := fstest.MapFS{"foo": {Data: []byte("bar")}}
		fsys := contextual.ToContextual(mapFS)
		data, err := fsys.(contextual.ReadFileFS).ReadFile(t.Context(), "foo")
		if err != nil || string(data) != "bar" {
			t.Errorf("ReadFile failed: %v, %s", err, data)
		}
	})

	t.Run("ReadDir", func(t *testing.T) {
		mapFS := fstest.MapFS{"dir/foo": {Data: []byte("bar")}}
		fsys := contextual.ToContextual(mapFS)
		entries, err := fsys.(contextual.ReadDirFS).ReadDir(t.Context(), "dir")
		if err != nil || len(entries) != 1 || entries[0].Name() != "foo" {
			t.Errorf("ReadDir failed: %v, %v", err, entries)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		mapFS := fstest.MapFS{"foo": {Data: []byte("bar")}}
		fsys := contextual.ToContextual(mapFS)
		fi, err := fsys.(contextual.StatFS).Stat(t.Context(), "foo")
		if err != nil || fi.Name() != "foo" {
			t.Errorf("Stat failed: %v", err)
		}
	})

	t.Run("Mkdir", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockDirFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Mkdir("foo", fs.FileMode(0755)).Return(nil)
		err := fsys.(contextual.DirFS).Mkdir(t.Context(), "foo", 0755)
		if err != nil {
			t.Errorf("Mkdir failed: %v", err)
		}
	})

	t.Run("MkdirAll", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockMkdirAllFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().MkdirAll("foo/bar", fs.FileMode(0755)).Return(nil)
		err := fsys.(contextual.MkdirAllFS).MkdirAll(t.Context(), "foo/bar", 0755)
		if err != nil {
			t.Errorf("MkdirAll failed: %v", err)
		}
	})

	t.Run("RemoveAll", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockRemoveAllFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().RemoveAll("foo").Return(nil)
		err := fsys.(contextual.RemoveAllFS).RemoveAll(t.Context(), "foo")
		if err != nil {
			t.Errorf("RemoveAll failed: %v", err)
		}
	})

	t.Run("Rename", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockRenameFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Rename("foo", "bar").Return(nil)
		err := fsys.(contextual.RenameFS).Rename(t.Context(), "foo", "bar")
		if err != nil {
			t.Errorf("Rename failed: %v", err)
		}
	})

	t.Run("Symlink", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockSymlinkFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Symlink("foo", "bar").Return(nil)
		err := fsys.(contextual.SymlinkFS).Symlink(t.Context(), "foo", "bar")
		if err != nil {
			t.Errorf("Symlink failed: %v", err)
		}
	})

	t.Run("ReadLink", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockSymlinkFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().ReadLink("foo").Return("bar", nil)
		link, err := fsys.(contextual.ReadLinkFS).ReadLink(t.Context(), "foo")
		if err != nil || link != "bar" {
			t.Errorf("ReadLink failed: %v, %s", err, link)
		}
	})

	t.Run("Lstat", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockSymlinkFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Lstat("foo").Return(nil, nil)
		fi, err := fsys.(contextual.ReadLinkFS).Lstat(t.Context(), "foo")
		if err != nil {
			t.Errorf("Lstat failed: %v", err)
		}
		if fi != nil {
			t.Errorf("Expected nil FileInfo, got %v", fi)
		}
	})

	t.Run("Truncate", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockTruncateFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Truncate("foo", int64(100)).Return(nil)
		err := fsys.(contextual.TruncateFS).Truncate(t.Context(), "foo", 100)
		if err != nil {
			t.Errorf("Truncate failed: %v", err)
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockWriteFileFS(ctrl)
		fsys := contextual.ToContextual(m)
		data := []byte("bar")
		m.EXPECT().WriteFile("foo", data, fs.FileMode(0644)).Return(nil)
		err := fsys.(contextual.WriteFileFS).WriteFile(t.Context(), "foo", data, 0644)
		if err != nil {
			t.Errorf("WriteFile failed: %v", err)
		}
	})

	t.Run("Chown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockChangeFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Chown("foo", "user", "group").Return(nil)
		err := fsys.(contextual.ChangeFS).Chown(t.Context(), "foo", "user", "group")
		if err != nil {
			t.Errorf("Chown failed: %v", err)
		}
	})

	t.Run("Lchown", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockLchownFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Lchown("foo", "user", "group").Return(nil)
		err := fsys.(contextual.LchownFS).Lchown(t.Context(), "foo", "user", "group")
		if err != nil {
			t.Errorf("Lchown failed: %v", err)
		}
	})

	t.Run("Chmod", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockChangeFS(ctrl)
		fsys := contextual.ToContextual(m)
		m.EXPECT().Chmod("foo", fs.FileMode(0644)).Return(nil)
		err := fsys.(contextual.ChangeFS).Chmod(t.Context(), "foo", 0644)
		if err != nil {
			t.Errorf("Chmod failed: %v", err)
		}
	})

	t.Run("Chtimes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mockfs.NewMockChangeFS(ctrl)
		fsys := contextual.ToContextual(m)
		atime := time.Now()
		mtime := atime.Add(time.Second)
		m.EXPECT().Chtimes("foo", atime, mtime).Return(nil)
		err := fsys.(contextual.ChangeFS).Chtimes(t.Context(), "foo", atime, mtime)
		if err != nil {
			t.Errorf("Chtimes failed: %v", err)
		}
	})
}
