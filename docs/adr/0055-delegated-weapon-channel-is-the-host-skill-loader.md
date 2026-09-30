# ADR-0055 — A delegated Role reaches an embedded Weapon through the host skill loader

> Superseded for Ranked internal Weapons by the build-owned Role/Weapon registry
> and Embedded prompt-bridge design in `docs/plans/2026-09-29-binding-weapons-runtime.md`.
> Custom Weapons retain the explicit host-connector policy described below.

**Status:** Superseded for Ranked internal Weapons. The historical decision remains
useful as evidence for the former delegated-host behavior; it is not an authority
for current Ranked dispatch.
**Date:** 2026-09-26
**Mission:** `20260926-embedded-weapon-channel-refinement` (extends [ADR-0029](0029-external-skill-provider-lifecycle.md);
related: [ADR-0027](0027-refinement-native-role-for-light-client.md), [ADR-0035](0035-embedded-weapon-fallback-policy.md))

## Context

The contracts say a Ranked Weapon is reached through "the embedded connector" and a Custom Weapon through its
explicitly selected host connector. When a Role (Ranger, Archivist) runs as a delegated sub-agent, the Ranger of
this mission measured what actually happens:

- No production code constructs `EmbeddedWeaponConnector`, `HostWeaponConnector` or `NativeRoleConnector`, and
  nothing implements `Invoker`. The only two callers of `Invoke` are not reachable from any binary. The CLI has no
  command that invokes a Weapon. Only `UnsupportedConnector`, `NativeRuntimeConnector` and `LocalPathConnector`
  run in production, and the `Invoke` of the last two does not claim invocation.
- The "invocation evidence" a connector returns is a string it fills itself; its tests assert the string's shape,
  not that anything ran.
- `brainstorming` is a prompt-only Weapon: its mirror under the runtime is a metadata file with no instructions
  and it has no runtime entry. `openspec-propose` differs: it has an executable runtime.
- The channel that a delegated sub-role really uses is the host's skill loader, resolving a copy of the skill the
  host owns. The contract text disagreed with itself about this, and `strategist check` reports "ready" from
  static certification only.

The same finding shows the risk of the alternative that [ADR-0029](0029-external-skill-provider-lifecycle.md) already
rejected, vendoring an upstream `SKILL.md`, is smaller than assumed for certification but does not create a channel.

## Decision

1. **Name the channel (option a+).** For a delegated run, the Weapon is reached through the host skill loader. This
   is a *defined degrade* of the embedded channel. It is not a claim of live embedded invocation, and the contracts
   no longer say Ranked uses the embedded connector. The Weapon is still invoked, its output is still untrusted and
   normalized by the Role, and `native_substitution: forbidden` is unchanged.
2. **Record it.** `weapon_invocation` is required for a delegated run and optional otherwise, with the fields
   `invoked`, `resolved_from`, `steps_dropped` and `resolved_digest`. `resolved_digest` is `sha256:<64 hex>` of the
   raw bytes of the file the loader served. It is self-reported.
3. **No comparison rule yet.** `resolved_digest` is not compared with the catalog pin by any rule. Whether the pin is
   a raw sha256 of the upstream `SKILL.md` is open (Q-01); until answered, no text may say the host copy matches the pin.
4. **The contradiction is fixed in the same change.** The agent-protocol discovery routing states this one channel.
5. **Where it lives.** Narrative (`03-discovery.md`, the agent-protocol template) and the Ranger handoff schema only.
   `roles/ranger.yaml` is not edited: the catalog `certification_digest` of the bound Weapons depends on role files,
   and this change must not force a recertification.
6. **Scope.** The requirement is for the discovery Role. Extending it to the Archivist's channel to
   `openspec-propose` is a separate analysis.

## Alternatives considered

| Option | Verdict | Cost and reason (as measured by the Ranger) |
|---|---|---|
| (b) an embedded `Invoker` reachable from the CLI | deferred | Needs an RFC and recertification. The CLI holds no model, so it would only produce evidence the caller fills in, the fabricated guarantee the native-role connector already refuses. Meaningful only for a Weapon with an executable. |
| (c) fill the runtime mirror with the Weapon's instructions | not done in isolation | Changes the mirror, its source and three package digests in the lock; `certification_digest`, `host_api_digest` and `connector_digest` stay unchanged and the roster drift test fails until regenerated. It creates no channel: nothing tells a Role to read the mirror. |
| (d) reclassify prompt-only Weapons as `runtime.kind: host` | waits for Q-02 | Whether Ranked validation accepts it is unanswered. |

The `connector_digest` covers the connector sources and is the same for both Ranked Weapons, so editing those files
recertifies both. That is the main reason to keep this decision out of the connector code.

## Consequences

- A delegated Ranger's artifact says where its Weapon came from and which steps it dropped, so the parent and the
  Archivist no longer have to guess.
- The guarantee is weaker than the contract once implied: the host copy is neither pinned nor certified, and the digest
  is self-reported. The contract now says so.
- Code that no production path reaches still contributes to `connector_digest`. Whether to remove it is a separate
  mission, not decided here.
- No Go code enforces `weapon_invocation`; a contract test pins the wording and the schema field.

## Current Ranked boundary

The current build flow embeds the canonical Weapon payload and generates the
Role, Weapon, and Ranked binding registry before compiling the CLI. A Ranked
binding is materialized into `plugins.lock` with its source digest, execution
mode, connector, entrypoint, and binding digest. Runtime resolution compares
that lock record with the compiled registry and fails closed on drift.

For an Embedded `prompt_bridge` Weapon, the host agent may supply model
execution through the explicitly registered bridge, but it receives the
payload selected by the compiled registry. It must not load
`/home/.../skills`, `external-skills-source`, or `skill-for-hire`, and it must
not replace the selected Weapon with a native Role. A missing bridge is
`role_invocation_failed`, not permission to use the historical host loader.

## Open questions

- **Q-01:** is `upstream_content_digest` a raw sha256 of the upstream `SKILL.md`, and is the host copy the pinned
  upstream release? The Ranger measured `4f88d12355e5298fafda0b0114876fcefd158718ed591c60929d4a80976810f5` for the host
  file against the pin `sha256:74edf03ea6d24ef53db48677b93558d14a979bdf052ca3f57ecdca0c66791608`. They differ; that alone
  does not say which algorithm the pin uses. It blocks any comparison check.
- **Q-02:** can a prompt-only Ranked Weapon be `runtime.kind: host` without breaking Ranked validation? It blocks (d).
