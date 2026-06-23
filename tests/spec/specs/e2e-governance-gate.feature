Feature: Governance Gate E2E
  Invariant: when SDD governance returns execution_gate=blocked, Strategist must not invoke
  Sniper under any circumstances, regardless of user approval at the persona gate.
  Source: HARD mode Rule 1 — execution_gate=blocked stops the pipeline immediately.

  Background:
    Given a local Strategist workspace is installed

  Scenario: execution_gate blocked prevents Sniper invocation
    Given a mission configured with apply_workspace execution policy
    And SDD governance injection returns execution_gate: blocked
    And the gate_reason is "policy_blocked: governance adapter denied execution"
    When Strategist evaluates the approval gate
    Then Strategist does NOT invoke Sniper
    And the mission resolves as plan_only
    And Strategist surfaces execution_gate=blocked to the user
    And Strategist surfaces the gate_reason to the user

  Scenario: execution_gate blocked overrides user approval
    Given a mission configured with apply_workspace execution policy
    And SDD governance injection returns execution_gate: blocked
    And the user has provided persona gate approval
    When Strategist evaluates whether to invoke Sniper
    Then Strategist does NOT invoke Sniper
    And Strategist reports that governance gate blocked execution
    And Strategist does NOT proceed to execution state

  Scenario: execution_gate allowed with user approval proceeds to Sniper
    Given a mission configured with apply_workspace execution policy
    And SDD governance injection returns execution_gate: allowed
    And the user has provided persona gate approval
    When Strategist evaluates the approval gate
    Then Strategist invokes Sniper
    And the mission proceeds to execution state
