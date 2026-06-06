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

  Scenario: skip_opportunity_attack_routine — role completes without running the routine
    Given the Ranger has completed discovery
    When the Ranger response does not include an Opportunity Attack section
    Then Strategist detects drift pattern "skip_opportunity_attack_routine"
    And surfaces the missing check to the user
    And requests Ranger to re-run before passing to Archivist

  Scenario: suppress_opportunity_attack_feedback — findings hidden from user
    Given Ranger ran opportunity_attack and detected side quests
    When Strategist advances to Archivist without showing opportunity_attack findings to the user
    Then Strategist detects drift pattern "suppress_opportunity_attack_feedback"
    And stops the handoff
    And presents the opportunity_attack findings to the user

  Scenario: hunter_decides_side_quest_strategy — Hunter sets side quest strategy
    Given Hunter detected a side quest during execution
    When Hunter sets side_quest.strategy without returning to Archivist
    Then Strategist detects drift pattern "hunter_decides_side_quest_strategy"
    And voids the Hunter decision
    And routes the side quest to Archivist for strategy decision

  Scenario: single_target_sweep_bypass — skipping sweeps due to narrow scope
    Given the mission request targets a single file refinement
    When Strategist skips opportunity_attack routine because of narrow focus
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
