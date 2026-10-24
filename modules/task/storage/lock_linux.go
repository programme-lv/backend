//go:build linux

package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// Lock exclusively owns this storage root until the returned file is closed.
// Both the server and maintenance commands hold it for their entire lifetime.
func Lock(root string) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(root, ".task-storage.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("storage is in use; stop the backend before maintenance: %w", err)
	}
	return f, nil
}
