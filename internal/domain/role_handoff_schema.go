package domain

// RoleHandoffSchema mirrors each native role's own handoff_contract
// declaration in roles/<name>.yaml's `canonical:` block (human-readable
// prose there, not machine-parsed). Single source of truth for the
// handoff_schema dimension of RoleContract, shared by internal/install
// (wizard compatibility filtering) and internal/check (live `strategist
// check` compatibility validation) so the two paths cannot silently
// diverge on which schema a role expects (see
// .analysis/refined/20260914-wizard-weapon-options-not-listed/design.md).
// If roles/<name>.yaml/handoff_contract changes, update this map to match.
// Sniper has no entry: it is the pipeline's terminal role and hands
// nothing downstream, so its RoleContract.HandoffSchema is correctly ""
// (the handoff_schema compatibility dimension is then skipped for it,
// imposing no constraint on that dimension).
var RoleHandoffSchema = map[string]string{
	"ranger":    "schemas/handoff-ranger-to-archivist.schema.yaml",
	"archivist": "schemas/handoff-archivist-to-sniper.schema.yaml",
}
