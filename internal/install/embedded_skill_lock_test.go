package install

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEmbeddedSkillLockBytesRejectsDigestMismatch(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v1\npackages:\n  - id: sample\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    normalized_digest: sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: verified\n    normalized_digest_evidence: verified\n    verification_state: declared\n")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(raw), "digest mismatch")
}

func TestValidateEmbeddedSkillLockBytesAcceptsDeclaredEvidence(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v1\npackages:\n  - id: sample\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    normalized_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: declared\n    normalized_digest_evidence: declared\n    verification_state: declared\n")
	require.NoError(t, validateEmbeddedSkillLockBytes(raw))
}

func TestValidateEmbeddedSkillLockBytesRejectsFabricatedEvidence(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v1\npackages:\n  - id: sample\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    normalized_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: fabricated\n    normalized_digest_evidence: verified\n    verification_state: declared\n")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(raw), "invalid evidence state")
}
