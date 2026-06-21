# Quickstart — Strategist

Five steps from zero to first mission. No lore.

## Prerequisites

- [Claude Code](https://claude.ai/code) installed

## 5 Steps

**1. Get the binary**

_If you have Go:_

```bash
go install github.com/SergioLacerda/strategist-skill/cmd/strategist@latest
```

_No Go? Download a pre-built binary from [GitHub Releases](https://github.com/SergioLacerda/strategist-skill/releases) — covers Linux, macOS, and Windows._

_Linux / macOS / WSL convenience script:_

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

_Windows: download `strategist-windows-amd64.zip` from GitHub Releases, extract, add to PATH._

**2. Configure the skill**

Run once in your target repository:

```bash
strategist install --wizard
```

The wizard creates `.strategist/` with the agent config. Accept the defaults to get started.

**3. Start discovery**

Open Claude Code in your repository and invoke:

```
/strategist <describe your task>
```

The Discoverer gathers requirements and writes a report to `.analysis/`.

**4. Review the spec**

The Spec Writer refines the report into an actionable specification. **This is the approval gate** — review it and confirm before any code is written.

**5. Execute**

After approval, the Executor implements exactly the approved spec.

---

→ [Full technical guide](https://sergiolacerda.github.io/strategist-skill/pragmatic/)  
→ [CLI reference](docs/cli-reference.md)  
→ [Configuration](docs/configuration.md)
