# Phase D token accounting

## Scope

Phase D token accounting applies a deterministic admission check to a declared
phase and role. Its unit is whitespace-delimited words. This is a local
estimate, not a provider tokenizer and not a provider-reported token count.

## Outcomes

- `budget_admitted`: the estimate fits the configured limit.
- `budget_overflow`: the estimate exceeds the limit. The request is rejected;
  no unbounded fallback is allowed.
- `budget_malformed`: the phase, role, or limit is invalid. The request is
  rejected.
- `observation_unavailable`: no provider usage was supplied. A fitting local
  estimate may still be admitted, but the observation remains unavailable.

## Operator tuning

Set limits according to the desired maximum number of whitespace-delimited
words for the phase and role. Keep provider-reported usage, when available,
separate from the estimate; a reported value does not alter the deterministic
admission result.

## Safety and rollback

Disable the accounting capability to return to the prior admission behavior.
Do not delete recorded estimates or provider observations, and do not relabel
an estimate as an exact measurement during rollback or review.
