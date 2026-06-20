# Security Audit Remediation — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remediate 6 verified security findings from the strategist-skill audit, bringing the project from 70/100 to 85+/100.

**Architecture:** Three independent code changes (telemetry config, path guard, integrity warning) plus two CI/CD changes (cosign signing in release workflow, goreleaser asset upload) and three documentation files (SECURITY.md, README update, CHANGELOG entry).

**Tech Stack:** Go 1.26, cobra, OpenTelemetry Go SDK, GitHub Actions, cosign (Sigstore keyless), GoReleaser v2.

---

## Context for implementor

The module path is `github.com/SergioLacerda/strategist-skill`.

Two audit findings were **already resolved** and are excluded from this plan:
- Bootstrap checksum — `bootstrap.sh` already verifies SHA256.
- Dependabot — `.github/dependabot.yml` already covers Go, npm, and Actions.

Run tests with: `make test` (all) or `make test-lite` (fast subset, no network).

---

## Task 1: Flip telemetry INSECURE default (F1)

**Files:**
- Modify: `internal/telemetry/config.go`
- Modify: `internal/telemetry/telemetry_test.go`

This is a **breaking change** — plaintext OTLP users must now explicitly set `OTEL_EXPORTER_OTLP_INSECURE=true`.

### Step 1: Update the failing test first

In `internal/telemetry/telemetry_test.go`, find `TestFromEnv_defaults`. The assertion `Insecure != true` will need to become `Insecure != false`. Update it now so the test reflects the new contract:

```go
func TestFromEnv_defaults(t *testing.T) {
    t.Setenv("OTEL_SERVICE_NAME", "")
    t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
    t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "")

    cfg := FromEnv()
    if cfg.ServiceName != "strategist" {
        t.Errorf("expected default ServiceName=strategist, got %q", cfg.ServiceName)
    }
    if cfg.Insecure != false {
        t.Error("expected Insecure=false by default (TLS required)")
    }
    if cfg.Endpoint != "" {
        t.Errorf("expected empty Endpoint, got %q", cfg.Endpoint)
    }
}
```

Also add a new test for the explicit opt-in:

```go
func TestFromEnv_insecure_explicit_optin(t *testing.T) {
    t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
    cfg := FromEnv()
    if !cfg.Insecure {
        t.Error("expected Insecure=true when env is 'true'")
    }
}
```

### Step 2: Run the test — expect it to fail

```bash
go test ./internal/telemetry/ -run TestFromEnv_defaults -v
```

Expected: `FAIL — expected Insecure=false by default`

### Step 3: Flip the logic in config.go

In `internal/telemetry/config.go`, replace the `FromEnv` function and struct comment:

```go
// Config holds telemetry configuration read from standard OTel environment variables.
type Config struct {
    Endpoint    string // OTEL_EXPORTER_OTLP_ENDPOINT
    ServiceName string // OTEL_SERVICE_NAME
    Insecure    bool   // OTEL_EXPORTER_OTLP_INSECURE — default false (TLS required).
                      // Set to "true" to allow plaintext gRPC (dev/self-hosted only).
}

// FromEnv reads OTel configuration from environment variables.
// If OTEL_SERVICE_NAME is unset, defaults to "strategist".
// If OTEL_EXPORTER_OTLP_INSECURE is unset, insecure is false (TLS required).
// Set OTEL_EXPORTER_OTLP_INSECURE=true to allow plaintext connections.
func FromEnv() Config {
    svcName := os.Getenv("OTEL_SERVICE_NAME")
    if svcName == "" {
        svcName = "strategist"
    }
    insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true"
    return Config{
        Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
        ServiceName: svcName,
        Insecure:    insecure,
    }
}
```

### Step 4: Run all telemetry tests

```bash
go test ./internal/telemetry/ -v
```

Expected: all PASS.

### Step 5: Commit

```bash
git add internal/telemetry/config.go internal/telemetry/telemetry_test.go
git commit -m "fix(telemetry): require TLS by default — INSECURE now opt-in

BREAKING CHANGE: OTEL_EXPORTER_OTLP_INSECURE default changed from true
to false. Plaintext OTLP users must now set OTEL_EXPORTER_OTLP_INSECURE=true
explicitly. Closes security finding F1."
```

---

## Task 2: Add SanitizePath guard to telemetry schema (F3)

**Files:**
- Modify: `internal/telemetry/schema.go`
- Create: `internal/telemetry/sanitize_test.go`

This adds a guardrail API. No existing call sites change today.

### Step 1: Write the failing test

Create `internal/telemetry/sanitize_test.go`:

```go
package telemetry

import (
    "testing"
)

func TestSanitizePath_absolute(t *testing.T) {
    got := SanitizePath("/home/user/projects/myapp")
    if got != "<redacted-path>" {
        t.Errorf("expected <redacted-path>, got %q", got)
    }
}

func TestSanitizePath_relative(t *testing.T) {
    got := SanitizePath(".analysis/refined")
    if got != ".analysis/refined" {
        t.Errorf("expected path unchanged, got %q", got)
    }
}

func TestSanitizePath_empty(t *testing.T) {
    got := SanitizePath("")
    if got != "" {
        t.Errorf("expected empty string unchanged, got %q", got)
    }
}

func TestSanitizePath_windows_absolute(t *testing.T) {
    got := SanitizePath(`C:\Users\user\projects`)
    // filepath.IsAbs on Linux returns false for Windows paths — document this.
    // The function is a best-effort guard for the current OS.
    _ = got // no assertion: platform-dependent
}
```

### Step 2: Run test — expect compile error (function not yet defined)

```bash
go test ./internal/telemetry/ -run TestSanitizePath -v
```

Expected: `FAIL — undefined: SanitizePath`

### Step 3: Add SanitizePath to schema.go

At the bottom of `internal/telemetry/schema.go`, add:

```go
import "path/filepath"  // add to existing imports if not present
```

Then append:

```go
const redactedPath = "<redacted-path>"

// SanitizePath replaces absolute paths with a sentinel before use as a span attribute.
// Call this on any string that may originate from user filesystem input before
// attaching it to a trace span.
func SanitizePath(p string) string {
    if filepath.IsAbs(p) {
        return redactedPath
    }
    return p
}
```

> **Note:** `schema.go` currently has no imports — add `"path/filepath"` at the top.

### Step 4: Run tests

```bash
go test ./internal/telemetry/ -run TestSanitizePath -v
```

Expected: all three assertions PASS.

Also verify the telemetry lite slice still compiles:

```bash
make test-telemetry-lite
```

Expected: PASS (sanitize_test.go uses only stdlib — it fits the lite slice).

### Step 5: Commit

```bash
git add internal/telemetry/schema.go internal/telemetry/sanitize_test.go
git commit -m "feat(telemetry): add SanitizePath guard for future span attributes

Preventive guardrail: any future code adding filesystem paths as OTel
span attributes must call SanitizePath first. Closes security finding F3."
```

---

## Task 3: Add cosign keyless signing to release workflow (F2 — part 1)

**Files:**
- Modify: `.github/workflows/release.yml`

### Step 1: Resolve the cosign-installer pinned hash

Look up the latest release at https://github.com/sigstore/cosign-installer/releases.
Get the full commit SHA for the latest tag (e.g., v3.8.1).
The project pins all actions by commit SHA — follow the same pattern:

```
sigstore/cosign-installer@<commit-sha> # v3.x.y
```

To get the SHA for a tag:
```bash
gh api repos/sigstore/cosign-installer/git/refs/tags/v3.8.1 --jq '.object.sha'
# If it's a tag object (not commit), dereference:
gh api repos/sigstore/cosign-installer/git/tags/<sha-from-above> --jq '.object.sha'
```

### Step 2: Add cosign steps to release.yml

In `.github/workflows/release.yml`, the current step order is:
1. `actions/checkout`
2. `actions/setup-go`
3. `goreleaser/goreleaser-action` (runs release)
4. `anchore/sbom-action` (generates SBOM)
5. `actions/attest-build-provenance` (SLSA provenance)

Insert two new steps **between goreleaser and sbom-action**:

```yaml
      - name: Install cosign
        uses: sigstore/cosign-installer@<pinned-sha> # v3.x.y

      - name: Sign release binaries (keyless)
        run: |
          for f in dist/strategist-*; do
            [[ "$f" == *.bundle ]] && continue
            cosign sign-blob --yes "$f" --bundle "${f}.bundle"
          done
        env:
          COSIGN_EXPERIMENTAL: "1"
```

The `[[ "$f" == *.bundle ]]` guard prevents re-signing already-created bundle files if the glob matches them on re-run.

The `id-token: write` permission is already present in the workflow — cosign keyless requires it and it's covered.

### Step 3: Verify workflow YAML is valid

```bash
cat .github/workflows/release.yml | python3 -c "import sys,yaml; yaml.safe_load(sys.stdin); print('YAML OK')"
```

Expected: `YAML OK`

### Step 4: Commit

```bash
git add .github/workflows/release.yml
git commit -m "feat(ci): add cosign keyless signing to release workflow

Signs each release binary with Sigstore keyless cosign and uploads
.bundle files alongside binaries. Requires id-token:write (already present).
Closes security finding F2 (part 1/2)."
```

---

## Task 4: Upload bundle files as release assets (F2 — part 2)

**Files:**
- Modify: `.goreleaser.yaml`

GoReleaser currently uses `format: binary` in archives, which means it uploads raw binaries directly (no tarball). Bundle files produced by cosign in Task 3 live in `dist/` but are not picked up automatically.

### Step 1: Add extra_files to goreleaser config

In `.goreleaser.yaml`, add an `extra_files` section under `release:`:

```yaml
release:
  github:
    owner: SergioLacerda
    name: strategist-skill
  draft: false
  extra_files:
    - glob: dist/strategist-*.bundle
```

Wait — the cosign step runs **after** goreleaser in the workflow (Task 3 inserts it after goreleaser). GoReleaser has already uploaded assets by then. The correct approach is to upload bundles separately using `gh release upload` after cosign runs.

**Revised approach:** Replace the `extra_files` idea with an explicit upload step in `release.yml` (not in goreleaser):

Add after the cosign signing step (Task 3):

```yaml
      - name: Upload cosign bundles to release
        run: |
          TAG="${GITHUB_REF_NAME}"
          for bundle in dist/strategist-*.bundle; do
            gh release upload "$TAG" "$bundle" --clobber
          done
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Step 2: Update release.yml with the upload step

Add the `gh release upload` step to `.github/workflows/release.yml`, after the `Sign release binaries` step.

### Step 3: Verify YAML

```bash
cat .github/workflows/release.yml | python3 -c "import sys,yaml; yaml.safe_load(sys.stdin); print('YAML OK')"
```

### Step 4: Commit

```bash
git add .github/workflows/release.yml
git commit -m "feat(ci): upload cosign bundle files to GitHub release assets

After signing, bundles are uploaded via gh release upload so users
can verify with cosign verify-blob --bundle. Closes security finding F2 (part 2/2)."
```

---

## Task 5: Config integrity warning (F4)

**Files:**
- Create: `internal/integrity/warning.go`
- Create: `internal/integrity/warning_test.go`
- Modify: `internal/install/active_yaml.go` (write lock after active.yaml)
- Modify: `cmd/strategist/root.go` (check lock on startup)

### Step 1: Write the failing tests

Create `internal/integrity/warning_test.go`:

```go
package integrity_test

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/SergioLacerda/strategist-skill/internal/integrity"
)

func TestWriteLock_and_CheckUnmodified(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "active.yaml")
    lockPath := filepath.Join(dir, ".config.lock")

    if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
        t.Fatal(err)
    }

    if err := integrity.WriteLock(configPath, lockPath); err != nil {
        t.Fatalf("WriteLock: %v", err)
    }

    modified, err := integrity.IsModified(configPath, lockPath)
    if err != nil {
        t.Fatalf("IsModified: %v", err)
    }
    if modified {
        t.Error("expected IsModified=false immediately after WriteLock")
    }
}

func TestIsModified_detects_external_change(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "active.yaml")
    lockPath := filepath.Join(dir, ".config.lock")

    if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
        t.Fatal(err)
    }
    if err := integrity.WriteLock(configPath, lockPath); err != nil {
        t.Fatal(err)
    }

    // Simulate external edit: rewrite file 1 second later
    time.Sleep(10 * time.Millisecond)
    if err := os.WriteFile(configPath, []byte("mode: pragmatic\n"), 0o644); err != nil {
        t.Fatal(err)
    }

    modified, err := integrity.IsModified(configPath, lockPath)
    if err != nil {
        t.Fatalf("IsModified: %v", err)
    }
    if !modified {
        t.Error("expected IsModified=true after external write")
    }
}

func TestIsModified_no_lock_file(t *testing.T) {
    dir := t.TempDir()
    configPath := filepath.Join(dir, "active.yaml")
    lockPath := filepath.Join(dir, ".config.lock")

    if err := os.WriteFile(configPath, []byte("mode: epic\n"), 0o644); err != nil {
        t.Fatal(err)
    }

    // No lock file written — IsModified should return false (not an error)
    modified, err := integrity.IsModified(configPath, lockPath)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if modified {
        t.Error("expected IsModified=false when no lock file exists")
    }
}
```

### Step 2: Run tests — expect compile error

```bash
go test ./internal/integrity/... -v
```

Expected: `FAIL — cannot find package "internal/integrity"`

### Step 3: Create internal/integrity/warning.go

```go
// Package integrity provides config integrity helpers for the Strategist CLI.
package integrity

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"
)

type configLock struct {
    Mtime time.Time `json:"mtime"`
    Path  string    `json:"path"`
}

// WriteLock records the current mtime of configPath into lockPath.
// Call this immediately after writing active.yaml during install.
func WriteLock(configPath, lockPath string) error {
    info, err := os.Stat(configPath)
    if err != nil {
        return fmt.Errorf("integrity: stat config: %w", err)
    }
    lock := configLock{Mtime: info.ModTime().UTC(), Path: configPath}
    data, err := json.Marshal(lock)
    if err != nil {
        return fmt.Errorf("integrity: marshal lock: %w", err)
    }
    if err := os.WriteFile(lockPath, data, 0o600); err != nil {
        return fmt.Errorf("integrity: write lock: %w", err)
    }
    return nil
}

// IsModified reports whether configPath has been modified since the last WriteLock.
// Returns false (not an error) when no lock file exists — first install scenario.
func IsModified(configPath, lockPath string) (bool, error) {
    data, err := os.ReadFile(lockPath)
    if errors.Is(err, os.ErrNotExist) {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("integrity: read lock: %w", err)
    }

    var lock configLock
    if err := json.Unmarshal(data, &lock); err != nil {
        return false, fmt.Errorf("integrity: parse lock: %w", err)
    }

    info, err := os.Stat(configPath)
    if errors.Is(err, os.ErrNotExist) {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("integrity: stat config: %w", err)
    }

    return !info.ModTime().UTC().Equal(lock.Mtime), nil
}
```

### Step 4: Run tests

```bash
go test ./internal/integrity/... -v
```

Expected: all three tests PASS.

### Step 5: Write lock after active.yaml in active_yaml.go

In `internal/install/active_yaml.go`, at the end of `writeActiveYAML`, after the existing `os.WriteFile` call:

```go
import "github.com/SergioLacerda/strategist-skill/internal/integrity"

// at end of writeActiveYAML, after os.WriteFile succeeds:
lockPath := filepath.Join(strategistDir, ".config.lock")
if err := integrity.WriteLock(path, lockPath); err != nil {
    // non-fatal: warn but don't block install
    fmt.Fprintf(os.Stderr, "[Strategist] WARN: could not write config lock: %v\n", err)
}
```

### Step 6: Add startup check to root.go

In `cmd/strategist/root.go`, inside `PersistentPreRunE`, after the existing telemetry setup, add:

```go
import (
    "github.com/SergioLacerda/strategist-skill/internal/integrity"
)

// inside PersistentPreRunE, after the slog.InfoContext call:
configPath := ".strategist/active.yaml"
lockPath := ".strategist/.config.lock"
if modified, err := integrity.IsModified(configPath, lockPath); err == nil && modified {
    fmt.Fprintf(os.Stderr,
        "[Strategist] WARN: active.yaml was modified outside the CLI.\n"+
            "             Config integrity unverified. Re-run `strategist install` to acknowledge.\n")
}
```

### Step 7: Add .config.lock to .gitignore

Check `.gitignore` for a `.strategist/` entry and add `.config.lock` if needed:

```bash
grep -n "strategist" /home/sergio/dev/strategist-skill/.gitignore
```

If `.strategist/` is already ignored, no change needed. If not, append:

```
.strategist/.config.lock
```

### Step 8: Run full test suite

```bash
make test
```

Expected: PASS.

### Step 9: Commit

```bash
git add internal/integrity/warning.go internal/integrity/warning_test.go \
        internal/install/active_yaml.go cmd/strategist/root.go
git commit -m "feat(integrity): warn when active.yaml modified outside CLI

Writes .strategist/.config.lock after install and checks mtime on
startup. Emits a WARN to stderr if the file was changed externally.
Non-blocking: users who intentionally edit active.yaml are not stopped.
Closes security finding F4."
```

---

## Task 6: Create SECURITY.md (F5)

**Files:**
- Create: `SECURITY.md`

### Step 1: Create the file

```markdown
# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest minor | ✅ Security fixes |
| Previous minor | ⚠️ Critical fixes only |
| Older | ❌ Not supported |

## Reporting a Vulnerability

Please **do not** open a public GitHub issue for security vulnerabilities.

Report by emailing: **sergio.lacerda.vieira@gmail.com**

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment (what an attacker could do)
- Any suggested mitigations

Expected response time: **5 business days**.

## Verifying Release Integrity

Every release binary is protected by two independent supply chain controls.

### 1. SLSA Provenance (via GitHub Attestation)

```bash
gh attestation verify strategist-linux-amd64 \
  --owner SergioLacerda
```

This verifies the binary was built by the official GitHub Actions workflow.
Requires the [GitHub CLI](https://cli.github.com/).

### 2. Cosign Keyless Signature

Download the binary and its `.bundle` file from the release assets, then verify:

```bash
cosign verify-blob strategist-linux-amd64 \
  --bundle strategist-linux-amd64.bundle \
  --certificate-identity-regexp "https://github.com/SergioLacerda/strategist-skill/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

Requires [cosign](https://docs.sigstore.dev/cosign/system_config/installation/).

### SHA256 Checksums

A `SHA256SUMS` file is included with every release for quick integrity checks:

```bash
sha256sum --check SHA256SUMS
```

## Branch Protection (Recommended for forks)

If forking this repository, enable these settings under **Settings → Branches → main**:

- Require pull request reviews: **2 approvals**
- Dismiss stale pull request approvals when new commits are pushed: **enabled**
- Require status checks to pass before merging: `test`, `lint`
- Require commits to be signed: **enabled**
- Restrict who can push to matching branches: **enabled**
```

### Step 2: Verify the file renders correctly

```bash
cat SECURITY.md | wc -l
```

Expected: ~65 lines.

### Step 3: Commit

```bash
git add SECURITY.md
git commit -m "docs: add SECURITY.md with disclosure policy and release verification"
```

---

## Task 7: Add "Verifying releases" to README (F2 — documentation)

**Files:**
- Modify: `README.md`

### Step 1: Find the right insertion point

```bash
grep -n "##" /home/sergio/dev/strategist-skill/readme.md | head -20
```

Insert the new section before the last `##` section (typically "Contributing" or "License").

### Step 2: Add the section

```markdown
## Verifying releases

Every release binary is signed and attested. See [SECURITY.md](SECURITY.md) for full verification instructions.

**Quick verify with SLSA provenance:**
```bash
gh attestation verify strategist-linux-amd64 --owner SergioLacerda
```

**Quick verify with cosign:**
```bash
cosign verify-blob strategist-linux-amd64 \
  --bundle strategist-linux-amd64.bundle \
  --certificate-identity-regexp "https://github.com/SergioLacerda/strategist-skill/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```
```

### Step 3: Commit

```bash
git add readme.md
git commit -m "docs(readme): add release verification section pointing to SECURITY.md"
```

---

## Task 8: CHANGELOG breaking change entry (F1)

**Files:**
- Modify: `CHANGELOG.md` (create if it does not exist)

### Step 1: Check for existing CHANGELOG

```bash
ls /home/sergio/dev/strategist-skill/CHANGELOG.md 2>/dev/null || echo "NOT_FOUND"
```

### Step 2: Add or create the entry

If file exists, prepend to the top (after any title line). If not, create:

```markdown
# Changelog

## [Unreleased]

### Security

- **BREAKING:** `OTEL_EXPORTER_OTLP_INSECURE` now defaults to `false` (TLS required).
  If you send telemetry to a plaintext gRPC endpoint, add `OTEL_EXPORTER_OTLP_INSECURE=true`
  to your environment. This change closes a security finding where telemetry data could be
  intercepted on networks without TLS.
```

### Step 3: Run full test suite one final time

```bash
make test
```

Expected: PASS.

### Step 4: Commit

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): document INSECURE default breaking change under Security"
```

---

## Execution order summary

```
Task 1 (T1, T8-depends)  →  config.go flip          independent
Task 2 (T2)              →  SanitizePath guard       independent
Task 3 (T3)              →  cosign in release.yml    before Task 4
Task 4 (T4)              →  bundle upload step       after Task 3
Task 5 (T5)              →  integrity warning        independent
Task 6 (T6)              →  SECURITY.md              before Task 7
Task 7 (T7)              →  README update            after Task 6
Task 8 (T8)              →  CHANGELOG entry          any time
```
