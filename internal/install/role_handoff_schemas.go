package install

// roleHandoffSchema mirrors each native role's own handoff_contract
// declaration in roles/<name>.yaml's `canonical:` block (human-readable
// prose there, not machine-parsed) — kept here as the machine-readable
// counterpart PlanRoleProviderMigration needs to populate
// domain.RoleContract.HandoffSchema. If roles/<name>.yaml/handoff_contract
// changes, update this map to match. Sniper has no entry: it is the
// pipeline's terminal role and hands nothing downstream, so its
// RoleContract.HandoffSchema is correctly "" (the handoff_schema
// compatibility dimension is then skipped for it, imposing no constraint on
// that dimension).
var roleHandoffSchema = map[string]string{
	"ranger":    "schemas/handoff-ranger-to-archivist.schema.yaml",
	"archivist": "schemas/handoff-archivist-to-sniper.schema.yaml",
}
