.PHONY: \
	lint lint-fix lint-status complexity-report go-file-size-report \
	mutation-role-weapon \
	coverage-manifest-check \
	install-gocognit quality-budget-gate \
	install-govulncheck vuln vuln-ci \
	cover cover-gate cover-html coverage-badge sync-test-styles-docs \
	sync-readme-badge coverage-docs-drift-check test-report

# lint-status makes lint verifiability explicit: when golangci-lint is absent
# the gate prints "lint: not verified" instead of passing silently. CI sets
# LINT_REQUIRED=1 (or runs the golangci-lint action) so absence there fails.
lint-status:
	@if [ -x "$(GOLANGCI_LINT)" ] || command -v "$(GOLANGCI_LINT)" >/dev/null 2>&1; then \
		echo "lint: golangci-lint available ($(GOLANGCI_LINT)); run 'make lint' to verify"; \
	elif [ "$(LINT_REQUIRED)" = "1" ]; then \
		echo "::error::lint: not verified - golangci-lint not found and LINT_REQUIRED=1" >&2; exit 1; \
	else \
		echo "lint: not verified (golangci-lint not installed; set LINT_REQUIRED=1 to make this fail)"; \
	fi

# lint is diagnostic-only: it must never rewrite source files.
lint: fmt-check
	GOCACHE="$(GOCACHE)" GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" GOTOOLCHAIN=$(PINNED_GOTOOLCHAIN) "$(GOLANGCI_LINT)" run ./...
	@$(MAKE) complexity-report
	@$(MAKE) go-file-size-report

# lint-fix applies only tool-supported repairs, then runs the same diagnostics
# as lint. Complexity and file-size findings remain manual work and therefore
# still fail here when they cannot be fixed automatically.
lint-fix:
	git ls-files -co --exclude-standard -z '*.go' | xargs -0r gofmt -w
	GOCACHE="$(GOCACHE)" GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" GOTOOLCHAIN=$(PINNED_GOTOOLCHAIN) "$(GOLANGCI_LINT)" run --fix ./...
	@$(MAKE) fmt-check
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
	@command -v "$(GOCOGNIT)" >/dev/null 2>&1 || GOCACHE="$(GOCACHE)" go install github.com/uudashr/gocognit/cmd/gocognit@$(GOCOGNIT_VERSION)

quality-budget-gate: install-gocognit
	bash scripts/check-quality-budgets.sh "$(QUALITY_BUDGETS)" "$(GOCOGNIT)" "$(COMPLEXITY_THRESHOLD)"

mutation-role-weapon:
	bash scripts/mutation-role-weapon.sh

install-govulncheck:
	GOCACHE="$(GOCACHE)" go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

vuln:
	GOTOOLCHAIN=$(PINNED_GOTOOLCHAIN) "$(GOVULNCHECK)" ./...

vuln-ci: install-govulncheck vuln

# cover shows per-package coverage (each package measured against itself),
# annotated with the same gate threshold check-coverage-gate.sh/CI enforce —
# report only, never fails the build (use cover-gate for that).
cover:
	@mkdir -p $(COVERAGE_DIR)
	@bash scripts/coverage-per-package.sh "$(COVERAGE_PKGS)" "$(COVERAGE_PROFILE)" "$(GOCACHE)" "$(COVERAGE_MANIFEST)"

# cover-gate fails the build when a package falls below its manifest threshold.
coverage-manifest-check:
	bash scripts/check-coverage-manifest.sh "$(COVERAGE_MANIFEST)" "$(COVERAGE_EXEMPTIONS)" "$(GOCACHE)"

cover-gate:
	bash scripts/check-coverage-gate.sh "$(COVERAGE_MANIFEST)" "$(COVERAGE_DIR)" "$(GOCACHE)" "$(COVERAGE_EXEMPTIONS)"

# test-report prints one status row per test style (unit, spec, integration,
# eval, eval-promptfoo, web) using the metric that fits each style.
test-report:
	bash scripts/test-style-report.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

# cover-html writes an HTML coverage report without opening a browser.
cover-html:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE="$(GOCACHE)" go test -race -coverprofile=$(COVERAGE_PROFILE) -coverpkg=./internal/... ./internal/... ./tests/integration/...
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "report written to $(COVERAGE_HTML)"

# coverage-badge generates SVG and JSON Shields endpoint badges from coverage data.
coverage-badge:
	@bash scripts/generate-coverage-badge.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

# sync-test-styles-docs synchronizes measured package coverage numbers into docs/test-styles.md.
sync-test-styles-docs:
	@bash scripts/sync-test-styles-docs.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

# sync-readme-badge rewrites the static README coverage badge from the measured value.
sync-readme-badge:
	@bash scripts/sync-readme-coverage-badge.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

# coverage-docs-drift-check fails (never rewrites) when README.md's badge or
# docs/test-styles.md differ from measured coverage by more than 1.0 point.
coverage-docs-drift-check:
	@bash scripts/check-coverage-docs-drift.sh "$(COVERAGE_DIR)" "$(GOCACHE)"

