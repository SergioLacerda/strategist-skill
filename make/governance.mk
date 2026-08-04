.PHONY: \
	analysis-structure-gate docs-governance-gate governance-check convergence-check \
	contract-consistency-gate

analysis-structure-gate:
	bash scripts/check-refined-structure.sh

docs-governance-gate:
	bash scripts/check-docs-governance.sh

contract-consistency-gate:
	bash scripts/check-contract-consistency.sh

convergence-check:
	@bash scripts/check-convergence.sh

governance-check:
	@bash scripts/check-governance-redirectors.sh
