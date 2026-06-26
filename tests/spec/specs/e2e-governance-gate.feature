Feature: Governance Gate E2E
  Invariant: when SDD governance returns execution_gate=blocked, Strategist must not invoke
  Sniper under any circumstances, regardless of user review acceptance at the persona gate.
  Source: HARD mode Rule 1 — execution_gate=blocked stops the pipeline immediately.

  Background:
    Given a local Strategist workspace is installed

  Scenario: execution_gate blocked prevents Sniper invocation
    Given a mission with documentation targets
    And SDD governance injection returns execution_gate: blocked
    And the gate_reason is "policy_blocked: governance adapter denied documentation"
    When Strategist evaluates the review gate
    Then Strategist does NOT invoke Sniper
    And the mission resolves as analysis_delivered
    And Strategist surfaces execution_gate=blocked to the user
    And Strategist surfaces the gate_reason to the user

  Scenario: execution_gate blocked overrides user review acceptance
    Given a mission with documentation targets
    And SDD governance injection returns execution_gate: blocked
    And the user has provided review gate acceptance
    When Strategist evaluates whether to invoke Sniper
    Then Strategist does NOT invoke Sniper
    And Strategist reports that governance gate blocked documentation
    And Strategist does NOT proceed to documentation state

  Scenario: execution_gate allowed with user acceptance proceeds to Sniper
    Given a mission with documentation targets
    And SDD governance injection returns execution_gate: allowed
    And the user has provided review gate acceptance
    When Strategist evaluates the review gate
    Then Strategist invokes Sniper
    And the mission proceeds to documentation materialization state
