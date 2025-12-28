//go:build linux

package fsx_test

import (
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/gwangyi/fsx"
	"github.com/gwangyi/fsx/mockfs"
	"go.uber.org/mock/gomock"
)

// TestExtendFileInfo_InvalidUserGroupLinux verifies that ExtendFileInfo
// correctly handles non-resolvable UIDs/GIDs on Linux by returning their
// numeric string representation.
func TestExtendFileInfo_InvalidUserGroupLinux(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Define non-existent UID and GID.
	// These are typically values that no real user/group would have.
	invalidUID := uint32(999999999) // A very large, unlikely UID
	invalidGID := uint32(999999998) // A very large, unlikely GID

	// Create a mock syscall.Stat_t with invalid UID/GID.
	mockStat := &syscall.Stat_t{
		Uid: invalidUID,
		Gid: invalidGID,
	}

	// Create a mock fs.FileInfo that returns our mockStat for Sys().
	// We embed a basic mock to satisfy other fs.FileInfo methods.
	now := time.Now()
	mfi := mockfs.NewMockFileInfo(ctrl)
	mfi.EXPECT().ModTime().Return(now).AnyTimes()
	mfi.EXPECT().Sys().Return(mockStat)
	fi := &mockBasicFileInfo{
		FileInfo: mfi,
	}

	// Call ExtendFileInfo with the mock.
	xfi := fsx.ExtendFileInfo(fi)

	// Verify that Owner and Group return the numeric IDs as strings.
	expectedOwner := strconv.Itoa(int(invalidUID))
	if owner := xfi.Owner(); owner != expectedOwner {
		t.Errorf("expected owner %q, got %q", expectedOwner, owner)
	}

	expectedGroup := strconv.Itoa(int(invalidGID))
	if group := xfi.Group(); group != expectedGroup {
		t.Errorf("expected group %q, got %q", expectedGroup, group)
	}
}

func TestExtendFileInfo_Getters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now()
	// Define distinct times for Access, Change, and Mod to ensure correct population.
	accessTime := now.Add(-time.Hour)
	modTime := now.Add(-2 * time.Hour)

	mockStat := &syscall.Stat_t{
		Atim: syscall.NsecToTimespec(accessTime.UnixNano()), // Set explicit AccessTime
		Ctim: syscall.NsecToTimespec(now.UnixNano()),        // ChangeTime
	}

	mfi := mockfs.NewMockFileInfo(ctrl)
	// ModTime is mocked to be different from Atim and Ctim to verify fillFromSys takes precedence.
	mfi.EXPECT().ModTime().Return(modTime).AnyTimes()
	mfi.EXPECT().Sys().Return(mockStat)
	fi := &mockBasicFileInfo{
		FileInfo: mfi,
	}

	// Call ExtendFileInfo with the mock.
	xfi := fsx.ExtendFileInfo(fi)

	// Verify that ChangeTime() matches the value set in mockStat.Ctim.
	// The time.Equal method correctly handles comparisons even if one time
	// has a monotonic clock reading and the other does not.
	if !xfi.ChangeTime().Equal(now) {
		t.Errorf("ChangeTime(): got %v, want %v", xfi.ChangeTime(), now)
	}

	// Verify that AccessTime() matches the value set in mockStat.Atim.
	// This confirms that fillFromSys correctly overwrites the default ModTime
	// with the value from syscall.Stat_t.Atim.
	if !xfi.AccessTime().Equal(accessTime) {
		t.Errorf("AccessTime(): got %v, want %v", xfi.AccessTime(), accessTime)
	}
}
