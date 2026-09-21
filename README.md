# Strategist Skill

[![Release](https://img.shields.io/github/v/release/SergioLacerda/strategist-skill?label=release)](https://github.com/SergioLacerda/strategist-skill/releases)
[![CI](https://github.com/SergioLacerda/strategist-skill/actions/workflows/test.yml/badge.svg)](https://github.com/SergioLacerda/strategist-skill/actions/workflows/test.yml)
[![Coverage](https://img.shields.io/badge/coverage-92.1%25-green)](docs/test-styles.md)
[![Mutation](https://img.shields.io/badge/mutation-passing-brightgreen)](scripts/mutation-role-weapon.sh)
[![Go](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev)
[![License](https://img.shields.io/github/license/SergioLacerda/strategist-skill)](LICENSE)

A governed AI mission orchestrator. Coordinates multi-phase work through three pluggable slots: Ranger (discovery) → Archivist (refinement) → Sniper (execution).

## Start here

In under two minutes, [Quickstart](QUICKSTART.md) takes you from installation
to `/strategist <your mission prompt>`, the refined package, and the Approval
Gate. You do not need to choose a mode before the first mission.

Strategist coordinates discovery, refinement, and approved documentation
materialization. Acceptance at the Approval Gate authorizes only declared
documentation targets; source-code, test, script, CI, and configuration work
remains a separate implementation handoff.

## Documentation paths

- [Pragmatic Mode](https://sergiolacerda.github.io/strategist-skill/pragmatic/) — direct technical guide.
- [Epic Mode](https://sergiolacerda.github.io/strategist-skill/epic/) — the narrative documentation experience.
- [`docs/`](docs/) — the repository documentation index.

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

- **[Conceptual quickstart](docs/onboarding/quickstart-concepts.md)** — roles, guiding questions, lifecycle, artifacts, and boundaries on one page.
- [Quickstart](QUICKSTART.md)
- [Technical guide (English)](docs/onboarding/readme-en.md)
- [Detailed documentation (English)](docs/onboarding/readme-detailed-en.md)
- [CLI reference](docs/cli-reference.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture/overview.md)

## License

MIT. Open source — free to use, modify, and redistribute, with attribution
per the LICENSE terms. See [LICENSE](LICENSE) for the full text, attribution
guidance, and professional services terms.

- Repository: <https://github.com/SergioLacerda/strategist-skill>
- Documentation: <https://sergiolacerda.github.io/strategist-skill/index.html?lang=en>
- Full text: [LICENSE](LICENSE)
