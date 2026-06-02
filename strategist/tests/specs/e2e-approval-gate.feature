Feature: Strategist Approval Gate E2E
  Invariant: execution never starts without explicit approval.

  Scenario: gate accepts and execution proceeds
    Given the approval gate has been presented
    When the user responds with "yes"
    Then Strategist proceeds to execution
    And the mission result is completed

  Scenario: gate declines and no execution happens
    Given the approval gate has been presented
    When the user responds with "no"
    Then Strategist ends the mission as plan_only
    And Sniper is not invoked

