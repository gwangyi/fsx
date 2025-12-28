//go:build unix

package osfs

import (
	"io/fs"
	"os/user"
	"strconv"
)

// lookupUid returns uid associated to given username.
//
// If the username is numeric, it considers the username as a uid.
func lookupUid(username string) (int, error) {
	if username == "" {
		return -1, nil
	}
	uid, err := strconv.Atoi(username)
	if err == nil {
		return uid, nil
	}

	u, err := user.Lookup(username)
	if err != nil {
		return 0, err
	}

	// We don't expect an error here, because unix always has numeric UID.
	return strconv.Atoi(u.Uid)
}

// lookupGid returns gid associated to given group.
//
// If the group is numeric, it considers the username as a gid.
func lookupGid(group string) (int, error) {
	if group == "" {
		return -1, nil
	}
	gid, err := strconv.Atoi(group)
	if err == nil {
		return gid, nil
	}

	g, err := user.LookupGroup(group)
	if err != nil {
		return 0, err
	}

	// We don't expect an error here, because unix always has numeric GID.
	return strconv.Atoi(g.Gid)
}

// Chown changes the numeric uid and gid of the named file within the filesystem's root.
// It resolves the provided string `owner` and `group` names to their corresponding
// numeric IDs using the `fsx.UserId` and `fsx.GroupId` helper functions.
//
// This method delegates to `os.Root.Chown`, ensuring that the operation is securely
// confined to the `osfs` instance's root directory.
//
// Parameters:
//
//	name:  The path to the file, relative to the confined root.
//	owner: The username of the new owner.
//	group: The groupname of the new group.
//
// Returns:
//
//	An error if the user or group cannot be resolved, or if the underlying `chown`
//	operation fails (e.g., permission denied, file not found).
func (fsys filesystem) Chown(name, owner, group string) error {
	uid, err := lookupUid(owner)
	if err != nil {
		return &fs.PathError{Op: "chown", Path: name, Err: err}
	}
	gid, err := lookupGid(group)
	if err != nil {
		return &fs.PathError{Op: "chown", Path: name, Err: err}
	}

	return fsys.Root.Chown(name, uid, gid)
}

// Lchown changes the numeric uid and gid of the named symbolic link within the filesystem's root.
// It behaves like `Chown`, but if the named file is a symbolic link, it changes the
// ownership of the link itself rather than the target file.
//
// It resolves the provided string `owner` and `group` names to their corresponding
// numeric IDs using the `fsx.UserId` and `fsx.GroupId` helper functions.
//
// This method delegates to `os.Root.Lchown`, ensuring that the operation is securely
// confined to the `osfs` instance's root directory.
//
// Parameters:
//
//	name:  The path to the symbolic link, relative to the confined root.
//	owner: The username of the new owner.
//	group: The groupname of the new group.
//
// Returns:
//
//	An error if the user or group cannot be resolved, or if the underlying `lchown`
//	operation fails (e.g., permission denied, file not found).
func (fsys filesystem) Lchown(name, owner, group string) error {
	uid, err := lookupUid(owner)
	if err != nil {
		return &fs.PathError{Op: "chown", Path: name, Err: err}
	}
	gid, err := lookupGid(group)
	if err != nil {
		return &fs.PathError{Op: "chown", Path: name, Err: err}
	}

	return fsys.Root.Lchown(name, uid, gid)
}
