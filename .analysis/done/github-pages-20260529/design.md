# Design: GitHub Pages Publishing Pipeline
**Mission ID:** github-pages-20260529

---

## Workflow Structure

```
.github/workflows/pages.yml
```

### Triggers
| Trigger | Purpose |
|---------|---------|
| `push` to `main`, `paths: pages/**` | Auto-deploy on content change |
| `workflow_dispatch` | Manual redeploy on demand |

Filtering by `pages/**` avoids unnecessary deploys when unrelated files (e.g., `readme.md`, `bootstrap.sh`) are pushed.

### Permissions
```yaml
permissions:
  contents: read
  pages: write
  id-token: write
```
`pages: write` + `id-token: write` are required by GitHub's OIDC-based Pages deployment mechanism. `contents: read` is principle of least privilege.

### Concurrency
```yaml
concurrency:
  group: "pages"
  cancel-in-progress: false
```
`cancel-in-progress: false` is intentional — GitHub recommends NOT cancelling in-progress Pages deploys to avoid leaving the site in a broken state.

### Job: deploy
```
environment:
  name: github-pages
  url: ${{ steps.deployment.outputs.page_url }}
runs-on: ubuntu-latest
```

The `environment` block links the job to the `github-pages` environment, which enables the deployment URL in the GitHub UI.

### Steps

| Step | Action | Purpose |
|------|--------|---------|
| Checkout | `actions/checkout@<SHA>` | Fetch repo content |
| Setup Pages | `actions/configure-pages@<SHA>` | Validate Pages config |
| Upload artifact | `actions/upload-pages-artifact@<SHA>` | Bundle `pages/` dir |
| Deploy | `actions/deploy-pages@<SHA>` | Push to GitHub Pages CDN |

### SHA Pinning (Repo Convention)
All actions pinned to full commit SHA with `# vX.Y` comment.
Reuse `actions/checkout` SHA from `release.yml`: `34e114876b0b11c390a56381ad16ebd13914f8d5`

SHAs for new actions — to be resolved at execution time:
- `actions/configure-pages@v5`
- `actions/upload-pages-artifact@v3`
- `actions/deploy-pages@v4`

### Upload Path
```yaml
- uses: actions/upload-pages-artifact@<SHA>
  with:
    path: 'pages'
```
The `path` is `pages` (the directory), not `pages/index.html`. This uploads the full directory as the artifact, which GitHub Pages serves as the site root.

---

## Affected Files

| File | Change |
|------|--------|
| `.github/workflows/pages.yml` | **Create** |

No other files are modified.

---

## Decision Log

| Decision | Rationale |
|----------|-----------|
| Trigger on `paths: pages/**` only | Avoid redeploys on unrelated commits |
| Include `workflow_dispatch` | Allows manual redeploy without a dummy commit |
| `cancel-in-progress: false` | GitHub Pages recommendation — don't leave site broken |
| No build step | `index.html` is self-contained static |
| SHA pinning | Repo convention established in `release.yml` |
