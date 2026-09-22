# Runbook: Orchestrating a Strategist Mission Without Silent Compliance Drift

## Trigger

You are the agent acting as Strategist's orchestrator (Scout/Ranger/
Archivist/Gate/Sniper) for a mission — from the moment `strategist check
--json` returns `status: ready` and you begin intake, through to
materialization. This checklist applies to every mission, not a specific
symptom.

## Steps

1. **Before invoking any ranked weapon's own CLI/tooling (a `skill_provider`
   slot, e.g. `openspec-propose`, `writing-plans`), resolve its scratch
   root first.** Read `.strategist/skills/<provider_id>/skill.yaml`'s
   `scratch_root` field. If it is `runtime`, per
   `.strategist/roles/archivist.yaml#canonical.resolve_weapon_scratch_root`:
   ensure `.strategist/weapon-runtime/<provider_id>/` exists, and `cd` into
   it (or otherwise set it as the weapon's working directory) *before*
   running the weapon's CLI. Do not assume the bare, globally-installed
   version of a weapon's CLI (e.g. `/usr/bin/openspec`) will find the right
   project root on its own — many such tools auto-detect their root by
   walking up from the current working directory, and will silently
   initialize a brand-new root wherever you happen to run them from if the
   real one isn't a plain ancestor of your cwd. Verify before writing
   anything: for a tool with a read-only "print resolved root" command
   (e.g. `openspec context`), run it first and confirm the resolved root is
   under `.strategist/`, not the host repository root.
2. **Before presenting the Approval Gate, load
   `contracts/narrative/05-approval-gate.md`** (it is correctly indexed
   under `contracts/index.yaml`'s `by_phase.gate` — load it like any other
   phase-specific contract, don't skip straight to writing a gate prompt
   from memory). Render its full "Gate display format", in particular the
   `🧠 CONFIDENCE` block. Populate it by running `strategist metrics
   confidence --mission <mission_id>` (this is
   `internal/telemetry.LoadConfidenceGateReview` — the same data, one
   command) and mapping its fields directly into the block's
   `policy`/`distribution`/`per-agent`/`claims`/`evidence`/`calibration`/
   `missing/rejected`/`violations` lines. If `implementation_plan` contains
   any `task_type: implementation_handoff` item, render the `⚠️
   IMPLEMENTATION HANDOFF` block before asking for acceptance — omitting it
   is a gate-format violation, not a simplification.
3. **Before narrating any role transition** (`ranger_start`/`_done`,
   `archivist_start`/`_done`, the gate prompt, `sniper_start`/`_done`/
   `_task_done`), resolve and render `role_level_header`. Call `strategist
   leveling label --role <role> --mission <mission_id> --host-model
   <actual running model> --host-effort <actual running effort> --json`
   and use its `rendered` field verbatim as the header — every one of these
   templates begins with `{role_level_header}` by contract; a narration
   line without it is incomplete, not a shorter valid form. **The command
   cannot infer which model is calling it** — always pass `--host-model`
   (and `--host-effort` when known) explicitly; omitting these flags
   returns all-blank fields with no visible warning, so a blank result must
   be treated as "you forgot the flags," not "leveling is unavailable
   here." Once a tuple is recorded for a given `--mission`+`--role`, later
   calls reuse it silently — pass `--reason <why>` only when you
   deliberately want a fresh re-resolution (e.g. after an escalation), not
   on every call.

## Decision Point

**All three steps were followed for every applicable phase of this
mission, and can be pointed to concretely** (a resolved-root check for
every `scratch_root: runtime` weapon invocation, a rendered `🧠 CONFIDENCE`
block at the gate, a rendered `role_level_header` on every role-transition
line): compliant — proceed normally.

**Any step was skipped, assumed, or "will add it in the summary at the
end" instead of applied live, phase by phase:** stop narrating as if the
mission is compliant. Go back and apply the missing step for the phase
where it was skipped before continuing — a corrected recap after the fact
does not substitute for the missing live application, since the whole point
of steps 2–3 is that the *user* sees the confidence/level data at the
decision point (the gate), not only in a retrospective the user didn't ask
for.

## Stop Conditions

- A ranked weapon with `scratch_root: runtime` was invoked without first
  checking (and, if needed, creating) `.strategist/weapon-runtime/
  <provider_id>/` as its working directory — stop, check whether it wrote
  outside `.strategist/`, and flag any resulting stray files to the user
  before continuing (do not silently delete or move them yourself).
- The Approval Gate was presented without a `🧠 CONFIDENCE` block, despite
  confidence claims having been recorded for the mission — not done; the
  gate must be re-presented with the block filled in from `strategist
  metrics confidence --mission <mission_id>`.
- Any role-transition narration line was sent without a resolved
  `role_level_header` — not done; resolve it via `strategist leveling
  label --host-model ... --host-effort ...` before the next narration line.
- `strategist leveling label` returned blank `model`/`effort` fields and
  the blank result was accepted as final instead of retried with
  `--host-model`/`--host-effort` — treat this as a compliance gap to fix
  immediately, not a tool limitation to route around.

## Reference

- Origin: `.analysis/refined/20260921-strategist-drift-scratch-and-metrics/analysis.md`
  — the mission that diagnosed and reproduced all three drifts described
  here, following a user report against the immediately preceding mission
  (`20260921-test-maintenance-runbook`).
- Root-cause detail for the scratch-root finding: `.strategist/roles/
  archivist.yaml#canonical.resolve_weapon_scratch_root`;
  `contracts/index.yaml`'s `by_phase.refinement` does not yet reference
  this role file, which is why an agent following only the phase-contract
  index can miss this step — tracked separately as an `implementation_handoff`
  item (a `contracts/index.yaml` / `internal/embed/defaults/...` fix), not
  resolved by this runbook. This runbook is the mitigation available
  without waiting for that fix.
- Gate display format in full: `.strategist/contracts/narrative/
  05-approval-gate.md` § "Gate display format".
