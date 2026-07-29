Feature: Policy Guardrails Across Side Paths
  Invariant: opportunist attack and main execution all pass through the same policy evaluator and gate semantics.
  Source: SKILL.md §5 and §6, protocol.md guarded transitions.

  Scenario: opportunity execution is skipped when execution transition is policy-blocked
    Given opportunity manifest is non-empty and gate response is yes
    And effective mission policy denies execution transition
    When Strategist evaluates execution before opportunity Sniper invocation
    Then Strategist emits "phase=policy_eval status=blocked" with transition_group=execution
    And Strategist continues to Archivist with execution_skipped_by_policy

  Scenario: main execution requires both gate approval and allowed policy
    Given Archivist produced tasks for execution
    And user approved in the main approval gate
    When Strategist evaluates execution transition before Sniper
    Then Strategist invokes Sniper only if policy_eval status is allowed
    And blocked decisions return analysis_delivered or documentation_skipped_by_policy
