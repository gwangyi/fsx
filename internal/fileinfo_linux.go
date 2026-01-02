//go:build linux

package internal

import (
	"os/user"
	"strconv"
	"syscall"
	"time"
)

// fillFromSys attempts to populate defaultFileInfo fields from the Sys() source
// using Linux-specific syscall.Stat_t structure.
//
// It extracts Dev, Ino, Nlink, Uid (mapped to Owner), Gid (mapped to Group),
// Rdev, Blksize, Blocks, Atim, and Ctim.
func fillFromSys(dfi *defaultFileInfo, sys any) {
	if st, ok := sys.(*syscall.Stat_t); ok {
		// Try to lookup owner name, fall back to numeric ID.
		uidStr := strconv.Itoa(int(st.Uid))
		if u, err := user.LookupId(uidStr); err == nil {
			dfi.owner = u.Username
		} else {
			dfi.owner = uidStr
		}

		// Try to lookup group name, fall back to numeric ID.
		gidStr := strconv.Itoa(int(st.Gid))
		if g, err := user.LookupGroupId(gidStr); err == nil {
			dfi.group = g.Name
		} else {
			dfi.group = gidStr
		}

		dfi.accessTime = time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
		dfi.changeTime = time.Unix(int64(st.Ctim.Sec), int64(st.Ctim.Nsec))
	}
}
