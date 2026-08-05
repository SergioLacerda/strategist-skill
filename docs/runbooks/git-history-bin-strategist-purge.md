# Runbook: Purge `bin/strategist` Blobs from Git History

`bin/strategist` is correctly gitignored today (`/bin/` in `.gitignore`), but
commits made before that ignore rule existed left compiled binaries in git
history. This is historical bloat only — not an active leak — and low
priority (a shallow clone won't grow further), but is documented here so the
exact procedure is ready when/if someone decides to run it.

Source mission: `.analysis/refined/20260804-critique-followup-corrections/`.

## Measured Severity (as of 2026-08-04)

| Metric | Value |
|---|---|
| Distinct `bin/strategist` blob versions in history | 78 |
| Total raw blob content (`git cat-file -s` summed, deduplicated) | 1.3 GB |
| Current `.git` directory size | 297 MB |

Re-measure before acting — these numbers only grow if more binaries get
committed by accident in the meantime:

```sh
git rev-list --objects --all | grep -E "bin/strategist$" | awk '{print $1}' | sort -u | wc -l
du -sh .git
```

## ⚠️ Before You Run Anything

This procedure **rewrites every commit hash** from the first
`bin/strategist` commit forward, and requires a **force-push**. Consequences:

- Every existing local clone, fork, and open pull request becomes based on
  now-orphaned history. Collaborators must re-clone or hard-reset to the
  new history — `git pull` will not work cleanly for them.
- GitHub PR discussion threads tied to rewritten commits may lose their
  diff context.
- This is irreversible on the remote once force-pushed (mitigated by the
  rollback step below, which only protects *your own* copy).

**Do not run this without coordinating with every active collaborator
first**, and confirm no one has in-flight work on a branch based on the
commits being rewritten.

## Rollback Safety Net

Before touching anything, make a full mirror backup you can restore from if
something goes wrong:

```sh
git clone --mirror <repo-url> strategist-skill-backup.git
```

Keep this until you've confirmed the rewritten history is correct and every
collaborator has successfully re-cloned.

## Procedure

1. Install `git-filter-repo` (not bundled with git):
   ```sh
   pip install git-filter-repo   # or: brew install git-filter-repo
   ```
2. In a **fresh clone** (not your working copy — `filter-repo` refuses to
   run in a repo with unpushed changes or unusual remote configuration by
   default):
   ```sh
   git clone <repo-url> strategist-skill-purge
   cd strategist-skill-purge
   ```
3. Remove every `bin/strategist` blob from all history:
   ```sh
   git filter-repo --path bin/strategist --invert-paths
   ```
4. Verify the objects are gone:
   ```sh
   git rev-list --objects --all | grep -E "bin/strategist$"
   # should print nothing
   du -sh .git
   # should be materially smaller
   ```
5. `filter-repo` removes the `origin` remote as a safety measure — re-add it:
   ```sh
   git remote add origin <repo-url>
   ```
6. Force-push the rewritten history (only after collaborator coordination):
   ```sh
   git push origin --force --all
   git push origin --force --tags
   ```
7. Notify every collaborator to re-clone, or to hard-reset their local
   branches to the new remote history (`git fetch && git reset --hard
   origin/<branch>`), and to close/reopen any PR whose base commits were
   rewritten.

## Not Covered by This Runbook

- Purging any other accidentally-committed large file — repeat step 3 with
  the appropriate `--path`/`--invert-paths` arguments for that path.
- BFG Repo-Cleaner as an alternative to `git-filter-repo` — equivalent
  outcome, different tool; not detailed here since `git-filter-repo` is the
  tool GitHub's own documentation currently recommends.
