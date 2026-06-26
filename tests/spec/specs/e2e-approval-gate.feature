Feature: Strategist Review Gate E2E
  Invariant: documentation materialization never starts without explicit review acceptance.

  Scenario: gate accepts and documentation proceeds
    Given the review gate has been presented
    When the user responds with "sim"
    Then Strategist proceeds to documentation materialization
    And the mission result is documentation_applied

  Scenario: gate rejected and no documentation happens
    Given the review gate has been presented
    When the user responds with "reject"
    Then Strategist ends the mission as analysis_delivered
    And Sniper is not invoked

  Scenario: revision requested and Archivist revisits
    Given the review gate has been presented
    When the user responds with "missing_item"
    Then Strategist ends the gate with revision_requested
    And Sniper is not invoked
    And pipeline control returns to Archivist

