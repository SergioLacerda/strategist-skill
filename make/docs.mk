.PHONY: docs-generate

DOC_GENERATOR_SCRIPTS := \
	scripts/generate-command-tree.sh \
	scripts/generate-schema-index.sh \
	scripts/generate-contract-index.sh \
	scripts/generate-event-catalog.sh \
	scripts/generate-taxonomy-inventory.sh \
	scripts/generate-quality-budgets.sh \
	scripts/generate-coverage-policy.sh \
	scripts/generate-eval-scenarios.sh

# docs-generate regenerates every file under docs/generated/ from its
# deterministic source. Running it twice with no source changes must produce
# zero diff — that determinism is what docs-generated-gate (make/governance.mk)
# checks. See docs/adr/0025-generated-documentation-anti-drift.md.
#
# command-tree.md requires the built binary (bin/strategist) to exist and be
# up to date with the current source — run `make build` first if it is stale.
docs-generate:
	@mkdir -p docs/generated
	@set -e; for generator in $(DOC_GENERATOR_SCRIPTS); do bash "$$generator"; done
