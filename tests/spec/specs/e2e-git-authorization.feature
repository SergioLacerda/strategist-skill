Feature: Git Authorization E2E
  Invariant: git state-modifying commands (add, commit, push, reset, merge, rebase) are
  blocked by default. They are only permitted when the user explicitly phrases "commit this"
  (or equivalent) in their message. Approval of the gate alone does NOT authorize git commands.
  Source: HARD mode Rule 2 — git authorization requires explicit user phrase.

  Background:
    Given a local Strategist workspace is installed
    And a mission configured with apply_workspace and explicit_commit policy
    And the approval gate has been granted by the user

  Scenario: git commands blocked when user message lacks explicit phrase
    Given the user message does NOT contain "commit this" or equivalent git authorization phrase
    When Sniper completes execution of the implementation tasks
    Then Strategist does NOT run git add
    And Strategist does NOT run git commit
    And Strategist does NOT run git push
    And Strategist surfaces a message indicating git authorization is required
    And Strategist instructs the user to say "commit this" to proceed

  Scenario: git commands allowed when user message contains explicit phrase
    Given the user message contains "commit this"
    When Sniper completes execution of the implementation tasks
    Then Strategist is permitted to run git add and git commit
    And the commit is scoped only to the approved workspace changes

  Scenario: gate approval alone does not constitute git authorization
    Given the user typed "yes" at the approval gate
    But the user message does NOT contain "commit this" or equivalent
    When Sniper attempts to run a git state-modifying command
    Then the command is blocked
    And Strategist surfaces: git authorization required
