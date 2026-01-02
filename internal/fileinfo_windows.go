//go:build windows

package internal

import (
	"syscall"
	"time"
)

// fillFromSys attempts to populate defaultFileInfo fields from the Sys() source
// using Windows-specific syscall.Win32FileAttributeData structure.
//
// It currently only extracts LastAccessTime.
func fillFromSys(dfi *defaultFileInfo, sys any) {
	if st, ok := sys.(*syscall.Win32FileAttributeData); ok {
		dfi.accessTime = time.Unix(0, st.LastAccessTime.Nanoseconds())
		// ChangeTime is not directly available in Win32FileAttributeData,
		// so we keep the default (ModTime).
	}
}
