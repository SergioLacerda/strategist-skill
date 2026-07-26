# Runbook — Deep Analysis Pass 0: Inventory & Metrics

Mechanical evidence-gathering. **Tables and facts only — no judgments.** Every
observation that smells like a problem is recorded as a routed flag (`F<n> → Pass N`),
not evaluated here. See `deep-analysis-workflow.md` for shared invariants.

## Trigger

Start of a deep-analysis workflow (`deep-analysis-workflow.md`), or any time a fresh
mechanical baseline of the corpus is needed. Inputs: frozen baseline (branch + commit)
and the corpus trees — for this repo: `strategist/` (authoring),
`internal/embed/defaults/` (embedded generation source), `.strategist/` (runtime
instance).

## Steps

### 1. Tree parity metrics

```sh
diff -rq <authoring-tree> <runtime-tree>            # count: only-in-A / only-in-B / differ
diff -rq <embed-tree> <runtime-tree>
find <tree> -type f | wc -l                          # per-tree file counts
```

Record counts and the divergent file list. **Caution (learned 2026-07-26):** runtime-only
divergences may be legitimate instance configuration (installed sources, lock files) —
record them, but do not label them drift without checking their nature; issue an erratum
if a later pass reclassifies them.

### 2. Compile coverage

```sh
grep -rl 'generated_by' <runtime-tree> --include='*.md' --include='*.yaml'
```

Table: which artifacts are compiler-generated vs hand-maintained, per tree.

### 3. Error/drift token taxonomy

```sh
# every token documented anywhere
grep -rhoE '(error|drift)=[a-z_]+' <docs-trees> | sort | uniq -c | sort -rn
# for each token: presence in code and tests
for t in <token-list>; do
  echo "$t go=$(grep -rl "$t" internal/ cmd/ | wc -l) tests=$(grep -rl "$t" tests/ | wc -l)"
done
```

Output: table token × kind × doc-occurrences × code-files × test-files. Zero-test rows
and doc-occurrence counts ≥ ~10 (copy-paste blocks) are routed flags. Also collect
drift-pattern IDs from the identity file and diff against any "quick reference" list.

### 4. Restated-rule map

Pick signature phrases for load-bearing rules (gate names, wire-field names, "Never"/"MUST
NOT" blocks) and count occurrences × files across entry docs + contracts:

```sh
grep -rc "<signature phrase>" <entry-docs> <contracts> | awk -F: '{s+=$2} END{print s}'
```

Any rule restated in ≥2 files is a dedup candidate (route to Pass 1) and a drift risk
(route to Pass 2).

### 5. Manifest / registry coverage

For each index or registry file (e.g. `contracts/index.yaml`): verify every artifact it
should govern is referenced, and every reference resolves. List unreferenced artifacts
and dangling references. Include secondary registries (internal skill dirs: which have
SKILL.md vs manifest-only).

## Decision Point

Pass 0 is done when `<base_path>/pending/<slug>/00-inventory.md` exists containing the
five tables above + a **routed flags** table (`F<n> | observation | evidence | route to
Pass N`). Passes 1–4 may then start, in any order.

## Stop Conditions

- Do not fix anything, even trivial typos — record and route.
- Do not run state-changing CLI commands to "check" behavior; read-only inspection only.
- Do not evaluate or judge — a Pass 0 artifact containing opinions is malformed.

## Reference

- `deep-analysis-workflow.md` (master); consumers: passes 1, 2, 4.
- Worked example: `.analysis/pending/strategist-deep-analysis/00-inventory.md` (2026-07-26).
