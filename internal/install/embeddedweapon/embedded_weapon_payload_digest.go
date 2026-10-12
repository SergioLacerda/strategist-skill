package embeddedweapon

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// SourceDigestAt returns the source digest of the embedded Weapon payload at path, or an empty string when it is absent.
func SourceDigestAt(path string) (string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // provider ID is validated catalog input under the defaults root
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read embedded Weapon payload %q: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("sha256:%x", sum), nil
}
