# Runbook: TREASURE-CHEST-TMP boundary and later cutover

## Purpose

Operate the Sprint 1 boundary in which the root Treasure Chest implementation
is exposed as the temporary `treasure-chest-tmp` Weapon. This runbook is
diagnostic and operational guidance; it is not an authorization to change
source, catalogs, locks, or provider data.

## Required production path

```text
Ranger / Archivist -> JEWELER -> ROSTER treasure-chest-tmp -> explicit executable
```

The old `strategist treasure-chest` namespace and local runbook selector are not
production paths. There is no fallback, compatibility alias, PATH discovery, or
second accepted identity.

## Readiness checks

1. Confirm the active ROSTER has `treasure-chest-tmp@0.1.0` and no active
   `treasure-chest` package.
2. Confirm the host binding supplies an executable explicitly.
3. Run the executable probe through JEWELER. Verify provider identity,
   `SchemaVersion: 1`, `Healthy: true`, and `FallbackState: none`.
4. Query only through JEWELER and verify requester, mission, correlation,
   result limit, source path, freshness, and limitations.

Example provider smoke check:

```bash
make treasure-chest-tmp
./bin/treasure-chest-tmp status --root .strategist --json
./bin/treasure-chest-tmp search --root .strategist --json \
  --token-budget 3 --intent runbook.select --concept telemetry
```

An unhealthy status, failed executable, identity mismatch, malformed response,
unsupported item kind, or non-`none` fallback state is a hard failure. Report
the JEWELER readiness or `role_invocation_failed` diagnostic; do not answer
from local data.

## ROSTER regeneration

The canonical package source is `external-skills-source/treasure-chest-tmp/`.
After an intentional metadata change, regenerate and check drift:

```bash
GOCACHE=/tmp/strategist-go-cache make generate-embedded
GOCACHE=/tmp/strategist-go-cache make generate-embedded-check
```

The parked future package under `external-skills-source-staging/` must not be
added to the active source directory during Sprint 1.

## Later sprints

- Sprint 2: admit the Skill-for-Hire Weapon and add an explicit shadow
  comparison. It is not a fallback for the temporary Weapon.
- Sprint 3: remove `treasure-chest-tmp`, its executable, root implementation,
  tests, and remaining internal references after an approved parity gate.

Do not perform either later sprint as part of this procedure.
