package install

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEmbeddedSkillLockBytesRejectsDigestMismatch(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v2\npackages:\n  - id: sample\n    origin: embedded\n    runtime_kind: embedded\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    normalized_digest: sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: verified\n    normalized_digest_evidence: verified\n    verification_state: declared\n")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(raw), "digest mismatch")
}

func TestValidateEmbeddedSkillLockBytesAcceptsDeclaredEvidence(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v2\npackages:\n  - id: sample\n    origin: embedded\n    runtime_kind: embedded\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    normalized_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: declared\n    normalized_digest_evidence: declared\n    verification_state: declared\n")
	require.NoError(t, validateEmbeddedSkillLockBytes(raw))
}

func TestValidateEmbeddedSkillLockBytesRejectsFabricatedEvidence(t *testing.T) {
	raw := []byte("schema_version: strategist-embedded-skill-lock/v2\npackages:\n  - id: sample\n    origin: embedded\n    runtime_kind: embedded\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    normalized_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    contract_version: skill-package/v1\n    transformation: none\n    original_digest_evidence: fabricated\n    normalized_digest_evidence: verified\n    verification_state: declared\n")
	require.ErrorContains(t, validateEmbeddedSkillLockBytes(raw), "invalid evidence state")
}

func TestValidateEmbeddedSkillLockBytesRejectsLegacyVocabulary(t *testing.T) {
	valid := "schema_version: strategist-embedded-skill-lock/v2\npackages:\n  - id: sample\n    origin: embedded\n    runtime_kind: embedded\n    digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    original_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    normalized_digest: sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n    contract_version: skill-package/v2\n    transformation: none\n    verification_state: declared\n    original_digest_evidence: declared\n    normalized_digest_evidence: declared\n"
	require.NoError(t, validateEmbeddedSkillLockBytes([]byte(valid)))

	legacyRuntime := strings.Replace(valid, "runtime_kind: embedded", "runtime_kind: embedded_skill", 1)
	require.ErrorContains(t, validateEmbeddedSkillLockBytes([]byte(legacyRuntime)), "regenerate or reinstall")

	legacySchema := strings.Replace(valid, "lock/v2", "lock/v1", 1)
	require.ErrorContains(t, validateEmbeddedSkillLockBytes([]byte(legacySchema)), "unsupported schema")

	missingOrigin := strings.Replace(valid, "origin: embedded\n    ", "", 1)
	require.ErrorContains(t, validateEmbeddedSkillLockBytes([]byte(missingOrigin)), "origin")
}
