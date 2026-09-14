---
name: brainstorming
description: Configured discovery-slot metadata for Strategist (risk_score only). Discovery invocation no longer resolves to this weapon for any subtype — all discovery always resolves to internal_skills/ranger (native role); see contracts/narrative/00-routing.md § Discovery Weapon Resolution by Subtype.
metadata:
  version: "1.0.0"
  author: strategist-project
---

# brainstorming — metadata mirror

This package is a thin, Strategist-owned metadata mirror, not a vendored
copy of any third party's actual prompt content (ADR-0029 DEC-001:
adapter-first, no blind vendoring). It exists only to carry
`risk_score`/`canonical_role` compatibility metadata for the discovery slot
in Strategist's role/provider catalog.

A live invocation of the real upstream skill's own instructions previously
revealed structural incompatibilities with Ranger's autonomous single-shot
contract (see `.analysis/refined/20260728-ranger-drift-eval/`), so this
mirror makes no discovery-subtype capability claim, and discovery always
resolves to `internal_skills/ranger` (native role) regardless of this
entry's presence in the catalog.
