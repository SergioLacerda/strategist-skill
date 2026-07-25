# Strategist Skill

[![Release](https://img.shields.io/github/v/release/SergioLacerda/strategist-skill?label=release)](https://github.com/SergioLacerda/strategist-skill/releases)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev)
[![License](https://img.shields.io/github/license/SergioLacerda/strategist-skill)](LICENSE)

A governed AI mission orchestrator. Coordinates multi-phase work through three pluggable slots: Ranger (discovery) → Archivist (refinement) → Sniper (execution).

## Choose your journey

| | |
|---|---|
| **[Pragmatic Mode](https://sergiolacerda.github.io/strategist-skill/pragmatic/)** | Direct technical guide: pipeline, neutral role names, 5-minute quickstart, any-LLM support. Recommended for most. |
| **[Epic Mode](https://sergiolacerda.github.io/strategist-skill/epic/)** | The full narrative experience with the system's lore. |

> Prefer to read here? Continue below.

## Installation

Installation is two stages: get the binary, then configure the skill.

### Step 1 — Get the binary

**Recommended (requires Go):**

```bash
go install github.com/SergioLacerda/strategist-skill/cmd/strategist@latest
```

**Universal — download a pre-built binary from [GitHub Releases](https://github.com/SergioLacerda/strategist-skill/releases)** (no Go required; works on Linux, macOS, Windows).

<details>
<summary>Convenience script — Linux / macOS / WSL (optional)</summary>

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

**Windows:** use the GitHub Releases link above to download `strategist-windows-amd64.zip`, extract, and add the binary to your PATH.
</details>

### Step 2 — Configure the skill

Run once in your target repository:

```bash
strategist install --wizard
```

The wizard creates `.strategist/` with the agent config. Accept the defaults to get started.

## Usage

Invoke from any agent after installation:

```
/strategist <your mission prompt>
```

See [`docs/`](docs/) for full documentation including [CLI reference](docs/cli-reference.md) and [configuration](docs/configuration.md).

## Verifying releases

Every release binary is protected by two independent supply chain controls. See [SECURITY.md](SECURITY.md#verifying-release-integrity) for full verification instructions.

**GitHub build provenance** (via GitHub Attestation — not a formal SLSA level claim):

```bash
gh attestation verify strategist-linux-amd64 --owner SergioLacerda
```

**Cosign keyless signature** (download binary + `.bundle` from release assets):

```bash
cosign verify-blob strategist-linux-amd64 \
  --bundle strategist-linux-amd64.bundle \
  --certificate-identity-regexp "https://github.com/SergioLacerda/strategist-skill/.*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"
```

**SHA256 checksum:**

```bash
sha256sum --check SHA256SUMS
```

## Security

See [SECURITY.md](SECURITY.md) for the vulnerability disclosure policy.

## Credits and attribution

`strategist` acts as an orchestration layer for governed workflows and may integrate external skills as specialized role providers.

### Recognized base skills

- `brainstorming` — project `obra/superpowers`  
  Source: <https://claudemarketplaces.com/skills/obra/superpowers/brainstorming>
- `openspec` — project `itechmeat/llm-code`  
  Source: <https://claudemarketplaces.com/skills/itechmeat/llm-code/openspec>

### Attribution policy

- Preserve the name, upstream project, and public URL whenever an external skill is integrated as a provider.
- Do not imply ownership over upstream prompts, artifacts, or implementation.
- Prefer canonical manifests/adapters in `.strategist/skills/<provider>/skill.yaml` instead of duplicating upstream content.

## Documentation

- [Quickstart](QUICKSTART.md)
- [Technical guide (English)](docs/onboarding/readme-en.md)
- [Detailed documentation (English)](docs/onboarding/readme-detailed-en.md)
- [CLI reference](docs/cli-reference.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)

## License

CC BY-NC 4.0. Commercial use requires prior authorization.

- Repository: <https://github.com/SergioLacerda/strategist-skill>
- Documentation: <https://sergiolacerda.github.io/strategist-skill/index.html?lang=en>
- Full text: [LICENSE](LICENSE)
