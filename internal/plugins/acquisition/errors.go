// Package acquisition implements isolated, immutable Git package acquisition.
package acquisition

import "fmt"

// Error is a stable, machine-readable acquisition failure.
type Error struct {
	Code   string
	Detail string
}

func (e Error) Error() string {
	if e.Detail == "" {
		return "acquisition_" + e.Code
	}
	return fmt.Sprintf("acquisition_%s: %s", e.Code, e.Detail)
}

func acquisitionError(code, detail string) error {
	return Error{Code: code, Detail: detail}
}
