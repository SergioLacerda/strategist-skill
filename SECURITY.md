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
