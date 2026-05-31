Feature: Forbidden Behavior Detection and Self-Correction
  Invariant: Strategist detects and corrects known drift patterns before each phase.
  Source: SKILL.md §Drift Self-Correction — drift-patterns.yaml loaded at preflight.

  Scenario: direct_execution — Strategist performs slot work itself
    Given Strategist is in the discovery phase
    When Strategist begins writing discovery content directly (not via slot provider)
    Then Strategist detects drift pattern "direct_execution"
    And stops the direct write
    And identifies the active slot provider from roles config
    And invokes the provider correctly
    And resumes phase from the correct delegation point

  Scenario: silent_phase_advance — next phase starts without done event
    Given Strategist completed the discovery phase
    When Strategist begins the refinement phase without emitting "[Strategist] phase=analysis status=done"
    Then Strategist detects drift pattern "silent_phase_advance"
    And emits the missing done event
    And only then continues to the refinement phase

  Scenario: approval_bypass — Sniper invoked without gate
    Given Archivist has completed and tasks.md has tasks
    When Strategist invokes Sniper without presenting the approval gate
    Then Strategist detects drift pattern "approval_bypass"
    And stops Sniper invocation immediately
    And presents the approval gate prompt
    And waits for user response before proceeding

  Scenario: side_quest_approval_bypass — ataque de oportunidade moves without mini gate
    Given opportunity_attack produced a non-empty side quest manifest
    When Strategist begins executing file moves without presenting the mini approval gate
    Then Strategist detects drift pattern "side_quest_approval_bypass"
    And stops immediately
    And presents the mini approval gate with the full manifest
    And waits for user response

  Scenario: single_target_sweep_bypass — skipping sweeps due to narrow scope
    Given the mission request targets a single file refinement
    When Strategist skips opportunist attack, treasure check, or sidequest manifest update because of narrow focus
    Then Strategist detects drift pattern "single_target_sweep_bypass"
    And emits blocked event reason=opportunity_sweep_failed
    And does not proceed to next phase until sweep invariants are satisfied

  Scenario: scope_expansion — addressing work outside the mission
    Given an active mission with a specific task_type
    When Strategist begins working on something outside the user's stated mission
    Then Strategist detects drift pattern "scope_expansion"
    And stops the out-of-scope work
    And returns to the current mission scope

  Scenario: route_plan_creation_to_sniper — asking Sniper to write docs
    Given Sniper is about to be invoked
    When the task given to Sniper is to create a spec, analysis, or implementation plan
    Then Strategist detects drift pattern "route_plan_creation_to_sniper"
    And stops the Sniper invocation
    And routes the document authoring to the Archivist (refinement) slot
