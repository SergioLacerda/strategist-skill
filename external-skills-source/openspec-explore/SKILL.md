---
name: openspec-explore
description: Refinement provider for Strategist. Consumes discovery output, structures the implementation approach, and writes analysis artifacts within the Strategist write_analysis boundary.
metadata:
  version: "1.0.0"
  author: strategist-project
---

# openspec-explore — metadata mirror

This package is a thin, Strategist-owned metadata mirror, not a vendored
copy of any third party's actual prompt content (ADR-0029 DEC-001:
adapter-first, no blind vendoring). It carries `risk_score`/`canonical_role`
compatibility metadata so the refinement slot can offer it as an alternative
to native `archivist`.
