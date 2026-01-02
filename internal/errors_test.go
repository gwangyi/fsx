package internal_test

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/gwangyi/fsx/internal"
)

func TestIntoPathErr(t *testing.T) {
	expectedErr := errors.New("something went wrong")

	tests := []struct {
		name    string
		op      string
		path    string
		err     error
		wantErr error
	}{
		{
			name:    "nil error",
			op:      "open",
			path:    "file.txt",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "generic error",
			op:      "read",
			path:    "data.json",
			err:     expectedErr,
			wantErr: expectedErr,
		},
		{
			name:    "fs.PathError already",
			op:      "write",
			path:    "output.txt",
			err:     &fs.PathError{Op: "origOp", Path: "origPath", Err: expectedErr},
			wantErr: expectedErr,
		},
		{
			name:    "os.LinkError",
			op:      "link",
			path:    "newlink",
			err:     &os.LinkError{Op: "link", Old: "oldfile", New: "newlink", Err: syscall.EACCES},
			wantErr: syscall.EACCES,
		},
		{
			name:    "os.SyscallError",
			op:      "unlink",
			path:    "file.tmp",
			err:     &os.SyscallError{Syscall: "unlink", Err: syscall.EPERM},
			wantErr: syscall.EPERM,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := internal.IntoPathErr(tt.op, tt.path, tt.err)

			if tt.wantErr == nil {
				if gotErr != nil {
					t.Errorf("IntoPathErr() = %v, want nil", gotErr)
				}
				return
			}

			// Check if the returned error is an fs.PathError
			pathErr, ok := gotErr.(*fs.PathError)
			if !ok {
				t.Errorf("IntoPathErr() returned error of type %T, want *fs.PathError", gotErr)
				return
			}

			// Check Op and Path
			if pathErr.Op != tt.op {
				t.Errorf("IntoPathErr().Op = %q, want %q", pathErr.Op, tt.op)
			}
			if pathErr.Path != tt.path {
				t.Errorf("IntoPathErr().Path = %q, want %q", pathErr.Path, tt.path)
			}

			// Check the underlying error
			if !errors.Is(pathErr.Err, tt.wantErr) {
				t.Errorf("IntoPathErr().Err = %v, want %v", pathErr.Err, tt.wantErr)
			}
		})
	}
}

func TestIntoLinkErr(t *testing.T) {
	expectedErr := errors.New("link failed")

	tests := []struct {
		name    string
		op      string
		oldpath string
		newpath string
		err     error
		wantErr error
	}{
		{
			name:    "nil error",
			op:      "rename",
			oldpath: "old.txt",
			newpath: "new.txt",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "generic error",
			op:      "link",
			oldpath: "src",
			newpath: "dst",
			err:     expectedErr,
			wantErr: expectedErr,
		},
		{
			name:    "fs.PathError",
			op:      "rename",
			oldpath: "old",
			newpath: "new",
			err:     &fs.PathError{Op: "open", Path: "old", Err: expectedErr},
			wantErr: expectedErr,
		},
		{
			name:    "os.LinkError already",
			op:      "link",
			oldpath: "src",
			newpath: "dst",
			err:     &os.LinkError{Op: "origOp", Old: "origOld", New: "origNew", Err: expectedErr},
			wantErr: expectedErr,
		},
		{
			name:    "os.SyscallError",
			op:      "symlink",
			oldpath: "target",
			newpath: "link",
			err:     &os.SyscallError{Syscall: "symlink", Err: syscall.EEXIST},
			wantErr: syscall.EEXIST,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := internal.IntoLinkErr(tt.op, tt.oldpath, tt.newpath, tt.err)

			if tt.wantErr == nil {
				if gotErr != nil {
					t.Errorf("IntoLinkErr() = %v, want nil", gotErr)
				}
				return
			}

			// Check if the returned error is an os.LinkError
			linkErr, ok := gotErr.(*os.LinkError)
			if !ok {
				t.Errorf("IntoLinkErr() returned error of type %T, want *os.LinkError", gotErr)
				return
			}

			// Check Op, Old and New
			if linkErr.Op != tt.op {
				t.Errorf("IntoLinkErr().Op = %q, want %q", linkErr.Op, tt.op)
			}
			if linkErr.Old != tt.oldpath {
				t.Errorf("IntoLinkErr().Old = %q, want %q", linkErr.Old, tt.oldpath)
			}
			if linkErr.New != tt.newpath {
				t.Errorf("IntoLinkErr().New = %q, want %q", linkErr.New, tt.newpath)
			}

			// Check the underlying error
			if !errors.Is(linkErr.Err, tt.wantErr) {
				t.Errorf("IntoLinkErr().Err = %v, want %v", linkErr.Err, tt.wantErr)
			}
		})
	}
}

func TestIsInvalid(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "fs.ErrInvalid",
			err:      fs.ErrInvalid,
			expected: true,
		},
		{
			name:     "ErrNotDir",
			err:      internal.ErrNotDir,
			expected: true,
		},
		{
			name:     "syscall.ENOTDIR",
			err:      syscall.ENOTDIR,
			expected: true,
		},
		{
			name:     "ErrIsDir",
			err:      internal.ErrIsDir,
			expected: true,
		},
		{
			name:     "syscall.EISDIR",
			err:      syscall.EISDIR,
			expected: true,
		},
		{
			name:     "wrapped fs.ErrInvalid",
			err:      &fs.PathError{Err: fs.ErrInvalid},
			expected: true,
		},
		{
			name:     "fs.ErrNotExist",
			err:      fs.ErrNotExist,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := internal.IsInvalid(tt.err); got != tt.expected {
				t.Errorf("IsInvalid(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestIsUnsupported(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "errors.ErrUnsupported",
			err:      errors.ErrUnsupported,
			expected: true,
		},
		{
			name:     "error string contains 'not implemented'",
			err:      errors.New("some feature not implemented"),
			expected: true,
		},
		{
			name:     "wrapped errors.ErrUnsupported",
			err:      &fs.PathError{Err: errors.ErrUnsupported},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := internal.IsUnsupported(tt.err); got != tt.expected {
				t.Errorf("IsUnsupported(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}
