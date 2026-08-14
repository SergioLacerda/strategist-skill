<!--
generated: true
source: tests/evals/{scenarios,contracts}/*_test.go (eval.Scenario{ID, Description} literals)
generator: scripts/generate-eval-scenarios.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Eval Scenarios

Scenario battery run by `strategist eval run` (`go test -tags=eval`).
Extracted from `eval.Scenario{}` struct literals — see
`docs/adr/0021-eval-cli-subcommand.md` for why these stay Go test
files rather than a CLI-loadable format.

| Group | File | Scenario ID | Description |
|---|---|---|---|
| `contracts` | `tests/evals/contracts/accept_execution_after_approval_test.go` | `accept-execution-after-approval` | gate_approved transitions APPROVAL_GATE to EXECUTION |
| `contracts` | `tests/evals/contracts/archivist_handoff_schema_valid_test.go` | `archivist-handoff-schema-valid` | handoff-archivist-to-sniper.schema.yaml exists, parses as YAML, and declares required_fields |
| `contracts` | `tests/evals/contracts/archivist_produces_no_tasks_resolves_as_analysis_test.go` | `archivist-produces-no-tasks-resolves-as-analysis` | archivist_done_no_tasks transitions REFINEMENT to DONE_ANALYSIS, bypassing the approval gate entirely |
| `contracts` | `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go` | `critical-hit-closure-report-shape-valid` | a Critical Hit closure completion report fixture has all four required fields |
| `contracts` | `tests/evals/contracts/progress_event_schema_valid_test.go` | `progress-event-schema-valid` | progress-contract.yaml exists, parses as YAML, and declares event_format |
| `contracts` | `tests/evals/contracts/ranger_artifact_shape_valid_test.go` | `ranger-artifact-shape-valid` | a Ranger analysis artifact fixture has correct frontmatter and all seven required sections |
| `contracts` | `tests/evals/contracts/reject_execution_gate_timeout_test.go` | `reject-execution-gate-timeout` | a gate timeout resolves as analysis_delivered, not execution |
| `contracts` | `tests/evals/contracts/reject_execution_without_approval_test.go` | `reject-execution-without-approval` | the skill must refuse execution when the approval gate denies the mission |
| `contracts` | `tests/evals/contracts/reject_gate_acceptance_as_code_mutation_test.go` | `reject-gate-acceptance-as-code-mutation` | direct_execute is blocked whenever the request touches source code, regardless of gate acceptance |
| `contracts` | `tests/evals/contracts/reject_implementation_handoff_as_sniper_task_test.go` | `reject-implementation-handoff-as-sniper-task` | Sniper's execution slot cannot write a .go file — only its declared documentation prefix/extension |
| `scenarios` | `tests/evals/scenarios/critical_hit_trigger_test.go` | `critical-hit-valid-plain-move-allowed` | a low-risk, ≤5-file, .md-only move between analysis folders is allowed |
| `scenarios` | `tests/evals/scenarios/critical_hit_trigger_test.go` | `critical-hit-valid-closure-move-allowed` | a closure move with an explicit completion claim and supplied evidence is allowed |
| `scenarios` | `tests/evals/scenarios/critical_hit_trigger_test.go` | `critical-hit-conditions-not-met-blocked` | a request with no completion claim or evidence for a closure move is blocked, falling back to main_mission |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `chest-grade-valid-fields-allowed` | a chest grade with all-enumerated field values passes validation |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `chest-grade-invalid-source-grade-blocked` | a chest grade with an out-of-enum source_grade is rejected |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `jewel-trust-within-chest-tier-allowed` | a jewel at the same trust tier as its parent chest is allowed |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `jewel-trust-exceeds-chest-tier-blocked` | a jewel claiming a more-trusted tier (T0) than its T2 parent chest is rejected |
| `scenarios` | `tests/evals/scenarios/treasure_chest_scope_filter_test.go` | `ranger-uses-discovery-scope-ignores-execution-scope` | filtering by 'discovery' selects discovery- and all-scoped chests, excludes execution-only chests |
