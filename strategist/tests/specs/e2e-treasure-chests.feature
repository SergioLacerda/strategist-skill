Feature: Strategist Treasure Chest E2E
  Invariant: treasure chests are consulted before the slot work continues.

  Scenario: treasure chests are available to every slot
    Given a local Strategist workspace is installed
    And active.yaml declares treasure chests with scope "all"
    When the mission begins
    Then Ranger consults treasure chests
    And Archivist consults treasure chests

  Scenario: no treasure chests configured
    Given a local Strategist workspace is installed
    And active.yaml declares no treasure chests
    When the mission begins
    Then Strategist emits treasure_chests=none

