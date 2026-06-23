Feature: SDD Missing Safe Fallback
  Invariant: when the .sdd/ governance directory is absent or corrupted, Strategist must
  default to the most restrictive mode (plan_only + forbidden) rather than failing hard
  or allowing unrestricted execution. The user must be warned.
  Source: protocol.md safe fallback — governance structure incomplete.

  Background:
    Given a local Strategist workspace is installed

  Scenario: .sdd/ directory absent enforces plan_only
    Given the workspace does not have a .sdd/ directory
    When Strategist bootstraps
    Then execution_mode defaults to plan_only
    And git_persistence_mode defaults to forbidden
    And Strategist surfaces a warning about missing governance structure
    And Sniper is not invoked regardless of user gate approval

  Scenario: .sdd/metadata.json missing triggers safe fallback
    Given the workspace has a .sdd/ directory
    But .sdd/metadata.json is absent
    When Strategist bootstraps
    Then Strategist enters safe fallback mode
    And execution_mode is forced to plan_only
    And Strategist warns the user that governance metadata is missing

  Scenario: .sdd/agent-instructions.md missing triggers safe fallback
    Given the workspace has a .sdd/ directory with metadata.json
    But .sdd/agent-instructions.md is absent
    When Strategist bootstraps
    Then Strategist enters safe fallback mode
    And HARD mode rules cannot be loaded
    And Strategist warns the user that HARD mode rules are unavailable
    And execution_mode is forced to plan_only

  Scenario: safe fallback does not hard-fail the mission
    Given any of the .sdd/ missing conditions above
    When Strategist encounters the missing governance structure
    Then the mission continues rather than aborting with a fatal error
    And the user can still receive a plan_only analysis result
