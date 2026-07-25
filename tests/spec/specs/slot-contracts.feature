Feature: Slot Write Scope Contracts
  Invariant: Each slot may only write to its declared scope.
  Source: discovery/refinement contracts — write outside declared scope blocks with slot_write_scope_violation.
  Roles: Ranger=write_analysis, Archivist=write_analysis, Sniper=controlled (documentation only)

  Scenario: Ranger respects write_analysis boundary
    Given Ranger (discovery slot) is executing
    And Ranger is declared with write_scope = "write_analysis"
    When Ranger attempts to write a file outside .analysis/pending/<mission_id>-analysis.md
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

  Scenario: Archivist writes four-file subdirectory correctly
    Given Archivist (refinement slot) is executing
    When Archivist reads .analysis/pending/<mission_id>-analysis.md
    And Archivist writes analysis.md, proposal.md, design.md, tasks.md to .analysis/refined/<mission_id>/
    Then no slot_write_scope_violation is emitted
    And all four files are present after completion
    And .analysis/pending/<mission_id>-analysis.md is removed after completion

  Scenario: Sniper resolves as a native role, not a risk_score-scored skill provider
    Given Sniper is declared as the execution provider in active.yaml
    And roles/sniper.yaml exists with role=sniper and slot=execution
    And skills/sniper/skill.yaml does not exist
    When preflight resolves the execution slot
    Then Sniper resolves through the native role branch (roles/<provider>.yaml + role
      schema validation + slot match) — not through known-providers.yaml or any
      risk_score lookup, since native roles never declare risk_score
    And preflight succeeds without a slot_risk_mismatch check for this slot

  Scenario: Skill provider requires matching risk_score at preflight
    Given a skill provider (e.g. brainstorming) is declared for a slot
    And the slot requires a specific risk_score (write_analysis for discovery/
      refinement, controlled for execution)
    When preflight resolves the skill provider's risk_score from its skill.yaml
    Then risk_score MUST equal the slot's required value
    If risk_score is any other value:
      Then Strategist emits blocked event reason=slot_risk_mismatch slot=<slot>
      And mission does not proceed past preflight

  Scenario: Provider with unknown risk_score is rejected
    Given a roles config declares execution provider "unknown-provider"
    And "unknown-provider" has no skill.yaml declaring risk_score
    And "unknown-provider" has no roles/unknown-provider.yaml native role definition
    When preflight attempts to resolve the provider
    Then Strategist emits blocked event reason=slot_provider_not_found
    And mission stops at preflight

  Scenario: Sniper materializes exactly one documentation target per loop iteration
    Given Archivist produced tasks.md with 3 documentation tasks
    And the review gate has been accepted
    When Sniper begins a documentation materialization loop iteration for task 1
    Then Sniper emits task=1 status=running
    And Sniper emits task=1 status=done
    And Sniper does NOT emit task=2 status=running in the same loop iteration
    And Sniper does NOT emit task=3 status=running in the same loop iteration
    And Strategist updates the task checklist before invoking Sniper again for task 2

  Scenario: Sniper blocked from writing code files
    Given the review gate has been accepted
    When Sniper attempts to write a .go file
    Then Strategist emits documentation_scope_violation reason=code_file_forbidden
    And the write is blocked

  Scenario: Sniper blocked from running Git mutating commands
    Given the review gate has been accepted
    When Sniper attempts to run git commit
    Then Strategist emits documentation_scope_violation reason=git_mutation_forbidden
    And the command is blocked
