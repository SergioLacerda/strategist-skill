Feature: Strategist Happy Path E2E
  Invariant: the user-requested implementation path must preserve treasure chest consultation, opportunity attack checks, and the approval gate.

  Scenario: user requests implementation and Ranger and Archivist find no side work
    Given a local Strategist workspace is installed
    And the project has treasure chests configured
    When the user asks for an implementation
    Then Ranger evaluates the request
    And Ranger consults treasure chests
    And Ranger runs opportunity attack
    And Ranger finds no actionable side quests
    And Ranger hands off to Archivist
    And Archivist evaluates the request
    And Archivist consults treasure chests
    And Archivist runs opportunity attack
    And Archivist finds no actionable side quests
    And Strategist presents the approval gate

  Scenario: gate decline ends the mission as plan_only
    Given the approval gate has been presented
    When the user responds with "no"
    Then Strategist ends the mission as plan_only
    And Sniper is not invoked

