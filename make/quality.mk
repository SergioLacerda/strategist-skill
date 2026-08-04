.PHONY: \
	lint complexity-report go-file-size-report \
	install-gocognit quality-budget-gate \
	install-govulncheck vuln vuln-ci \
	cover cover-gate cover-html test-report

lint: fmt-check
	$(GOLANGCI_LINT) run ./...
	@$(MAKE) complexity-report
	@$(MAKE) go-file-size-report

# complexity-report lists files that contain functions with cognitive complexity > 7.
complexity-report:
	@$(MAKE) install-gocognit
	@bash scripts/complexity-report.sh "$(GOCOGNIT)" "$(CURDIR)" "$(COMPLEXITY_THRESHOLD)"

# go-file-size-report lists primary Go source files over 200 lines.
go-file-size-report:
	@bash scripts/go-file-size-report.sh

install-gocognit:
	@command -v $(GOCOGNIT) >/dev/null 2>&1 || GOCACHE=$(GOCACHE) go install github.com/uudashr/gocognit/cmd/gocognit@$(GOCOGNIT_VERSION)

quality-budget-gate: install-gocognit
	bash scripts/check-quality-budgets.sh "$(QUALITY_BUDGETS)" "$(GOCOGNIT)" "$(COMPLEXITY_THRESHOLD)"

install-govulncheck:
	GOCACHE=$(GOCACHE) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

vuln:
	$(GOVULNCHECK) ./...

vuln-ci: install-govulncheck vuln

# cover shows per-package coverage (each package measured against itself).
cover:
	@mkdir -p $(COVERAGE_DIR)
	@bash scripts/coverage-per-package.sh "$(COVERAGE_PKGS)" "$(COVERAGE_PROFILE)" "$(GOCACHE)"

# cover-gate fails the build when a package falls below its manifest threshold.
cover-gate:
	bash scripts/check-coverage-gate.sh "$(COVERAGE_MANIFEST)" "$(COVERAGE_DIR)" "$(GOCACHE)"

# test-report prints one status row per test style (unit, spec, integration,
# eval, eval-promptfoo, web) using the metric that fits each style.
test-report:
	bash scripts/test-style-report.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

# cover-html writes an HTML coverage report without opening a browser.
cover-html:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE=$(GOCACHE) go test -race -coverprofile=$(COVERAGE_PROFILE) -coverpkg=./internal/... ./internal/... ./tests/integration/...
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "report written to $(COVERAGE_HTML)"
