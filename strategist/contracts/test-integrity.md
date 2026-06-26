# TEST INTEGRITY
id: test-integrity
severity: high

Tests are the executable specification of expected behavior.
Code adapts to the test — never the other way around.

## Forbidden

- Weakening an assertion to make the test pass
  (`assert.Equal` → `assert.NotNil` without justification)
- Removing a test case without documenting the reason
- Updating a golden file or snapshot without an explained diff in the commit
- Writing a test that does not fail when the behavior it tests breaks
- Test dependent on execution order
- Arbitrary `time.Sleep` in tests — use `testify/assert.Eventually` or channels
- Mock that makes the test insensitive to real behavior changes
- Test that only validates internal implementation details (not observable behavior)

## Required When Modifying Tests

Any modification to a `*_test.go` file that weakens coverage or removes test cases
requires a comment explaining the reason in the same commit.

## Enforcement

`testifylint` in golangci-lint detects misuse of testify assertions.
Coverage gate (90%) detects coverage regression.
response-critic evaluates test integrity after each Sniper run.
