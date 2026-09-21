# Quickstart — Strategist

Five steps from installation to a reviewed mission. No mode selection is
needed before the first use.

## Prerequisites

- The `strategist` binary on your `PATH`.
- An agent host with the `/strategist` skill or slash command available.

## 5 Steps

**1. Get the binary**

_If you have Go:_

```bash
go install github.com/SergioLacerda/strategist-skill/cmd/strategist@latest
```

_No Go? Download a pre-built binary from [GitHub Releases](https://github.com/SergioLacerda/strategist-skill/releases)._

_Linux / macOS / WSL convenience script:_

```bash
curl -fsSL https://raw.githubusercontent.com/SergioLacerda/strategist-skill/main/bootstrap.sh | bash
```

_Windows: download the appropriate archive from GitHub Releases, extract it,
and add the binary to `PATH`._

**2. Configure the skill**

Run once in the target repository:

```bash
strategist install --wizard
```

Accept the defaults to create `.strategist/` and its mission configuration.

**3. Start a mission**

Open your agent host in the repository and invoke:

```
/strategist <describe your task>
```

Strategist discovers the scope, records uncertainties, and refines the request
into a reviewable package under `.analysis/refined/<mission_id>/`.

**4. Review the package**

Read `analysis.md`, `proposal.md`, `design.md`, and `tasks.md`. Confirm that
the objective, boundaries, evidence, and validation criteria match your intent.

**5. Respond at the Approval Gate**

Choose `accept`, `review`, or `reject`. Acceptance permits only the declared
documentation targets to be materialized. Any `implementation_handoff` for
source code, tests, scripts, CI, or configuration remains a separate coding
request; it is never authorized by this documentation gate.

---

→ [Conceptual quickstart](docs/onboarding/quickstart-concepts.md)  
→ [Full technical guide](docs/onboarding/readme-en.md)
→ [CLI reference](docs/cli-reference.md)  
→ [Configuration](docs/configuration.md)
