# Strategist Philosophy and Engineering Principles

**Status:** Accepted
**Last Updated:** 2026-09-18

This document states the reasons behind the Strategist contracts. The numbered
runtime contracts and the Go implementation remain authoritative for behavior;
this page explains the principles that changes must preserve.

## Fixed pipeline, replaceable weapons

Strategist owns one auditable mission pipeline:

```text
Scout → Ranger → Archivist → Approval Gate → Sniper → Learning
```

The roles and checkpoints are fixed. A configured skill package is a replaceable
weapon used inside a role boundary; it cannot redefine the pipeline, introduce a
second state protocol, or silently replace another role/provider. Discovery
subtypes are routed by Scout and normalized by Ranger. Static provider readiness
is never presented as proof that a provider was invoked.

## Evidence before authority

Different records answer different questions and must not be collapsed:

- `.strategist/plugins.lock` is the durable authority for Custom bindings.
- Certified embedded catalog data is the authority for Ranked bindings.
- Package provenance, digests, and compatibility records are evidence, not
  authorization.
- A successful runtime probe is required for live readiness.

Missing, stale, unsupported, failed, or contradictory evidence fails closed.
The system does not repair a binding, choose a native fallback, or infer live
behavior from a catalog entry.

## One mission transition facade

`MissionEngine` is the mission-level transition facade. Callers submit the
mission event vocabulary through it; validation, replay, restore, gate ordering,
and retry boundaries remain consistent across live and restored missions. The
lower-level transition table is an implementation detail, not a second public
authority.

## Approval before materialization

Analysis and refinement are separate from execution. The Approval Gate is a
human decision point, and local execution policy does not replace it. Sniper's
current contract materializes approved documentation and handoff artifacts; it
does not edit source code, tests, hooks, locks, generated defaults, or release
state. Code implementation requires an execution path whose contract explicitly
permits that mutation.

## Bounded runtime and hermetic validation

Provider-owned runtimes use explicit roots, minimal environments, and scoped
writable directories. Tests isolate HOME, XDG state, provider scratch, and Go
caches. A fixture can prove containment and contract behavior, but it cannot
certify an external live provider. Release and CI claims must name the evidence
layer they actually verify.

## Change discipline

Before changing a contract, locate its source writer, generated/runtime mirror,
tests, and downstream consumers. Prefer a narrow behavior-preserving change,
add a regression test at the violated boundary, regenerate derived artifacts,
and run the smallest relevant hermetic gate before broader validation. Do not
close an analysis package or call a release complete from checkboxes alone.
