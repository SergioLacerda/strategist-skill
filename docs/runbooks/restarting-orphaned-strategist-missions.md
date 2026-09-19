# Runbook: Restarting an Orphaned Strategist Mission

## Trigger

Use this runbook when a requested Strategist mission points to an analysis file
whose frontmatter is still `mission_status: ranger_pending`, but the prior
Ranger execution is no longer active and the operator wants to restart the
mission using the existing material.

This procedure is a controlled recovery. It does not treat `ranger_pending` as
permission to skip discovery, and it never promotes an incomplete artifact by
changing its status to `ranger_done`.

## Safety boundary

- First determine whether another agent/session is still writing the mission.
  If activity is present or cannot be ruled out, stop with
  `blocked reason=mission_in_progress`.
- Preserve the original artifact before replacing or regenerating the
  canonical pending analysis.
- Reuse facts, constraints, source hints, and partial findings as untrusted
  input. Revalidate them against the current workspace.
- The restarted Ranger must produce a complete canonical analysis and set
  `mission_status: ranger_done` only after its required sections and handoff
  fields are complete.
- Do not manually set `ranger_done`, `archivist_done`, or a gate state to bypass
  a missing phase.

## Recovery steps

1. **Confirm the target and runtime.**

   ```sh
   test -d .strategist
   strategist check --json
   sed -n '1,80p' .strategist/active.yaml
   ```

   Stop if preflight is blocked. Use the configured `base_path`; do not infer
   `.analysis` when the active configuration says otherwise.

2. **Inspect the mission state and concurrent activity.**

   ```sh
   sed -n '1,80p' .analysis/pending/<mission-id>-analysis.md
   rg -n "<mission-id>|mission_in_progress|ranger_pending" \
     .strategist .analysis --glob '*.jsonl' --glob '*.yaml' --glob '*.yml' --glob '*.md'
   ```

   If a live agent, lock, control record, or active session owns the mission,
   stop and let it finish. A missing lock is not proof that concurrent activity
   is absent; confirm through the session/operator context.

3. **Create a recoverable snapshot.**

   Copy the pending artifact to a clearly named recovery location before
   regeneration. Keep the snapshot read-only and do not treat it as a second
   canonical mission artifact. Record the recovery timestamp and operator in
   the session notes or approved incident record.

   The snapshot must preserve the original frontmatter and body exactly. If the
   workspace policy does not permit a recovery folder, store the copy outside
   the canonical artifact root and record its path without committing secrets.

4. **Classify reusable material.**

   Reuse only as input to the restarted Ranger:

   - mission objective, constraints, and expected output;
   - known facts that still match current source evidence;
   - source paths and search hints, after checking they still exist;
   - partial uncertainties, risks, and side quests.

   Do not reuse as proof:

   - `ranger_pending` as evidence that discovery completed;
   - stale fingerprints, old command output, or unchecked implementation claims;
   - a provider scratch file as a canonical analysis;
   - a previous `ranger_done`, gate, or execution status copied from another
     mission.

5. **Restart Ranger with the preserved material as context.**

   Run the configured discovery route again. The restarted Ranger must follow
   the normal retrieval cascade, consult applicable treasure chests, refresh
   source evidence, normalize the result, and write exactly one canonical
   pending artifact. It may retain valid material from the snapshot, but must
   update stale facts and explicitly record remaining uncertainties.

6. **Validate the restarted handoff.**

   Confirm that the new artifact contains non-empty `mission_objective`,
   `known_facts`, `uncertainties`, `affected_scope`, `side_quests`,
   `scope_observations`, and `recommended_refinement_focus` sections. Confirm
   the frontmatter is now `mission_status: ranger_done` and that the Ranger →
   Archivist handoff schema validates.

7. **Resume the normal pipeline.**

   Archivist may now consume the fresh Ranger artifact, create the four-file
   refined package, validate its handoff, and promote the analysis. Do not run
   Archivist concurrently with the restarted Ranger.

8. **Apply the approval and execution boundaries.**

   Present the refined package at the Strategist Approval Gate. Approval allows
   only declared documentation targets for Sniper; `implementation_handoff`
   items remain a separate coding task. Never interpret recovery authorization
   as source-code mutation authorization.

## Stop conditions

- `strategist check --json` is blocked or the active runtime is missing.
- Another session may still own the mission.
- The original artifact cannot be snapshotted without data loss.
- Required source evidence is unavailable and the uncertainty materially
  changes the mission scope.
- The restarted artifact lacks required sections or handoff fields.
- A provider attempts to write outside the configured workspace artifact roots
  or writes planning output to `docs/plans/`.

## Verification

```sh
strategist check --json
test ! -e .analysis/pending/<mission-id>-analysis.md \
  || rg -q '^mission_status: ranger_done$' .analysis/pending/<mission-id>-analysis.md
find .analysis/refined/<mission-id> -maxdepth 1 -type f -print
```

For an Archivist handoff, also run the provider's strict validation from its
private runtime and confirm the four canonical files exist under the configured
`base_path`.

## References

- `.strategist/contracts/narrative/03-discovery.md` — Ranger state and handoff
  protocol.
- `.strategist/contracts/narrative/04-refinement.md` — promotion and Archivist
  handoff protocol.
- `.strategist/contracts/machine/mission-status.yaml` — mission state model.
- `docs/runbooks/role-invocation-failed.md` — provider invocation failures.
