# Runbook: Verifying a Breaking Dependency Upgrade

## Trigger

`npm audit fix --force` (or any dependency bump) reports a "breaking change" /
major-version jump for a direct dependency, and you need to confirm it's safe
to land before committing.

## Steps

1. Identify the target version's official migration/upgrade guide; fetch it.
2. Cross-check the guide's breaking-change list against what this specific
   codebase actually uses (features, config options, file conventions named
   in the guide) — rule out inapplicable items explicitly, don't skip this
   silently.
3. Build the CURRENT (pre-upgrade) version and copy the build output aside
   as a baseline.
4. Apply the dependency bump (`npm install <pkg>@<version> ...`).
5. Build again.
6. Diff the two build outputs — but not as raw text. HTML/JS serializers can
   legitimately change *how* they escape/format equivalent content between
   versions (e.g. entity-escaping `<`/`>` inside attribute values) without
   any functional change. Decode HTML entities (or otherwise normalize)
   before diffing, and diff per meaningful unit (per-attribute, per visible
   text node) rather than raw bytes, or the diff will show false-positive
   "regressions" that are actually just serialization-style changes.
7. Re-run the project's full verification set: build, full test suite, lint,
   and a from-scratch clean install (`npm ci`) to confirm the lockfile is
   self-consistent — not just that `npm install` happened to work locally.
8. Re-run the vulnerability scan to confirm the target advisory is actually
   resolved (not just that *a* version changed).

## Decision Point

**All checks pass** (build succeeds both before and after, decoded diff shows
no real content change, full test/lint suite passes, clean `npm ci` succeeds,
target advisory is gone from the audit):
1. Land the bump. Update the dependency's declared version range in
   `package.json` if the package manager didn't already do so.
2. Record what was verified (commands run, before/after audit counts) in the
   commit message or an accompanying report — don't rely on memory of "I
   checked it" without the concrete evidence.

**Any check fails or the decoded diff shows a real content difference:** this
is not a safe drop-in bump. Stop, investigate the specific failing check, and
do not land the upgrade until it's resolved or explicitly accepted as a known,
scoped behavior change.

## Stop Conditions

- The migration guide names a breaking change this codebase's actual usage
  triggers (a removed API, a changed default this project relies on) — stop,
  don't silently patch around it or assume it's fine.
- The decoded/normalized diff shows a real content difference (not just a
  serialization-style change) — stop, investigate before landing.
- Any of build/test/lint/audit fails post-bump — the bump isn't done, don't
  report success on a partial pass.

## Reference

- Origin: `.analysis/archived/20260728-landing-preview-and-deps-report.md` —
  the mission where this procedure was first executed (Astro 5.18.2 → 7.1.5
  in `web/landing/`, verified via before/after decoded-attribute diff).
