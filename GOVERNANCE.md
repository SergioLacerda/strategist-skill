# Governance

Strategist is currently a **solo-maintainer** project (see
[MAINTAINERS.md](MAINTAINERS.md)). This document describes the decision
process as it exists today — it will be revised if/when the maintainer set
grows, rather than pre-declaring a multi-maintainer model this project
doesn't have yet.

## Decision making

- The maintainer has final say on scope, architecture, and release
  timing. There is no voting body.
- Design-level decisions for the runtime contracts (`internal/embed/defaults/contracts/`,
  slot/role definitions, schemas) are recorded as ADRs under `docs/adr/` —
  see an existing ADR for the format. A change to a normative contract
  should generally come with a new or amended ADR explaining why.
- External contributions (issues, PRs) are welcome and reviewed per
  [CONTRIBUTING.md](CONTRIBUTING.md); acceptance is at the maintainer's
  discretion.

## Adding a maintainer

Not currently a defined process — there is only one maintainer. If this
changes, this document and [MAINTAINERS.md](MAINTAINERS.md) will be updated
together.

## Security

Vulnerability handling is described separately in [SECURITY.md](SECURITY.md).
