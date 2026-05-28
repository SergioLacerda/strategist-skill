# Curl Installer — Design Spec
**Date:** 2026-05-28  
**Status:** pending implementation  
**Topic:** Cross-platform curl installer (Linux/Mac/WSL + Windows PowerShell) + GitHub Actions release CI/CD

---

## Problem Statement

The Strategist skill currently requires cloning the repo and running `strategist/install.sh` manually. There is no one-liner install experience. Users on Windows without access to the skill repo cannot install at all.

---

## Goal

A user on any platform runs a single command and gets Strategist installed with `.strategist/` runtime and agent shims registered:

```bash
# Linux / Mac / WSL
curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.sh | bash

# Windows PowerShell
irm https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.ps1 | iex
```

---

## Architecture

Two installer scripts at the repo root + one GitHub Actions workflow:

```
strategist-skill/
├── bootstrap.sh          ← new: Linux/Mac/WSL installer
├── bootstrap.ps1         ← new: Windows PowerShell installer
└── .github/
    └── workflows/
        └── release.yml   ← new: auto-publish GitHub Release on tag push
```

`install.sh` (inside `strategist/`) is NOT modified — bootstrap scripts are thin wrappers that download and delegate to it.

---

## Section 1 — `bootstrap.sh` (Linux / Mac / WSL)

### Variables (top of file)

```bash
REPO="owner/repo"          # GitHub repo slug — update before publishing
DEFAULT_REF="main"         # fallback if no release exists
```

### Flow

1. Parse args: `--wizard`, `--target=<path>`, `--ref=<tag>` (optional version pin)
2. Resolve version:
   - If `--ref` provided: use it directly
   - Else: query GitHub API `https://api.github.com/repos/${REPO}/releases/latest`
   - If no releases exist: fall back to `DEFAULT_REF` with a notice
3. Construct download URL:
   - Release tarball: `https://github.com/${REPO}/archive/refs/tags/${VERSION}.tar.gz`
   - Fallback (branch): `https://github.com/${REPO}/archive/refs/heads/${DEFAULT_REF}.tar.gz`
4. Create temp dir: `TMPDIR=$(mktemp -d)`
5. Register `EXIT` trap: `rm -rf "$TMPDIR"`
6. Download: `curl -fsSL "$URL" -o "$TMPDIR/strategist.tar.gz"`
7. Extract with prefix strip: `tar -xzf "$TMPDIR/strategist.tar.gz" --strip-components=1 -C "$TMPDIR/extracted"`
8. Execute: `bash "$TMPDIR/extracted/strategist/install.sh"` forwarding original `$@`

### Args forwarded to install.sh

`--wizard`, `--target=<path>` are passed through. `--ref` is consumed by bootstrap and not forwarded.

---

## Section 2 — `bootstrap.ps1` (Windows PowerShell)

### Variables (top of file)

```powershell
$REPO = "owner/repo"
$DEFAULT_REF = "main"
```

### Flow

```
try {
  1. Parse $args for --wizard, --target, --ref
  2. Resolve version (Invoke-RestMethod GitHub API, fallback to DEFAULT_REF)
  3. Download zip: Invoke-WebRequest -Uri $url -OutFile "$env:TEMP\strategist-$PID.zip"
  4. Extract: Expand-Archive to "$env:TEMP\strategist-install-$PID\"
  5. Detect bash runtime:
     a. Git Bash: "C:\Program Files\Git\bin\bash.exe" (test-path check)
     b. WSL: wsl.exe (Get-Command check)
     c. None found: Write-Error with install instructions, exit 1
  6. Execute: & $BASH_EXE "extracted/strategist/install.sh" @INSTALL_ARGS
} finally {
  Remove-Item temp dirs -Recurse -Force
}
```

### Bash detection order

```
1. $env:GIT_BASH (override env var)
2. C:\Program Files\Git\bin\bash.exe
3. C:\Program Files (x86)\Git\bin\bash.exe
4. wsl.exe (run as: wsl bash <script>)
```

If none found, print:
```
Error: No bash runtime detected. Install Git for Windows (https://git-scm.com) or enable WSL.
```

### PowerShell `irm | iex` args limitation

`irm URL | iex` does not support passing args to the piped script. For wizard mode on Windows:

```powershell
# Option 1: download then run
Invoke-WebRequest -Uri $URL -OutFile bootstrap.ps1
.\bootstrap.ps1 --wizard

# Option 2: inline workaround (PowerShell 7+)
& ([scriptblock]::Create((irm $URL))) --wizard
```

Both options documented in README.

---

## Section 3 — GitHub Release Structure

GitHub automatically generates source archives when a release tag is pushed:

| Asset | URL pattern |
|-------|-------------|
| Tarball | `github.com/OWNER/REPO/archive/refs/tags/vX.Y.Z.tar.gz` |
| Zip | `github.com/OWNER/REPO/archive/refs/tags/vX.Y.Z.zip` |

**Internal structure after extraction:**
```
strategist-skill-X.Y.Z/   ← prefix stripped by --strip-components=1
├── bootstrap.sh
├── bootstrap.ps1
└── strategist/
    └── install.sh
```

The CI workflow uploads explicit named assets (`strategist-skill-X.Y.Z.tar.gz`) in addition to GitHub's auto-generated archives. This gives clean, predictable download URLs independent of repo name changes.

---

## Section 4 — Usage Commands

**Linux / Mac / WSL:**
```bash
# silent install
curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.sh | bash

# interactive wizard
curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.sh | bash -s -- --wizard

# target a specific repo
curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.sh | bash -s -- --target=/my/project

# pin to a specific version
curl -fsSL https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.sh | bash -s -- --ref=v1.0.0
```

**Windows PowerShell:**
```powershell
# silent install (pipe works for silent)
irm https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.ps1 | iex

# wizard install (download first)
Invoke-WebRequest -Uri https://raw.githubusercontent.com/OWNER/REPO/main/bootstrap.ps1 -OutFile bootstrap.ps1
.\bootstrap.ps1 --wizard

# pin version
Invoke-WebRequest ... -OutFile bootstrap.ps1
.\bootstrap.ps1 --ref=v1.0.0
```

---

## Section 5 — CI/CD: GitHub Actions Release Workflow

**File:** `.github/workflows/release.yml`

**Trigger:** `push` on tags matching `v*.*.*`

```yaml
on:
  push:
    tags: ['v*.*.*']

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4

      - name: Validate scripts
        run: |
          bash -n bootstrap.sh
          bash -n strategist/install.sh

      - name: Package release assets
        run: |
          VERSION="${GITHUB_REF_NAME#v}"
          tar -czf "strategist-skill-${VERSION}.tar.gz" \
            bootstrap.sh bootstrap.ps1 strategist/
          zip -r "strategist-skill-${VERSION}.zip" \
            bootstrap.sh bootstrap.ps1 strategist/

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            strategist-skill-*.tar.gz
            strategist-skill-*.zip
          generate_release_notes: true
```

**Developer workflow to publish:**
```bash
git tag v1.0.1
git push origin v1.0.1
# GitHub Actions creates the release automatically
```

---

## Out of Scope

- Uninstall command (documented as manual: `rm -rf ~/.claude/skills/strategist`)
- Version pinning UI in wizard
- Homebrew formula / Chocolatey package
- Signature/checksum verification of downloaded archives
