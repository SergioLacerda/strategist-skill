Feature: Slot Write Scope Contracts
  Invariant: Each slot may only write to its declared scope.
  Source: SKILL.md §2d — "Any write outside that scope: BLOCK slot_write_scope_violation."
  Roles: Ranger=write_pending, Archivist=write_analysis, Sniper=controlled

  Scenario: Ranger respects write_pending boundary
    Given Ranger (discovery slot) is executing
    And Ranger is declared with write_scope = "write_pending"
    When Ranger attempts to write a file to .analysis/refined/
    Then Strategist emits "slot_write_scope_violation"
    And event.slot = "discovery"
    And the write is blocked
    And mission continues from the current phase

  Scenario: Ranger blocked from writing non-.md files
    Given Ranger (discovery slot) is executing
    When Ranger attempts to write a .sh file to .analysis/pending/
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
    When Archivist writes proposal.md, design.md, tasks.md to .analysis/refined/<mission_id>/
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
