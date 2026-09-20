# ADR-0048 — Skill Package Evidence and Authority Boundaries

**Status:** Accepted
**Date:** 2026-09-17

## Context

External skill packages expose publisher metadata, host adapter metadata,
workspace bindings, and runtime probe results. These facts have different
owners. Combining them into one apparently authoritative manifest can make a
static catalog claim live readiness or silently repair a user binding.

## Decision

The Skill Package Contract v1 is a validated projection over existing package,
adapter, catalog, and provenance records. It is evidence, not authorization.

- `external-skills-source.lock.yaml` owns deterministic ingestion provenance,
  source/revision declarations, transformation information, and source and
  normalized digests.
- `.strategist/plugins.lock` remains the durable authority for Custom slot
  bindings and must not be replaced or repaired by package metadata.
- Certified embedded catalog data remains the authority for Ranked bindings;
  any runtime mirror is optional and non-authoritative.
- Inventory/probe state owns runtime readiness. Catalog presence, capability
  declarations, or provenance verification cannot substitute for a successful
  live probe.
- Provenance states are explicit. `declared` and `unknown` are not equivalent
  to `verified`; `unsupported`, `failed`, and `blocked` remain failure states.
- The package contract accepts the current `skill-package/v1` and the
  non-destructive N-1 `skill-package/v0` window. Role-to-slot affinity is
  validated before preparation accepts a package; lifecycle-only adapters may
  remain role-less and are not inferred as mission bindings.
- The external-source lock keeps `digest`/`original_digest` as the source
  digest and computes `normalized_digest` independently over the materialized
  package and generated mirror manifest. The two digest evidence states are
  recorded separately, so a declared upstream identity is not presented as a
  verified upstream attestation.

## Consequences

Package preparation can reject contradictory identity, affinity, contract, or
digest evidence before embedding. Existing valid records remain readable, but
unverifiable or incompatible records fail closed with actionable diagnostics.
Mutable upstream references require a new reviewed digest and never silently
refresh an existing binding.

This ADR does not authorize remote/OCI distribution, Treasure Chest or Atlas
migration, or a second binding file.
