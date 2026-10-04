# Runbook: Pending or Expired Embedded Invocation Requests

## Symptom

A `strategist mission invoke` request (`inv_` followed by 32 hex characters) is still
`pending` although its Weapon was already run, or it sits beside a newer request for the same
mission, role and slot, or a command reports `invocation_request_expired` or
`invocation_replay`. No phase is blocked.

It is typically an orphan: the first request of a `mission invoke` call whose output was truncated
or discarded, left unfinished when the command was run again.

## Root Cause

Each `strategist mission invoke --json` issues a new request with a one-hour time to live. It
is not idempotent: if the output is truncated (`head`, `cut`, a display limit) or discarded
and the command is run again, the first request is never completed.

The store handles this by design. A request past `expires_at` can no longer be completed
(`invocation_request_expired`), and a consumed one is rejected as `invocation_replay`.
Publication target leases are taken only when a request is completed, and a stale one is
reclaimed after ten minutes once its owner is gone. A request that was not completed is removed by a best-effort prune when a later
request is issued, once its `expires_at` is more than 24 hours in the past; completed records
are removed 24 hours after completion. The envelope is about 17 KB and contains a `nonce`.

## Resolution Steps

1. List the requests without printing their payload or nonce:

   ```sh
   strategist mission requests --mission-id <id>
   ```

   The `expired` column tells you whether a `pending` request can still be completed. Omit
   `--mission-id` to list every mission. On a binary that predates this command, read the same
   fields with `jq` from `.strategist/missions/invocations/inv_*.json` and never print the
   `payload`, `input` or `nonce`.
2. If the request is `pending` and `expires_at` is in the past, do nothing. It blocks no
   later request and is removed automatically. Do not delete or edit it by hand.
3. If it is not expired, another session may own it. Requests from other missions created
   minutes ago are normal in parallel use. Never complete or discard a request that you did
   not issue.
4. If you still need the Weapon to run, issue a fresh request and write its envelope to a
   file, then read fields from the file:

   ```sh
   strategist mission invoke --mission-id <id> --role <role> --slot <slot> --json 2>/dev/null | grep '^{' > <scratch>/envelope.json
   jq -r '.request_id' <scratch>/envelope.json
   ```

   Complete only that request id with `strategist mission complete`.
5. Do not use `restarting-orphaned-strategist-missions` for this case: it covers an analysis
   artifact stuck at `mission_status: ranger_pending`, not a request record.
6. Do not print the `nonce` or paste an envelope into notes or tickets.

## Prevention

- Write the `mission invoke` output to a file on the first call and read it with `jq`; never
  pipe it through `head`, `cut` or similar.
- Run `mission invoke` once per phase and keep its `request_id`.

## Reference

- `internal/mission/invocation_retention.go` (retention and prune)
- `internal/mission/invocation_store.go` (expiry and replay)
- `docs/runbooks/restarting-orphaned-strategist-missions.md` (a different case)
