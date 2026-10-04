# domain-aggregation-evaluation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Document a bounded `internal/system/<package>` namespace without merging or physically relocating the six independent system packages.

**Architecture:** Preserve package ownership and sibling isolation. Record the namespace as a future import-path option, while keeping the current paths and runtime behavior unchanged until a separately approved migration.

**Tech Stack:** Markdown architecture documentation, repository documentation gates, Go architecture tests.

---

### Task 1: Publish the namespace contract

**Files:**
- Modify: `docs/architecture/overview.md`
- Modify: `docs/architecture/strategist-concepts.md`

Document membership, exclusions, detector ownership, orchestration/adapter ownership, and the no-sibling-import rule.

### Task 2: Record the migration decision

**Files:**
- Modify: `.analysis/refined/20261003-domain-aggregation-evaluation/tasks.md`

Record that physical relocation is rejected for this evaluation and remains a separately approved migration; do not edit Go import paths.

### Task 3: Verify documentation and architecture boundaries

**Files:**
- Test: `internal/domain` architecture suite and repository documentation gates.

Run docs governance/link/consistency checks, focused architecture tests, and `git diff --check`; preserve unrelated dirty worktree changes.
