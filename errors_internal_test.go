package fsx

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"
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
			gotErr := intoPathErr(tt.op, tt.path, tt.err)

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
