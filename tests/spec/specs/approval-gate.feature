Feature: Review Gate Enforcement
  Invariant: Sniper never materializes documentation without explicit user review acceptance.
  Source: SKILL.md §6 — "Invoking Sniper without receiving explicit review acceptance is a forbidden behavior."

  Background:
    Given Archivist has completed successfully
    And tasks.md contains documentation targets
    And the review gate has been evaluated

  Scenario: Sniper blocked before review response
    When Strategist evaluates tasks.md scope
    Then Strategist emits "[Strategist] phase=approval_gate status=pending"
    And Strategist does NOT invoke the Sniper slot
    And Strategist waits for explicit user response

  Scenario: Sniper proceeds after explicit acceptance
    Given the review gate prompt has been presented
    When user responds with "sim"
    Then Strategist emits "[Strategist] phase=execution status=running"
    And the documentation materialization slot provider is invoked
    And a report artifact is written to <base_path>/archived/

  Scenario: Mission ends as analysis_delivered after rejection
    Given the review gate prompt has been presented
    When user responds with "reject"
    Then Strategist emits "[Strategist] phase=approval_gate status=rejected"
    And Sniper is never invoked
    And mission result has status=rejected
    And analysis and refined package artifacts are returned

  Scenario: Mission ends as revision_requested after missing item
    Given the review gate prompt has been presented
    When user responds with "missing_item"
    Then Strategist emits "[Strategist] phase=approval_gate status=revision_requested"
    And Sniper is never invoked
    And pipeline control returns to Archivist

  Scenario: "review" causes analysis presentation before re-asking
    Given the review gate prompt has been presented
    When user responds with "review"
    Then Strategist presents the full content of tasks.md
    And re-presents the review gate prompt
    And does NOT invoke Sniper until an analysis_accepted response is received

  Scenario: analysis_delivered when tasks.md is empty
    Given Archivist has completed
    And tasks.md is empty or absent
    Then Strategist emits "[Strategist] phase=approval_gate status=analysis_delivered"
    And does NOT present the review gate prompt
    And does NOT invoke Sniper

  Scenario: analysis_accepted response triggers documentation materialization
    Given the review gate prompt has been presented
    When user responds with "analysis_accepted"
    Then Strategist records review gate acceptance
    And proceeds to Sniper for documentation materialization
    And Sniper writes only declared documentation_targets

  Scenario: outside base_path targets declared by Archivist are allowed after acceptance
    Given Archivist declared documentation targets outside <base_path>
    And the review gate prompt has been presented with those targets listed
    When user responds with "sim"
    Then Sniper materializes documentation at the declared outside-base-path targets
    And no documentation_scope_violation is emitted

  Scenario: code file target is blocked regardless of acceptance
    Given the review gate was accepted
    When Sniper attempts to write a code file (.go, .ts, .py, etc.)
    Then Strategist emits documentation_scope_violation
    And the write is blocked

  Scenario: Git mutating command is blocked during documentation materialization
    Given the review gate was accepted
    When Sniper attempts to run a git mutating command (add, commit, push, reset)
    Then Strategist emits documentation_scope_violation reason=git_mutation_forbidden
    And the command is blocked
