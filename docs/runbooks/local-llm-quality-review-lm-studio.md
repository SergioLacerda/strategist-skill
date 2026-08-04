# Runbook: Local LLM Quality Review via LM Studio (SQ-003)

## Purpose

Optional, manual procedure for getting a local model's subjective opinion
on a Strategist-produced artifact (an analysis.md, a refined package, an
ADR) — e.g. artifacts harvested by `strategist eval harvest` (SQ-002) into
`tests/evals/regression/`. This is a "second pair of eyes" aid for a human
operator, not an automated check.

## Non-Goals

- Not part of `make eval` or any CI pipeline.
- Not a pass/fail gate — a local model's opinion is advisory only.
- Not deterministic — do not treat two runs as comparable evidence of
  regression or improvement by themselves.

## Why Not Automated

Local LLM output is not deterministic, even at temperature zero: it can
vary across model version, quantization, runtime, chat template, seed,
hardware, and inference library. Wiring this into an automated gate would
produce flaky, unreproducible pass/fail signal. Treat every run as one
opinion from one configuration at one point in time.

## Prerequisites

- LM Studio installed and running locally, with a model loaded and its
  local server enabled.
- **Operator-verify, not assumed:** LM Studio's local server typically
  exposes an OpenAI-compatible endpoint (commonly `http://localhost:1234/v1`,
  `/v1/chat/completions`) — confirm the exact host, port, and path against
  your own LM Studio version before running the steps below; this runbook
  was not verified against a live instance.
- A harvested fixture to review, e.g. from
  `tests/evals/regression/<mission_id>/analysis.md` (see the `eval harvest`
  runbook/command, SQ-002).

## Steps

1. Start LM Studio and load a model; enable its local server (check LM
   Studio's own UI for the exact toggle and displayed port).
2. Pick the artifact to review, e.g.:
   ```sh
   cat tests/evals/regression/<mission_id>/analysis.md
   ```
3. Send it to the local model with a quality-review prompt, e.g.:
   ```sh
   curl http://localhost:1234/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{
       "model": "<your-loaded-model-name>",
       "messages": [
         {"role": "system", "content": "You are reviewing a Strategist mission analysis artifact for clarity, evidence grounding, and internal consistency. Point out anything that reads as unsupported, vague, or contradictory."},
         {"role": "user", "content": "<paste artifact content here>"}
       ]
     }'
   ```
   (Adjust the URL/port/model name to match your own LM Studio setup per
   Prerequisites above.)
4. Read the response as one opinion, not a verdict. Cross-check anything it
   flags against the artifact yourself before acting on it.
5. Do not record the model's output as project evidence (e.g. in an ADR or
   analysis artifact) without human review and explicit attribution that it
   came from an unverified, non-deterministic local-model pass.

## Future: Promptfoo-Integrated Path

`SQ-004` (Promptfoo CI adapter) is unscoped as of this runbook's writing.
Promptfoo natively supports local OpenAI-compatible servers (including LM
Studio) as configurable providers. Once `SQ-004` is scoped, this manual
procedure could become a Promptfoo test case — but that is a separate,
future mission's decision, not assumed here.

## Reference

- `.analysis/pending/20260804-lm-studio-eval-integration-disposition.md` (SQ-003)
- `.analysis/archived/20260804-eval-fake-provider-adr.md` (DEC-2/DEC-3 —
  why no Go provider interface exists for this)
- `.analysis/todo/v2/tests/tests_v2.txt` (original critique, Layer 6 /
  non-determinism warning)
