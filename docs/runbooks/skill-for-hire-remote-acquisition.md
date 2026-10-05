# Runbook: Importing a Skill-for-Hire Package

**Status:** Accepted
**Last Updated:** 2026-10-05

This runbook describes the controlled process for acquiring a Skill-for-Hire
package from the approved upstream repository and importing it into
Strategist's embedded skill catalog.

The process has two separate boundaries:

1. `strategist plugins acquire-remote` fetches and validates one immutable Git
   commit into an isolated quarantine.
2. A human reviews and promotes the package into `external-skills-source/`;
   `make generate-embedded` then compiles and registers it.

There is no single Make target that downloads and imports a remote skill. The
existing `make skill-for-hire-pin-check` target is read-only: it checks the
recorded upstream pin and does not acquire or import a package.

## Required inputs

Before starting, obtain:

- the approved repository URI, for example
  `https://github.com/SergioLacerda/skill-for-hire`;
- the exact full 40-character lowercase commit SHA to acquire;
- the package identifier used for the quarantine and promotion path;
- the package's expected version and Role/slot metadata for the later review.

The commit SHA is the immutable acquisition reference. A semantic version such
as `batata@1.0.0` is not resolved by this command and must not replace the
commit pin. Resolve the version to its approved commit first, then pass that
commit to the command. The `--package-id` value is a package/path label; it is
not a request to resolve the latest version.

## 1. Confirm the upstream pin

Record the exact commit selected for review. If the upstream project exposes a
version or tag, use it only to identify the commit; the acquisition command
must still receive the full commit SHA.

For example, after resolving the approved `batata@1.0.0` release to
`<full-commit-sha>`:

```bash
COMMIT=<full-commit-sha>
REPOSITORY=https://github.com/SergioLacerda/skill-for-hire
PACKAGE_ID=batata
```

Do not use a branch, a tag, `latest`, or a short SHA in the acquisition
command.

## 2. Acquire into quarantine

Run the command from the Strategist checkout:

```bash
go run ./cmd/strategist plugins acquire-remote \
  --source git \
  --repository "$REPOSITORY" \
  --commit "$COMMIT" \
  --package-id "$PACKAGE_ID" \
  --allow-network
```

The equivalent installed-binary form is:

```bash
strategist plugins acquire-remote \
  --source git \
  --repository "$REPOSITORY" \
  --commit "$COMMIT" \
  --package-id "$PACKAGE_ID" \
  --allow-network
```

`--allow-network` is mandatory and makes the network boundary explicit. The
command creates a temporary quarantine outside the workspace, verifies that
the resolved commit matches the requested SHA, validates the package shape,
and writes `acquisition.json` followed by `READY`.

The command does not write any of the following:

- `external-skills-source/`;
- `.strategist/`;
- the embedded catalog;
- the embedded payload mirror;
- the source lockfile.

Save the `quarantined <path>` reported by the command as
`QUARANTINE_PATH` for the review step.

## 3. Review provenance and package contents

Inspect the acquisition evidence and package files:

```bash
QUARANTINE_PATH=<reported-quarantine-path>
cat "$QUARANTINE_PATH/acquisition.json"
test -f "$QUARANTINE_PATH/READY"
test -s "$QUARANTINE_PATH/SKILL.md"
test -s "$QUARANTINE_PATH/strategist.yaml"
```

Review at least:

- repository URI and requested/resolved commit;
- package and version identity;
- deterministic tree digest;
- `SKILL.md` instructions and referenced files;
- `strategist.yaml` Role, supported slot, runtime, risk, provenance, and
  upstream content digest fields;
- license and trust evidence required by the active ingestion policy.

Do not promote a package if `READY` is missing, the manifest does not match the
requested commit, the package shape is invalid, or the trust review is
incomplete.

## 4. Manually promote the reviewed package

Only after approval, create the package directory under the repository's raw
input boundary and copy the reviewed package contents into it:

```bash
mkdir -p "external-skills-source/$PACKAGE_ID"
cp -a "$QUARANTINE_PATH/SKILL.md" \
  "$QUARANTINE_PATH/strategist.yaml" \
  "external-skills-source/$PACKAGE_ID/"
```

Copy any reviewed package subdirectories required by `SKILL.md` as well. Do
not copy `acquisition.json` or `READY` as package payload files unless the
package contract explicitly requires them; they are acquisition evidence, not
Strategist package metadata.

The promoted package must contain the required root files:

```text
external-skills-source/<package-id>/
├── SKILL.md
└── strategist.yaml
```

The sidecar remains mandatory. Its declared version, Roles, supported slots,
runtime, risk, provenance and trust metadata must be reviewed before ingestion.

## 5. Check generated-artifact drift

Run the read-only check before changing generated defaults:

```bash
make generate-embedded-check
```

This confirms whether the current raw input set is consistent with the
catalog, payload mirrors, registry and lockfile. It does not promote the new
package or write generated files.

## 6. Generate and register the embedded package

After the promotion has been separately authorized, run:

```bash
make generate-embedded
```

This invokes `strategist plugins prepare-embedded` over
`external-skills-source/` and materializes the catalog, versioned skill
payloads, registry data and provenance lock. Review the resulting diff and run
the repository's normal tests and checks before delivery.

## 7. Verify the result

At minimum, confirm:

```bash
make generate-embedded-check
git diff --check
git status --short
```

The final check should report no generated drift. Confirm that the diff contains
only the reviewed package and its expected generated artifacts. Acquisition
itself must never be presented as proof of live provider invocation or Role
capability isolation; those are separate runtime validations.

## Failure and rollback

Acquisition failures clean up the exact temporary transaction and do not leave
a successful `READY` marker. If review rejects the package, delete or discard
the quarantine transaction and do not copy it into
`external-skills-source/`. If a package was promoted but must be withdrawn,
remove that specific raw package and regenerate the embedded artifacts through
the normal reviewed workflow; do not edit the generated catalog or lockfile by
hand.

## Related references

- [Remote acquisition runbook contract](skill-for-hire-remote-acquisition.runbook.yaml)
- [ADR-0060 — Weapon Acquisition Flow and Registry-Derived Roster](../adr/0060-weapon-acquisition-flow-and-registry-derived-roster.md)
- [Makefile and script reference](../makefile-scripts.md)

