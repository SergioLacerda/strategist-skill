# Proposal: GitHub Pages Publishing Pipeline
**Mission ID:** github-pages-20260529
**Date:** 2026-05-29

---

## What

Create `.github/workflows/pages.yml` — a GitHub Actions workflow that automatically deploys the `pages/` directory to GitHub Pages whenever changes are pushed to `main`.

## Why

The `pages/` directory contains a self-contained static site (`index.html`) that should be publicly accessible at `https://sergiolacerda.github.io/strategist-skill/`. Currently, there is no automated publishing pipeline — the workflow must be created from scratch.

## Scope

- **In scope:** Create one new file: `.github/workflows/pages.yml`
- **Out of scope:** Modifying `pages/` content, custom domain setup, build tooling

## Manual Pre-Step (One-Time)

Before the workflow can deploy, the repo owner must enable GitHub Actions as the Pages source:
> GitHub → Settings → Pages → Source = **GitHub Actions**

This is a one-time setting change; all future deploys are fully automated.
