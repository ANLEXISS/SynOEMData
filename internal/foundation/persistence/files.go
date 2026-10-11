// Package persistence provides the local durable file operations used by
// Foundation state owners. It deliberately knows nothing about cognitive
// schemas or Universal Store records.
package persistence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrDirectorySyncAfterReplace identifies an error after rename has installed
// the new file. The replacement is visible, but power-loss durability is
// uncertain.
var ErrDirectorySyncAfterReplace = errors.New("replacement installed but directory sync failed")

// ReplaceSynced writes a temporary file, syncs it, renames it over the
// destination, then syncs the containing directory. Its ordering matches the
// Universal Store's established materialized-state write path.
func ReplaceSynced(dir, filename, tempPattern string, body []byte) error {
	tmp, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := writeAll(tmp, body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(dir, filename)); err != nil {
		return err
	}
	if err := SyncDir(dir); err != nil {
		return fmt.Errorf("%w: %w", ErrDirectorySyncAfterReplace, err)
	}
	return nil
}
