Feature: Slot Write Scope Contracts
  Invariant: Each slot may only write to its declared scope.
  Source: discovery/refinement contracts — write outside declared scope blocks with slot_write_scope_violation.
  Roles: Ranger=write_analysis, Archivist=write_analysis, Sniper=controlled

  Scenario: Ranger respects write_analysis boundary
    Given Ranger (discovery slot) is executing
    And Ranger is declared with write_scope = "write_analysis"
    When Ranger attempts to write a file outside .analysis/refined/<mission_id>-analysis.md
    Then Strategist emits "slot_write_scope_violation"
    And event.slot = "discovery"
    And the write is blocked
    And mission continues from the current phase

  Scenario: Ranger blocked from writing non-.md files
    Given Ranger (discovery slot) is executing
    When Ranger attempts to write a .sh file to .analysis/refined/
    Then Strategist emits "slot_write_scope_violation"
    And event.reason contains "non-.md type"
    And the write is blocked

  Scenario: Archivist respects write_analysis boundary
    Given Archivist (refinement slot) is executing
    And Archivist is declared with write_scope = "write_analysis"
    When Archivist attempts to write outside .analysis/
    Then Strategist emits "slot_write_scope_violation"
    And event.slot = "refinement"
    And the write is blocked

  Scenario: Archivist writes three-file subdirectory correctly
    Given Archivist (refinement slot) is executing
    When Archivist reads .analysis/refined/<mission_id>-analysis.md
    And Archivist writes proposal.md, design.md, tasks.md to .analysis/refined/<mission_id>/
    Then no slot_write_scope_violation is emitted
    And all three files are present after completion

  Scenario: Sniper requires controlled risk_score at preflight
    Given Sniper is declared in roles config
    When preflight resolves Sniper's risk_score from known-providers.yaml
    Then risk_score MUST equal "controlled"
    If risk_score is any other value:
      Then Strategist emits blocked event reason=slot_risk_mismatch slot=execution
      And mission does not proceed past preflight

  Scenario: Provider with unknown risk_score is rejected
    Given a roles config declares execution provider "unknown-provider"
    And "unknown-provider" is not in known-providers.yaml
    And "unknown-provider" has no skill.yaml declaring risk_score
    When preflight attempts to resolve the risk_score
    Then Strategist emits blocked event reason=slot_risk_mismatch
    And mission stops at preflight

  Scenario: Sniper executes exactly one task per loop iteration
    Given Archivist produced tasks.md with 3 tasks
    And the approval gate has been granted
    When Sniper begins an execution loop iteration for task 1
    Then Sniper emits task=1 status=running
    And Sniper emits task=1 status=done
    And Sniper does NOT emit task=2 status=running in the same loop iteration
    And Sniper does NOT emit task=3 status=running in the same loop iteration
    And Strategist updates the task checklist before invoking Sniper again for task 2
