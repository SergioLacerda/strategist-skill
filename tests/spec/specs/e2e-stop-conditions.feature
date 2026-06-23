Feature: Pipeline Stop Conditions
  Invariant: specific conditions must cause the pipeline to halt immediately and report
  the stop reason to the user. The mission must not proceed past the stop condition.
  Source: protocol.md — Stop Conditions (Immediate Halt).

  Background:
    Given a local Strategist workspace is installed

  Scenario: slot_provider_not_found stops pipeline at preflight
    Given the roles config declares a slot provider that cannot be resolved
    And the provider skill.yaml is absent from all search paths
    When Strategist runs preflight
    Then the pipeline stops at preflight
    And Strategist emits reason=slot_provider_not_found
    And the mission does not proceed to intake

  Scenario: slot_risk_mismatch stops pipeline at preflight
    Given a roles config declares execution provider with risk_score "uncontrolled"
    And the slot requires risk_score "controlled"
    When Strategist runs preflight
    Then the pipeline stops at preflight
    And Strategist emits blocked event reason=slot_risk_mismatch slot=execution
    And the mission does not proceed past preflight

  Scenario: preflight_failed stops pipeline before intake
    Given preflight encounters a fatal configuration error
    When Strategist attempts to proceed to intake
    Then Strategist emits reason=preflight_failed
    And the mission halts before intake begins

  Scenario: discovery_failed stops pipeline before refinement
    Given the discovery slot fails to produce the handoff artifact
    When Ranger completes without writing the analysis artifact
    Then Strategist emits reason=discovery_failed
    And the pipeline does not advance to the refinement phase
    And Archivist is not invoked

  Scenario: refinement_failed stops pipeline before approval gate
    Given Ranger produced a valid analysis artifact
    And the refinement slot fails to produce the refined package
    When Archivist completes without writing tasks.md
    Then Strategist emits reason=refinement_failed
    And the pipeline does not advance to the approval gate
    And Sniper is not invoked

  Scenario: pipeline_bypass_detected halts mission immediately
    Given a Sniper is attempting to mutate the repository
    And no canonical pipeline evidence exists for the current mission
    When the mutation is attempted
    Then Strategist emits reason=pipeline_bypass_detected
    And the mutation is blocked
    And the mission halts immediately
