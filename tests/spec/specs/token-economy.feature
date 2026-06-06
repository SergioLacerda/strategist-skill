Feature: Token Economy — Mode Inference and Triage Gate
  Invariant: prompt-intake infers token_strategy before discovery opens.
  Source: skill.yaml token_economy_stop_conditions, prompt-intake/skill.yaml behavior.

  Scenario: triage_gate_blocked — discovery blocked when acceptance criteria missing
    Given a user prompt with task_type=feature and no acceptance criteria
    When prompt-intake runs the mode inference algorithm
    Then token_strategy.triage_gate.blocked is true
    And token_strategy.triage_gate.blocking_question is present
    And the pipeline does not advance to context_enrichment

  Scenario: mode_upgraded_by_pressure — balanced upgraded to deep by uncertainty signals
    Given a user prompt mentioning multiple systems with ambiguous language
    When prompt-intake runs the mode inference algorithm
    Then pressure_score is >= 2
    And token_strategy.mode is deep

  Scenario: lean_bug_no_gate — lean bugfix bypasses triage gate
    Given a user prompt classified as task_type=bugfix with clear scope
    When prompt-intake runs the mode inference algorithm
    Then token_strategy.mode is lean
    And token_strategy.triage_gate.blocked is false

  Scenario: skip_triage_gate — pipeline advances to discovery while triage is blocked
    Given token_strategy.triage_gate.blocked is true
    When Strategist advances to context_enrichment without presenting the blocking_question
    Then Strategist detects drift pattern "skip_triage_gate"
    And stops the pipeline
    And presents the blocking_question to the user
