Feature: Strategist Opportunity Attack E2E
  Invariant: only Archivist performs Opportunity Attack, and it only evaluates ADR side quests.

  Scenario: Ranger records scope observations and hands off
    Given the user asks for an implementation
    When Ranger evaluates the request
    Then Ranger records scope observations
    And Ranger hands off to Archivist

  Scenario: Archivist evaluates ADR need and presents the gate
    Given Ranger has already handed off to Archivist
    When Archivist evaluates the request
    Then Archivist runs opportunity attack
    And Archivist finds no ADR side quest
    And Strategist presents the approval gate

  Scenario: Opportunity Attack does not close analysis cards
    Given a refined package appears implemented
    When Archivist runs opportunity attack
    Then Archivist only evaluates ADR criteria
    And Critical Hit remains responsible for any pending/refined to done closure
