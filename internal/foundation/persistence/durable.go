package persistence

import (
	"fmt"
	"io/fs"
	"os"
)

// AppendSynced appends bytes to a file, syncs the file, and closes it. The
// caller owns record framing and the transaction order around the append.
func AppendSynced(path string, body []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, mode)
	if err != nil {
		return err
	}
	writeErr := writeAll(file, body)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// WriteSynced truncates or creates a file, writes all bytes, syncs the file,
// and closes it. It does not sync the containing directory.
func WriteSynced(path string, body []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	writeErr := writeAll(file, body)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// SyncDir requests persistence of directory-entry changes made within path.
func SyncDir(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func writeAll(file *os.File, body []byte) error {
	for len(body) > 0 {
		n, err := file.Write(body)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("write made no progress")
		}
		body = body[n:]
	}
	return nil
}
