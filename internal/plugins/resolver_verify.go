package plugins

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// VerifyLockDigest checks a lock's internal self-consistency — its declared
// schema version, and whether GraphDigest still matches a fresh digest of its
// own Nodes — without requiring the current candidate set VerifyLock needs
// for full replay verification. A caller that has an on-disk lock but no
// resolved candidate list at hand (e.g. a preflight check reading
// plugins.lock before any resolution runs) can still catch a hand-edited or
// otherwise corrupted lock with this alone: a node's digest or id changed
// without recomputing graph_digest reliably changes the recomputed digest.
func VerifyLockDigest(lock domain.PluginLock) error {
	if lock.SchemaVersion != lockSchemaVersion {
		return fmt.Errorf("lock_schema_unsupported: %s", lock.SchemaVersion)
	}
	if got := DigestLockNodes(lock.Nodes); got != lock.GraphDigest {
		return fmt.Errorf("lock_graph_digest_mismatch: got %s want %s", got, lock.GraphDigest)
	}
	return nil
}
