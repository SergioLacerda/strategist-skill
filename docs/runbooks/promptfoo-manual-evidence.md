# Promptfoo manual evidence

Promptfoo evaluation is an optional, operator-run evidence collection path. It
is not a CI prerequisite and it does not certify a live conformance row.

## Preconditions

- Obtain authorization to access the selected compatible endpoint and to retain
  the resulting report.
- Set `PROMPTFOO_LM_STUDIO_URL` to the endpoint selected by the operator. Do
  not place credentials in the command line, Promptfoo configuration committed
  to the repository, or report metadata.
- Choose a report identity before running: record the UTC collection time,
  repository revision, Promptfoo configuration path, endpoint host class (not a
  credential-bearing URL), and the operator or authorized runner identity.

## Procedure

1. Perform endpoint preflight without treating a failure as a test result:

   ```sh
   curl -sf -m 3 "$PROMPTFOO_LM_STUDIO_URL/models" >/dev/null
   ```

   If it fails, stop and record `unavailable`; do not retry through an
   undeclared provider or silently substitute a local endpoint.
2. Run the guarded manual target only after the endpoint is reachable:

   ```sh
   make eval-promptfoo PROMPTFOO_LM_STUDIO_URL="$PROMPTFOO_LM_STUDIO_URL"
   ```
3. Store the report outside the repository unless an approved artifact store
   says otherwise. Before publication, redact credentials, authorization
   headers, query tokens, private endpoint paths, provider payloads, and any
   prompt content that is not approved for retention.
4. Attach the report identity and the selected retention period to the stored
   artifact. A missing retention decision means the result remains local and
   must not be published as project evidence.

## Evidence boundary

A reachable manual result is advisory evidence about the selected endpoint and
configuration at one point in time. It neither changes structural CI nor
certifies the `strategist-live-evidence/v1` contract. Automated live
conformance requires the separately authorized provider, runner, secret
boundary, timeout, teardown, report location, and retention policy tracked in
`.analysis/pending/20260918-cicd-coverage-live-evidence.md`.

## References

- `make/go.mk` (`eval-promptfoo`)
- `docs/testing/client-conformance.md`
- `docs/adr/0020-promptfoo-ci-adapter.md`
