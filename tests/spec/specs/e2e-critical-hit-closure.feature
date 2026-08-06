Feature: Strategist Critical Hit — Plain Move vs Closure Move
  Invariant: Critical Hit has two distinct modes, and closure move is never
  inferred from code alone — it always requires an explicit claim plus
  supplied evidence.

  Scope note: this feature covers the end-to-end *distinction* between plain
  move and closure move at the Gherkin granularity used by the other
  `e2e-*.feature` files in this directory. It intentionally does not
  duplicate `tests/spec/critical_hit_closure_test.go` (which asserts that
  specific narrative/machine contract files contain required phrases) or
  `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go`
  (which validates completion-report.md's schema shape). Like the other
  `e2e-*.feature` files, this is consumed by a Go test helper via
  `strings.Contains` needle checks — not a Cucumber/Godog runner (see
  `docs/test-styles.md`).

  Scenario: plain move requires no evaluation and no evidence
    Given a workspace artifact under pending/ or refined/
    And no completion or validation claim is present
    When Critical Hit's plain-move trigger conditions are satisfied
    Then Strategist relocates the artifact without requiring evidence
    And no completion-report.md is written

  Scenario: closure move requires an explicit claim and supplied evidence
    Given a workspace artifact under pending/ or refined/
    And the user supplies an explicit completion/validation claim
    And the user supplies an evidence summary
    When Critical Hit's closure-move trigger conditions are satisfied
    Then Strategist writes completion-report.md inside the source package
    And the package is moved to done/
    And tasks.md is updated only for the tasks the supplied evidence covers

  Scenario: closure move is never inferred from documentation_applied alone
    Given a main mission reaches mission_status documentation_applied
    And no explicit completion/validation claim was supplied
    Then Strategist does not treat the package as a closure candidate
    And the package remains in refined/ as the normal terminal state
