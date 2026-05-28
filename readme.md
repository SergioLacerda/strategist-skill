# Strategist Skill + SDD Harness

![CI](https://img.shields.io/badge/CI‑passing‑brightgreen)
![License](https://img.shields.io/badge/License‑MIT‑blue)
![Docs](https://img.shields.io/badge/Docs‑available‑orange)
![Version](https://img.shields.io/badge/Version‑1.0‑yellow)

**Strategist** is an autonomous skill that orchestrates analysis, automatic discovery, refinement, planning, and hand‑off to a **Hunter** within the **SDD Harness** governance runtime. It guarantees that AI‑driven work follows a convergent, approved, and traceable workflow.

---

### Key Capabilities
- **Discovery & Refinement Automation** – runs automatically when no conflicts are detected.
- **Approval Gate** – never executes code without explicit human approval.
- **Hunter Delegation** – delegates implementation to specialized providers (e.g., `caveman`, `rewriter`).
- **Governance Integration** – works with the `.sdd` contracts, mandates, and runtime state.

---

### Architecture Overview
![General Flow](/home/sergio/dev/strategist-skill/docs/fluxo-geral.png)

![Integration Flow](/home/sergio/dev/strategist-skill/docs/fluxo-integracao.png)

---

For a complete technical description, usage details, and deeper documentation, see the [detailed README](readme_detailed.md).
