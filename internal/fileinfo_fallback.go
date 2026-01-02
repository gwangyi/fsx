//go:build !linux && !windows

package internal

// fillFromSys attempts to populate defaultFileInfo fields from the Sys() source.
// This is the fallback implementation for operating systems other than Linux and Windows.
// It currently performs no operations, leaving default values.
func fillFromSys(dfi *defaultFileInfo, sys any) {
	// No extended info support for this OS yet.
}
