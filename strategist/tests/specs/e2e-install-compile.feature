Feature: Strategist Install and Compile E2E
  Invariant: install preserves user edits unless force is requested, and compiled artifacts are fresh immediately after compile.

  Scenario: first install prepares the workspace
    Given an empty local workspace
    When the user runs strategist install
    Then .strategist/active.yaml is created
    And .strategist/SKILL.md is created
    And .strategist/knowledge.index.yaml is created

  Scenario: second install preserves customized active.yaml
    Given a workspace with a customized .strategist/active.yaml
    When the user runs strategist install again without --force
    Then the customized active.yaml content is preserved

  Scenario: force install rewrites active.yaml
    Given a workspace with a customized .strategist/active.yaml
    When the user runs strategist install with --force
    Then active.yaml is rewritten from the embedded epic template

  Scenario: compile makes artifacts fresh and check-stale reports freshness
    Given a workspace installed by strategist
    And the workspace has a valid active.yaml for validation
    When the user runs strategist compile
    Then the compiled artifacts are written
    And strategist check-stale reports the compiled config as fresh

