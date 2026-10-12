# Runbook: Repairing a Stale Runtime Without Resetting the Configuration

## Symptom

One of these, usually right after a `strategist check` or `make install`:

- `strategist check` or an install prints `runtime_stale`, `runtime_stale_conflict`,
  `runtime_stale_unknown_manifest`, `runtime_stale_auto_repairable` or `runtime_newer_than_binary`.
- After a reinstall, `strategist check --json` reports a different provider for a slot than the one you chose
  (for example `refinement` is `openspec-propose` instead of `writing-plans`) and still says `status: ready`.
- A running mission fails at `strategist mission complete` with
  `role_invocation_failed: Ranked openspec_root bindings execute through their declared private runtime and mission normalize-openspec, not mission invoke`.
- `.strategist/active.yaml` starts with "Generated from: templates/epic-standalone.yaml" and your customizations
  (slots, language, ADR and runbook roots) are gone.

## Root Cause

Until 2026-10-11, `strategist install --silent --force` rewrote `.strategist/active.yaml` from the epic template
(and regenerated `plugins.lock`), and the stale-runtime diagnostics recommended `--force` without saying so.
Now `--force` overwrites only normative runtime files; rewriting `active.yaml`, `plugins.lock` and `.config.lock`
needs `--reset-config`, which backs the three files up first. The diagnostics recommend
`strategist upgrade --dry-run` then `strategist upgrade`. The wizard is a second path: its defaults come from the
plugin catalog, not from the existing `active.yaml`, so accepting defaults resets everything.

Nothing records who ran which flags. In the incident of 2026-10-11 (15:46) the silent-template writer is
established, but whether `--force` was passed, and by whom, is **unproven**.

## Resolution Steps

1. **Read what is stale.** Run `strategist check --json` and note the diagnostic and the files it names.
2. **Plan without writing.** Run `strategist upgrade --dry-run`. It classifies every file as `managed`,
   `missing`, `auto_upgrade`, `customized` or `orphaned` and writes nothing.
3. **Repair without force.** Run `strategist upgrade`. It writes `missing` and `auto_upgrade` files, preserves
   `customized` ones and snapshots every file it overwrites under `.strategist/.upgrade-backups/<timestamp>/`.
   `strategist upgrade --rollback latest` (or a named timestamp) restores that snapshot. `upgrade` has no
   reference to `active.yaml` in its code.
4. **Before any `--force`, copy the configuration.** Copy `.strategist/active.yaml`, `.strategist/plugins.lock`
   and `.strategist/.config.lock` to a directory outside `.strategist/`. `strategist install --force` rewrites
   `active.yaml` and `plugins.lock` from the template (verified in `applySilentConfig`); `upgrade` snapshots do
   not include them.
5. **Verify after every reinstall.** Run `strategist check --json` and compare each `bindings[].provider` with
   what you expect. Also read `grep -n "refinement:" .strategist/active.yaml`. A `ready` status alone proves
   nothing about which Weapon is bound.
6. **If the binding was reset, restore it through the wizard.** Run `strategist install --wizard` in an
   interactive terminal, choose the Weapon for each slot (for example `writing-plans` Ranked for the Archivist),
   then repeat step 5. This is the path verified on 2026-10-11. Copying the saved files back is **unverified**:
   `plugins.lock` and `ranked-runtimes.yaml` are regenerated together with `active.yaml`.
7. **Missions that were running.** Check `strategist mission requests --mission-id <id>`. A request issued
   before the reset completed normally with `strategist mission complete` once the same Weapon binding was
   restored (observed 2026-10-11: the binding digest matched). Do not reissue it with `mission invoke`: after
   the Ranger outcome was consumed it fails with `handoff_outcome_replayed`.

## Stop Conditions

- `strategist upgrade --dry-run` lists `customized` files you did not expect: stop and inspect them before
  running anything with `--force`.
- `strategist check --json` shows a provider that differs from your choice after the repair: stop and restore
  through the wizard; do not start a mission.
- `runtime_newer_than_binary`: update the binary (`make install`); do not downgrade by reinstalling, and use
  `--allow-downgrade` only for a deliberate rollback.

## What changed in the tooling

Implemented: `--force` no longer rewrites configuration; `--reset-config` is the explicit request and backs the
configuration up under `.strategist/.upgrade-backups/<timestamp>/` (restore with
`strategist upgrade --rollback <timestamp|latest>`); the diagnostics recommend `upgrade --dry-run` then `upgrade`;
every install and upgrade appends one record (command, mode, `force`, `reset_config`, slots before and after) to
`.strategist/memory/install-events.jsonl`, so a reset is attributable.

Also implemented: `strategist check` adds an advisory line (`binding_changed slot=<slot> started_with=<a>
now=<b>`) when a slot's binding differs from the one the most recent mission started with, and
`strategist mission status` prints the same line on stderr for that mission (starting the next mission
re-baselines it); `mission start` records the starting bindings under `.strategist/missions/bindings/`. The
wizard now starts from the current `active.yaml` (language, workspace, documentation roots and the slots, a
Custom binding staying Custom), so entering through every default keeps the workspace, and it backs the
replaced configuration up first.

Still manual: copy the three configuration files before any `--reset-config` is not required any more (it backs
them up), but verify with `strategist check --json` after every reinstall, because a deliberate rebinding is
still reported as drift until the next mission starts.

## Reference

- `internal/install/installer_config.go` (`applySilentConfig`, `resolvePrompter`), `internal/install/wizard/wizard.go`
- `internal/domain/runtime_defaults.go` (`FormatRuntimeStaleDiagnostic`)
- `strategist upgrade --help`, `strategist install --help`
- `.analysis/refined/20261011-binding-reset-investigation/` (the diagnostic and the specification)
- `docs/runbooks/corrupt-mission-state-recovery.md` for a mission whose state, not its binding, was lost
- Location note: written under the configured runbooks root `.analysis/runbooks/`, which `strategist runbook
  select` does not read today; the operator asked for `docs/runbooks/`, so move this file and its sidecar there
  and set `source_doc` accordingly.
