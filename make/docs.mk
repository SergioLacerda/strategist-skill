.PHONY: docs-generate

# docs-generate regenerates every file under docs/generated/ from its
# deterministic source. Running it twice with no source changes must produce
# zero diff — that determinism is what docs-generated-gate (make/governance.mk)
# checks. See docs/adr/0025-generated-documentation-anti-drift.md.
#
# command-tree.md requires the built binary (bin/strategist) to exist and be
# up to date with the current source — run `make build` first if it is stale.
docs-generate:
	@mkdir -p docs/generated
	bash scripts/generate-command-tree.sh
	bash scripts/generate-schema-index.sh
	bash scripts/generate-contract-index.sh
	bash scripts/generate-event-catalog.sh
	bash scripts/generate-quality-budgets.sh
	bash scripts/generate-coverage-policy.sh
	bash scripts/generate-eval-scenarios.sh
