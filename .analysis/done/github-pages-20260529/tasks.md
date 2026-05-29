# Tasks: GitHub Pages Publishing Pipeline
**Mission ID:** github-pages-20260529

---

## Task 1 — Resolve Action SHAs

Fetch the current pinned commit SHAs for:
- `actions/configure-pages@v5`
- `actions/upload-pages-artifact@v3`
- `actions/deploy-pages@v4`

Use `gh api` or GitHub web to resolve the SHAs from the version tags:
```bash
gh api repos/actions/configure-pages/git/refs/tags/v5 --jq '.object.sha'
gh api repos/actions/upload-pages-artifact/git/refs/tags/v3 --jq '.object.sha'
gh api repos/actions/deploy-pages/git/refs/tags/v4 --jq '.object.sha'
```
Note: if the tag points to a tag object (not a commit), follow `.object.sha` → get the tag object → use `object.sha` from that.

## Task 2 — Create `.github/workflows/pages.yml`

Create the file with the following structure (substitute real SHAs from Task 1):

```yaml
name: Deploy GitHub Pages

on:
  push:
    branches: [main]
    paths:
      - 'pages/**'
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: "pages"
  cancel-in-progress: false

jobs:
  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4

      - name: Setup Pages
        uses: actions/configure-pages@<SHA_CONFIGURE_PAGES> # v5

      - name: Upload artifact
        uses: actions/upload-pages-artifact@<SHA_UPLOAD_PAGES_ARTIFACT> # v3
        with:
          path: 'pages'

      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@<SHA_DEPLOY_PAGES> # v4
```

## Task 3 — Manual Repo Setting (instructions only, not automated)

Document for the user:
> Go to: GitHub → repo Settings → Pages → Build and deployment → Source → select **GitHub Actions**

This must be done once before the first deploy succeeds.

## Task 4 — Validate

After the workflow file is committed and pushed to `main`:
- Confirm the Actions tab shows the `Deploy GitHub Pages` workflow
- Confirm the first run succeeds and deploys to https://sergiolacerda.github.io/strategist-skill/
