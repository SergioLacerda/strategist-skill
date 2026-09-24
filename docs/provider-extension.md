# Provider extension guide

**Status:** Draft (v1, local-only provider onboarding)
**Last Updated:** 2026-09-20

This guide describes the local-only v1 path for adding a Strategist provider.
It composes the existing package and adapter contracts; `skill.yaml` is only a
read-only compatibility view and is never an authority for binding, trust, or
lock state.

## Source contract

An already-materialized provider directory contains:

```text
package.yaml   # publisher-owned identity, digest, provenance, version
adapter.yaml   # host compatibility, roles, slots, handoffs, permissions
skill.yaml     # optional legacy/generated compatibility view
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
