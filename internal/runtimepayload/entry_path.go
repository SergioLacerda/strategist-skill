package runtimepayload

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// entryPath validates a runtime tree entry name, applies strip, and returns the
// relative slash path to write; ok is false when the entry is fully stripped.
func entryPath(name string, strip int) (rel string, ok bool, err error) {
	parts, err := entryParts(name)
	if err != nil {
		return "", false, err
	}
	if len(parts) <= strip {
		return "", false, nil
	}
	rel = path.Clean(strings.Join(parts[strip:], "/"))
	if rel == "." || strings.HasPrefix(rel, "../") {
		return "", false, fmt.Errorf("%w: %q", ErrUnsafeArchive, name)
	}
	return rel, true, nil
}

// entryParts rejects absolute, drive/backslash and parent-traversal names.
func entryParts(name string) ([]string, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, `\`) || strings.Contains(name, ":") {
		return nil, fmt.Errorf("%w: %q", ErrUnsafeArchive, name)
	}
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	if slices.Contains(parts, "..") {
		return nil, fmt.Errorf("%w: %q", ErrUnsafeArchive, name)
	}
	for _, segment := range parts {
		if windowsHostileSegment(segment) {
			return nil, fmt.Errorf("%w: %q is not a valid Windows file name", ErrUnsafeArchive, name)
		}
	}
	return parts, nil
}

// windowsHostileSegment reports a path segment Windows cannot store as written:
// a reserved device name (with or without an extension) or a trailing dot or
// space. It is checked on every platform so a payload behaves the same
// everywhere.
func windowsHostileSegment(segment string) bool {
	if segment == "" {
		return false
	}
	if last := segment[len(segment)-1]; last == '.' || last == ' ' {
		return true
	}
	stem := segment
	if dot := strings.IndexByte(segment, '.'); dot >= 0 {
		stem = segment[:dot]
	}
	switch strings.ToUpper(stem) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}
