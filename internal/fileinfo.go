package internal

import (
	"io/fs"
	"time"
)

// FileInfo extends the standard fs.FileInfo interface with additional metadata
// commonly supported by Unix-like filesystems (and partially by others).
//
// It allows access to extended attributes like ownership (Owner/Group),
// Inode numbers, and timestamp details (AccessTime, ChangeTime).
type FileInfo interface {
	fs.FileInfo

	// Owner returns the user name of the owner of the file.
	// If resolution fails, it may return the numeric UID as a string.
	Owner() string

	// Group returns the group name of the group of the file.
	// If resolution fails, it may return the numeric GID as a string.
	Group() string

	// AccessTime returns the last access time of the file (atime).
	AccessTime() time.Time

	// ChangeTime returns the last status change time of the file (ctime).
	ChangeTime() time.Time
}

// defaultFileInfo is a concrete implementation of the FileInfo interface.
// It wraps a standard fs.FileInfo and stores the extended attributes as fields.
type defaultFileInfo struct {
	fs.FileInfo
	owner      string
	group      string
	accessTime time.Time
	changeTime time.Time
}

// Owner returns the owner name.
func (d *defaultFileInfo) Owner() string { return d.owner }

// Group returns the group name.
func (d *defaultFileInfo) Group() string { return d.group }

// AccessTime returns the last access time.
func (d *defaultFileInfo) AccessTime() time.Time { return d.accessTime }

// ChangeTime returns the last status change time.
func (d *defaultFileInfo) ChangeTime() time.Time { return d.changeTime }

// ExtendFileInfo returns a FileInfo that wraps the provided fs.FileInfo.
//
// It attempts to extract extended system-specific information from the underlying
// Sys() method of the input fs.FileInfo (e.g., from syscall.Stat_t on Linux).
// If the information is not available, it populates the fields with best-effort
// defaults (e.g., ModTime is used for AccessTime and ChangeTime if they are missing).
//
// Parameters:
//
//	fi: The standard fs.FileInfo to extend.
//
// Returns:
//
//	FileInfo: The extended file info. If fi already implements FileInfo, it is returned directly.
func ExtendFileInfo(fi fs.FileInfo) FileInfo {
	if fi == nil {
		return nil
	}
	if f, ok := fi.(FileInfo); ok {
		return f
	}
	dfi := &defaultFileInfo{
		FileInfo:   fi,
		accessTime: fi.ModTime(),
		changeTime: fi.ModTime(),
	}
	fillFromSys(dfi, fi.Sys())
	return dfi
}
