package calibration

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
)

// HeldOut reports whether a digest belongs to the held-out part. The split is a
// pure function of the digest and a salt, so it is reproducible and every
// mutation of a package inherits the package's side.
func HeldOut(digest, salt string, percent int) bool {
	if percent <= 0 {
		return false
	}
	sum := sha256.Sum256([]byte(salt + "|" + digest))
	return int(binary.BigEndian.Uint32(sum[:4])%100) < percent
}

// Digest is the content digest of a projection: the sorted field names and values.
func Digest(fields Fields) string {
	var text strings.Builder
	for _, name := range sortedKeys(fields) {
		text.WriteString(name + "\x00" + fields[name] + "\x00")
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(text.String())))
}
