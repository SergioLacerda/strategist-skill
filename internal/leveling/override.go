package leveling

import (
	"errors"
	"fmt"
	"os"
)

// ReadOverride reads the customer LEVELING override at path. A missing file is
// not an error (nil, false): the embedded defaults and the install authority
// decide what that means. Any other read failure is reported as
// leveling_policy_unreadable. CLI and wizard both use this reader, so the same
// file always yields the same reason code on both surfaces.
func ReadOverride(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is <root>/leveling.yaml, resolved by the caller
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: read %s: %w", ReasonLevelingPolicyUnreadable, path, err)
	}
	return raw, true, nil
}
