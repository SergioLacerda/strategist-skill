package install

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEmbeddedSkillLockBytesRejectsDigestMismatch(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v1\npackages:\n  - id: sample\n    digest: sha256:abc\n    original_digest: sha256:def\n    normalized_digest: sha256:abc\n    contract_version: skill-package/v1\n    transformation: none\n    verification_state: declared\n")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(raw), "digest mismatch")
}

func TestValidateEmbeddedSkillLockBytesAcceptsDeclaredEvidence(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v1\npackages:\n  - id: sample\n    digest: sha256:abc\n    original_digest: sha256:abc\n    normalized_digest: sha256:abc\n    contract_version: skill-package/v1\n    transformation: none\n    verification_state: declared\n")
	require.NoError(t, validateEmbeddedSkillLockBytes(raw))
}
