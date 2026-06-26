# NO HACK WITHOUT EVIDENCE
id: no-hack-without-evidence
severity: high

Hacks are forbidden by default. Any exception requires all 5 mandatory items:

1. **diagnosis** — what was investigated before choosing the hack
2. **evidence** — why the correct approach does not work in this case
3. **explicit trade-off** — what is lost by using the hack
4. **temporary marker** — `// HACK: <reason>` with issue or task referenced in the same comment
5. **follow-up task** — issue registered to resolve the root cause

> A hack may exist as an exception. Never invisible.

## Behaviors Forbidden Without Evidence

- Suppressing errors without diagnosis
- Weakening tests to make code pass
- `recover()` or generic silent error handling
- `_ = err` without a comment explaining why the error is safe to ignore
- Altering public contracts without escalation
- Adding abstractions without evidence of need
- Bypassing architectural layers
- Disabling linters, tests, or checks
- Introducing mutable global state
- Adding dependencies without approval

## Enforcement

This mandate is validated by Archivist during refinement and by response-critic
after materialization. Detected violations are reported as blockers in the learning loop.
