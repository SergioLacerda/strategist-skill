# Generated Documentation Index

This directory contains deterministic reference inventories generated from
repository sources. Use this page as the directory landing surface; it is an
index only and does not duplicate generated content.

## Inventories

- [Contract Inventory](contract-index.md) — machine contracts and their origins.
- [CLI Inventory](command-tree.md) — the built `strategist` command tree.
- [Schema Index](schema-index.md) — embedded schema references.
- [Event Catalog](event-catalog.md) — emitted event and telemetry references.
- [Quality Budgets](quality-budgets.md) — generated quality policy values.
- [Coverage Policy](coverage-policy.md) — generated coverage manifest policy.
- [Evaluation Scenarios](eval-scenarios.md) — generated evaluation scenario references.

## Provenance

The inventory files are generated and must not be edited manually. Regenerate
them through the sole entrypoint `make docs-generate`; the provenance header in
each file identifies its source and generator.
