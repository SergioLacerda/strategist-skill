.PHONY: \
	analysis-structure-gate docs-governance-gate docs-generated-gate governance-check \
	convergence-check contract-consistency-gate

analysis-structure-gate:
	bash scripts/check-refined-structure.sh

docs-governance-gate:
	bash scripts/check-docs-governance.sh

# docs-generated-gate fails when docs/generated/ is missing, or when
# regenerating it produces a diff against the committed copy — i.e. someone
# edited a generated file by hand, or a source changed without regenerating.
# Never auto-fixes: same "fail, don't silently reconcile" posture as the
# other gates in this file. See docs/adr/0025-generated-documentation-anti-drift.md.
docs-generated-gate: build
	@test -d docs/generated || { echo "FAIL: docs/generated/ is missing — run 'make docs-generate'"; exit 1; }
	$(MAKE) docs-generate
	@git diff --exit-code -- docs/generated/ || { \
	  echo "FAIL: docs/generated/ is out of date — run 'make docs-generate' and commit the diff"; \
	  exit 1; \
	}

contract-consistency-gate:
	bash scripts/check-contract-consistency.sh

convergence-check:
	@bash scripts/check-convergence.sh

governance-check:
	@bash scripts/check-governance-redirectors.sh
