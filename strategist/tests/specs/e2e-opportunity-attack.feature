Feature: Strategist Opportunity Attack E2E
  Invariant: both Ranger and Archivist must perform opportunity attack before the next handoff.

  Scenario: Ranger evaluates, finds no side quests, and hands off
    Given the user asks for an implementation
    When Ranger evaluates the request
    Then Ranger runs opportunity attack
    And Ranger finds no actionable side quests
    And Ranger hands off to Archivist

  Scenario: Archivist evaluates, finds no side quests, and presents the gate
    Given Ranger has already handed off to Archivist
    When Archivist evaluates the request
    Then Archivist runs opportunity attack
    And Archivist finds no actionable side quests
    And Strategist presents the approval gate

