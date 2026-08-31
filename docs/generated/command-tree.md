<!--
generated: true
source: strategist --help (recursive walk of the built binary)
generator: scripts/generate-command-tree.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Command Tree

Full command surface of the `strategist` CLI, walked from `bin/strategist --help`.

`strategist` — Strategist — install, compile, and manage the Strategist skill for Claude agents.

- `check` — Pre-mission slot validation
- `check-stale` — Check if a compiled artifact is stale (exit 0=fresh, exit 1=stale)
- `compile` — Compile all skill artifacts from a .strategist/ root
- `completion` — Generate the autocompletion script for the specified shell
  - `bash` — Generate the autocompletion script for bash
  - `fish` — Generate the autocompletion script for fish
  - `powershell` — Generate the autocompletion script for powershell
  - `zsh` — Generate the autocompletion script for zsh
- `dojo` — Health-check scenarios for the Strategist skill
  - `check` — Run offline checks for a dojo scenario
  - `list` — List available dojo scenarios
- `eval` — Strategist eval harness utilities
  - `harvest` — Copy real mission artifacts into tests/evals/regression/ as fixtures
  - `run` — Run the internal/eval scenario battery via go test
- `handoff` — Run and record Handoff Challenge verification
  - `verify` — Verify a Handoff Challenge acknowledgment and record the result
- `help` — Help about any command
- `install` — Install the Strategist skill into a target repository
- `metrics` — Report metrics computed from Strategist's own runtime memory
  - `handoff` — Report Handoff Challenge governance metrics
  - `scout` — Report Scout routing metrics
- `mission` — Report and inspect mission-level facts this binary cannot observe directly
  - `report-usage` — Record real token usage for a mission, reported by the invoking agent
- `runbook` — Operate on the typed docs/runbooks/*.runbook.yaml corpus
  - `select` — Select applicable runbooks for the given mission signals
- `sync-governance` — Sync .strategist/skill.yaml with active SDD governance mandates
- `treasure-chest` — Show treasure chest runtime status and index health
  - `add` — Register a new treasure chest across active/governed/indexed layers
  - `doctor` — Detect consistency drift across active.yaml, treasure-chests.yaml, and knowledge.index.yaml
  - `index` — Rebuild the offline knowledge substrate (scan, propose jewels, refresh compiled index)
  - `items` — Inspect and curate jewels and potions regardless of curation status
    - `accept` — Promote item ids to status: accepted
    - `check-evidence` — Advisory scan for expired, duplicate, or conflicting jewels in a chest
    - `deprecate` — Mark item ids as status: deprecated
    - `list` — List items (jewels and/or potions), optionally filtered by kind, status, or chest
    - `migrate-status` — One-time migration: rewrite legacy status: active jewels to status: accepted
    - `show` — Show a single item's (jewel or potion) full content
    - `verify` — Promote item ids to status: verified (requires --evidence)
  - `list` — Show treasure chest runtime status (explicit name for the default view)
  - `remove` — Tombstone a treasure chest (mark inactive, do not hard-delete)
- `upgrade` — Reconcile an installed .strategist/ runtime against the current embedded defaults
- `validate` — Validate the .strategist/ configuration tree
- `version` — Print the strategist version
