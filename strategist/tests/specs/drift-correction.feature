Feature: Drift Self-Correction via outcomes.jsonl
  Invariant: Strategist uses learning memory to detect recurring failure patterns.
  Source: SKILL.md §8 — LearningBuffer + outcomes.jsonl.

  Background:
    Given .strategist/memory/outcomes.jsonl exists
    And the LearningBuffer is initialized

  Scenario: LearningBuffer flushes at mission start when threshold is reached
    Given outcomes.tmp has >= 20 lines (default buffer_size)
    When a new mission starts (§0 Pre-Bootstrap)
    Then Strategist appends outcomes.tmp to outcomes.jsonl
    And clears outcomes.tmp
    And proceeds with the mission normally
    And the flush happens BEFORE any new outcome is written

  Scenario: LearningBuffer does NOT flush below threshold
    Given outcomes.tmp has < 20 lines
    When a new mission starts
    Then Strategist does NOT flush outcomes.tmp
    And proceeds with the mission normally

  Scenario: Flush failure is non-blocking
    Given outcomes.tmp has >= 20 lines
    And the flush operation fails (e.g., disk full)
    When a new mission starts
    Then Strategist logs the flush failure to stderr
    And proceeds with the mission normally
    And outcomes.tmp is NOT cleared (retry on next mission start)
    And mission result is NOT affected by the failure

  Scenario: outcome appended after mission completes
    Given a mission has completed (status=completed or plan_only)
    When the Learning Phase runs (§8)
    Then learning-curator presents a checkpoint to the user
    And after user confirmation, appends one JSON line to outcomes.tmp
    And does NOT write to outcomes.jsonl directly
    And mission result is returned unchanged regardless of learning phase outcome

  Scenario: Manual flush procedure
    Given outcomes.tmp has accumulated entries
    When user or operator runs the manual flush procedure:
      """
      cat .strategist/memory/outcomes.tmp >> .strategist/memory/outcomes.jsonl
      : > .strategist/memory/outcomes.tmp
      """
    Then all entries are appended to outcomes.jsonl
    And outcomes.tmp is empty
    And outcomes.jsonl is NOT truncated — only appended
