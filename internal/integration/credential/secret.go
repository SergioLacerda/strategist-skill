// Package credential resolves provider secrets from references without ever
// exporting them: a secret lives in memory, prints as redacted and is never put
// into the process environment.
package credential

import (
	"fmt"
	"io"
)

// Secret holds a credential value. Every formatting verb prints a redaction.
type Secret struct{ value string }

// Reveal returns the value for the single place that sends it to the provider.
func (s Secret) Reveal() string { return s.value }

// String redacts the value.
func (Secret) String() string { return "[redacted]" }

// GoString redacts the value for %#v.
func (Secret) GoString() string { return "credential.Secret{[redacted]}" }

// Format redacts the value for every verb.
func (s Secret) Format(f fmt.State, _ rune) {
	if _, err := io.WriteString(f, s.String()); err != nil {
		return
	}
}
