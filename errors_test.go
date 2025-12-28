package fsx_test

import (
	"errors"
	"io/fs"
	"syscall"
	"testing"

	"github.com/gwangyi/fsx"
)

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
			err:      fsx.ErrNotDir,
			expected: true,
		},
		{
			name:     "syscall.ENOTDIR",
			err:      syscall.ENOTDIR,
			expected: true,
		},
		{
			name:     "ErrIsDir",
			err:      fsx.ErrIsDir,
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
			if got := fsx.IsInvalid(tt.err); got != tt.expected {
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
			if got := fsx.IsUnsupported(tt.err); got != tt.expected {
				t.Errorf("IsUnsupported(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}
