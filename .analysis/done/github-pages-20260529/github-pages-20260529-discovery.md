# Discovery: GitHub Pages Publishing Pipeline
**Mission ID:** github-pages-20260529
**Date:** 2026-05-29
**Status:** pending

---

## User Intent

Set up a GitHub Actions workflow to publish the `pages/` directory to GitHub Pages automatically.

- Trigger: push to `pages/**` on main branch
- Source dir: `pages/` (static, no build step required)
- URL: https://sergiolacerda.github.io/strategist-skill/
- `pages/index.html` is self-contained — no bundler or build needed
- `pages/docs/banner.png` is the image referenced by `readme.md`

---

## Existing Context

| Item | State |
|------|-------|
| `.github/workflows/release.yml` | Exists — tag-based release workflow |
| `.github/workflows/pages.yml` | **Missing — must be created** |
| `pages/` directory | Exists with `index.html`, `banner.html`, `docs/`, `dungeon.css`, etc. |
| GitHub Pages source | Must be set to "GitHub Actions" in repo Settings → Pages |
| Action SHA pinning | Repo convention enforced (see release.yml — all actions pinned to commit SHA) |

---

## Findings

### What's Needed
1. A new workflow file: `.github/workflows/pages.yml`
2. Workflow must:
   - Trigger on `push` to `main` (filter: `paths: ['pages/**']`)
   - Include `workflow_dispatch` for manual redeploys
   - Use `actions/configure-pages`, `actions/upload-pages-artifact`, `actions/deploy-pages`
   - Set permissions: `pages: write`, `id-token: write`
   - Set concurrency group `"pages"` to prevent overlapping deploys
   - Upload artifact from path `pages/` (not repo root)

### No Build Step
The index.html is self-contained — no npm install, no bundler, no compile step. Upload directly.

### Action SHA Pinning (Repo Convention)
Must pin all third-party actions to their full commit SHA, with version comment.
Actions required:
- `actions/checkout@v4`
- `actions/configure-pages@v5`
- `actions/upload-pages-artifact@v3`
- `actions/deploy-pages@v4`

`actions/checkout` SHA already known from release.yml: `34e114876b0b11c390a56381ad16ebd13914f8d5`

### Repo Settings (manual step — not automatable via workflow)
User must go to:
> Settings → Pages → Source = **GitHub Actions**

This is a one-time manual step; the workflow handles all subsequent deploys.

---

## Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Pages source not set to "GitHub Actions" | Blocks deploy | Document as manual pre-step |
| Overlapping deploys on rapid pushes | Low | `concurrency` group cancels in-progress |
| SHA pins go stale | Low | Comment with version tag for future pinning updates |
| Large binary in pages/docs/ | Low | `docs/banner.png` is small; no issue |

---

## Out of Scope
- Building or bundling the pages/ content (already static)
- Setting up a custom domain
- Modifying pages/ content
