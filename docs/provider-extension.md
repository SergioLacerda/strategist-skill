# Provider extension guide

**Status:** Draft (v1, local-only provider onboarding)
**Last Updated:** 2026-09-20

This guide describes the local-only v1 path for adding a Strategist provider.
It composes the existing package and adapter contracts. A legacy `skill.yaml`
may remain in a source directory for migration evidence, but it is not copied
to the runtime and is never an authority for binding, trust, or lock state.

## Source contract

An already-materialized provider directory contains:

```text
package.yaml   # publisher-owned identity, digest, provenance, version
adapter.yaml   # host compatibility, roles, slots, handoffs, permissions
skill.yaml     # optional legacy source evidence; never generated at runtime
```

`package.yaml` must validate as `domain.PluginPackage`, and `adapter.yaml` as
`domain.AdapterContract`. The package and adapter IDs must match. Roles map to
their fixed slots: Ranger to `discovery`, Archivist to `refinement`, and Sniper
to `execution`.

The v1 source boundary accepts local directories only. Git references, HTTP
URLs, registries, and marketplace acquisition are deliberately deferred.

## Validate, onboard, and verify

Validate is read-only:

```bash
strategist provider validate ./my-provider --format table
strategist provider validate ./my-provider --format json
```

The report includes package and adapter digests, roles, slots, contract
readiness, permission/dependency evidence, and `live_invocation`. Static
contract success never certifies live invocation; without an authorized probe,
that dimension remains `unknown` with `live_probe_not_run`.

Onboard a compatible provider into an installed workspace:

```bash
strategist provider add ./my-provider --slot refinement
```

The command stages the source under `.strategist/providers/`, updates the
existing `.strategist/plugins.lock` inventory and binding generation, compiles
the workspace, and records the transaction in
`.strategist/provider-transactions.yaml`. Lock writes and materialization are
restored when staging or compilation fails, so the previous last-known-good
binding remains active.

`provider add` does not edit `active.yaml`. It prints the line to set, for example
`active.yaml: set slots.refinement to my-provider@1.0.0`, and `strategist check`
keeps reporting the slot as diverging from `plugins.lock` until you do. Name the
installed instance (`<package-id>@<version>`), not the bare package id: `check`
rejects the package id with `custom_package_use_instance_id`. A package added this
way is a `custom` binding, even over a slot whose previous binding was `ranked`.

For the `discovery` and `refinement` slots the adapter must declare
`risk_score: write_analysis`; `provider add` refuses a package that declares none
or another value (`analysis_risk_missing`, `analysis_risk_mismatch`), because
`check` would block it at mission time. The `execution` slot keeps its own rule.

A package whose adapter requests permissions stays blocked in `check` with
`permission_grant_missing` until a grant for its adapter digest exists in
`.strategist/permission-grants.yaml`. There is no CLI to create grants yet, so the
file is written by hand; a package whose lock entry has no adapter digest is
blocked with `custom_package_digest_missing`.

The native Ranger role owns discovery, but its selected Weapon is invoked through
the role contract:

```text
strategist provider add ./my-provider --slot discovery
```

is accepted only when the provider satisfies the discovery contract and Ranger can
invoke and normalize its untrusted result. Missing or incompatible invocation fails
closed with `role_invocation_failed`; there is no silent native substitution. A
catalog entry or static manifest is not proof of runtime invocation.

## Upgrade and rollback

Re-run `provider validate` before onboarding a new version. Each installed
instance is keyed by provider ID and version, while the lock pins package and
adapter digests. Inspect `plugins.lock` and
`provider-transactions.yaml` when diagnosing a failed add. A failed transaction
is recorded as `rolled_back`; do not hand-edit the lock or generated mirrors.

This path intentionally does not implement remote acquisition, trust-root
distribution, or a catalog-wide audit. Those are separate design decisions and
must not be inferred from local static validation.
