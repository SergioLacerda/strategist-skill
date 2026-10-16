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
# command-tree.md is generated from the freshly built, platform-correct binary
# so the inventory cannot silently read a stale Unix or Windows executable.
docs-generate: build
	@mkdir -p docs/generated
	@bash scripts/generate-command-tree.sh "$(STRATEGIST_BIN)"
	@set -e; for generator in $(filter-out scripts/generate-command-tree.sh,$(DOC_GENERATOR_SCRIPTS)); do bash "$$generator"; done
