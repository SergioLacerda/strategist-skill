Feature: Approval Gate Enforcement
  Invariant: Sniper never executes without explicit user approval.
  Source: SKILL.md §6 — "Invoking Sniper without receiving explicit approval is a forbidden behavior."

  Background:
    Given Archivist has completed successfully
    And tasks.md contains tasks that write outside .analysis/
    And the approval gate has been evaluated

  Scenario: Sniper blocked before approval response
    When Strategist evaluates tasks.md scope
    Then Strategist emits "[Strategist] phase=approval_gate status=pending"
    And Strategist does NOT invoke the Sniper slot
    And Strategist waits for explicit user response

  Scenario: Sniper proceeds after explicit "yes"
    Given the approval gate prompt has been presented
    When user responds with "yes"
    Then Strategist emits "[Strategist] phase=sniper status=running"
    And the execution slot provider is invoked
    And a report artifact is written to .analysis/done/

  Scenario: Mission ends as plan_only after "no"
    Given the approval gate prompt has been presented
    When user responds with "no"
    Then Strategist emits "[Strategist] phase=approval_gate status=plan_only"
    And Sniper is never invoked
    And mission result has status=plan_only
    And discovery and refined plan artifacts are returned

  Scenario: "review" causes plan presentation before re-asking
    Given the approval gate prompt has been presented
    When user responds with "review"
    Then Strategist presents the full content of tasks.md
    And re-presents the approval gate prompt
    And does NOT invoke Sniper until a yes/no response is received

  Scenario: plan_only when tasks.md is empty
    Given Archivist has completed
    And tasks.md is empty or absent
    Then Strategist emits "[Strategist] phase=approval_gate status=plan_only"
    And does NOT present the approval gate prompt
    And does NOT invoke Sniper
