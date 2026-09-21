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
| `contracts` | `tests/evals/contracts/archivist_handoff_schema_valid_test.go` | `archivist-handoff-schema-valid` | handoff-archivist-to-sniper.schema.yaml exists, parses as YAML, and declares required_fields |
| `contracts` | `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go` | `critical-hit-closure-report-shape-valid` | a Critical Hit closure completion report fixture has all four required fields |
| `contracts` | `tests/evals/contracts/progress_event_schema_valid_test.go` | `progress-event-schema-valid` | progress-contract.yaml exists, parses as YAML, and declares event_format |
| `contracts` | `tests/evals/contracts/ranger_artifact_shape_valid_test.go` | `ranger-artifact-shape-valid` | a Ranger analysis artifact fixture has correct frontmatter and all seven required sections |
| `contracts` | `tests/evals/contracts/reject_gate_acceptance_as_code_mutation_test.go` | `reject-gate-acceptance-as-code-mutation` | direct_execute is blocked whenever the request touches source code, regardless of gate acceptance |
| `contracts` | `tests/evals/contracts/reject_implementation_handoff_as_sniper_task_test.go` | `reject-implementation-handoff-as-sniper-task` | Sniper's execution slot cannot write a .go file — only its declared documentation prefix/extension |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `chest-grade-valid-fields-allowed` | a chest grade with all-enumerated field values passes validation |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `chest-grade-invalid-source-grade-blocked` | a chest grade with an out-of-enum source_grade is rejected |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `jewel-trust-within-chest-tier-allowed` | a jewel at the same trust tier as its parent chest is allowed |
| `scenarios` | `tests/evals/scenarios/treasure_chest_grading_test.go` | `jewel-trust-exceeds-chest-tier-blocked` | a jewel claiming a more-trusted tier (T0) than its T2 parent chest is rejected |
| `scenarios` | `tests/evals/scenarios/treasure_chest_scope_filter_test.go` | `ranger-uses-discovery-scope-ignores-execution-scope` | filtering by 'discovery' selects discovery- and all-scoped chests, excludes execution-only chests |
