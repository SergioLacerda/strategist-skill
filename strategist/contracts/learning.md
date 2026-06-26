# Strategist — Learning Contract

## 9. Learning Phase (non-blocking)
> **Contracts:** `.strategist/contracts/learning-curator.yaml`, `.strategist/contracts/learning-buffer.yaml`

After mission completes (either `documentation_applied`, `analysis_delivered`, `revision_requested`, or `rejected`):

Invoke `response-critic` with the slot outputs and the task-type rubric.

Invoke `learning-curator` with:
- Critic evaluation
- Mission result
- `task_type`

Learning curator MUST present a checkpoint to the user before writing anything.
If the learning phase fails or times out: log the failure, return the mission result unchanged.
The mission result is NEVER blocked or modified by learning phase failure.

**LearningBuffer write procedure:**

After learning-curator completes (or if it fails — still append outcome):

1. Append the mission outcome JSON line to:
   `.strategist/memory/outcomes.tmp`

   - Minimum required fields remain: `mission_id`, `status`, `timestamp`.
   - Preferred structured shape is defined in `.strategist/schemas/outcome-entry.schema.yaml`.
   - Producers SHOULD populate `task_type`, `outcome_summary`, `tags`, `lessons`, and `files_touched`
     when that data is available without additional inference work.

2. The buffer is flushed at the START of the next mission (§0 Pre-Bootstrap), not here.
   Do not flush at end of mission — this is intentional for crash safety.

3. Historical retrieval remains layered:
   - baseline: structured outcomes in `outcomes.jsonl`
   - optional: tag-based retrieval and lexical search over the JSONL corpus
   - future capability: semantic retrieval only if explicitly enabled and healthy

4. Semantic retrieval is never part of the critical path.
   Any future semantic index MUST be local, rebuildable, and optional.
   If unavailable or unhealthy, Strategist MUST fall back to tags or lexical retrieval.

**Manual flush (if needed):**
```sh
cat .strategist/memory/outcomes.tmp >> .strategist/memory/outcomes.jsonl
: > .strategist/memory/outcomes.tmp
```

**Rollback:** Delete `.strategist/.compiled/` to revert to YAML-only path. No code change needed.
