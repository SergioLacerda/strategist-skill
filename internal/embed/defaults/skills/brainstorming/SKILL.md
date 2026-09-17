---
name: brainstorming
description: Flexible brainstorming weapon for Ranger discovery; Ranger owns normalization into the canonical pending artifact and handoff.
metadata:
  version: "1.0.0"
  author: strategist-project
---

# brainstorming — metadata mirror

This package carries the flexible brainstorming behavior for Ranger discovery.
It does not own Strategist's pipeline state or handoff. Ranger treats its
result as untrusted input and normalizes it through the fixed discovery
checkpoint before emitting the canonical pending artifact and handoff.
It never writes a planning document to `docs/plans/`; Ranger emits the
normalized discovery artifact only at `<base_path>/pending/<mission_id>-analysis.md`.
