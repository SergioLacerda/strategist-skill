package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// atomicWrite writes through a temporary file in the same directory and renames
// it over the target, so a failure leaves the previous file intact and no
// temporary file behind.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+FileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary %s: %w", FileName, err)
	}
	if err := writeAndClose(tmp, data); err != nil {
		return discard(tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return discard(tmp.Name(), fmt.Errorf("replace %s: %w", FileName, err))
	}
	return nil
}

// writeAndClose restricts, fills and syncs the temporary file and always closes it.
func writeAndClose(tmp *os.File, data []byte) error {
	var err error
	for _, step := range []func() error{
		func() error { return tmp.Chmod(0o600) },
		func() error { return writeAll(tmp, data) },
		tmp.Sync,
	} {
		if err = step(); err != nil {
			err = fmt.Errorf("write temporary %s: %w", FileName, err)
			break
		}
	}
	return errors.Join(err, tmp.Close())
}

func discard(name string, cause error) error {
	return errors.Join(cause, os.Remove(name))
}

func writeAll(tmp *os.File, data []byte) error {
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("data: %w", err)
	}
	return nil
}
