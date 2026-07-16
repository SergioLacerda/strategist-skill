Feature: Strategist Treasure Chest E2E
  Invariant: treasure chests are consulted before the slot work continues.

  Scope note: this feature covers mission-level chest *consultation* (Ranger/Archivist
  loading configured chests) only. `treasure-chest index`/`mine` CLI subcommand behavior —
  candidate detection, proposed-jewel curation, migration — is intentionally out of scope
  here; it is covered by tests/spec/jewel_lifecycle_spec_test.go instead.

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

