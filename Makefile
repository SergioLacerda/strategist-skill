.PHONY: \
	fmt fmt-check mod-tidy mod-check vet build \
	test test-all integration spec validate-expanded validate-all \
	test-lite test-telemetry-lite test-compile-cache test-domain-architecture \
	ci-lint ci-test ci lint complexity-report go-file-size-report \
	quality-budget-gate install-gocognit \
	install-govulncheck vuln vuln-ci bench \
	cover cover-gate cover-html \
	analysis-structure-gate docs-governance-gate governance-check convergence-check \
	contract-consistency-gate \
	validate-fixtures install release-verify release-check install-goreleaser \
	check-release-artifacts check-release-assets release-reproducible-check release-test release-dry-run \
	release snapshot clean compile-skill build-site build-all \
	install-web lint-web test-web cover-web ci-web

GOCACHE ?= /tmp/go-build-cache

GOLANGCI_LINT       := $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
GOVULNCHECK         := $(shell which govulncheck 2>/dev/null || echo $(shell go env GOPATH)/bin/govulncheck)
GOCOGNIT            := $(shell which gocognit 2>/dev/null || echo $(shell go env GOPATH)/bin/gocognit)
GORELEASER          := $(shell which goreleaser 2>/dev/null || echo $(shell go env GOPATH)/bin/goreleaser)
GOVULNCHECK_VERSION ?= v1.1.4
GOCOGNIT_VERSION    ?= v1.2.1
GORELEASER_VERSION  ?= v2.12.2
COVERAGE_MANIFEST   := scripts/coverage-packages.tsv
COVERAGE_PKGS       := $(shell awk 'NF && $$1 !~ /^#/ {print $$1}' $(COVERAGE_MANIFEST))
COVERAGE_DIR        ?= coverage
COVERAGE_PROFILE    := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML       := $(COVERAGE_DIR)/coverage.html
QUALITY_BUDGETS     := scripts/quality-budgets.tsv
COMPLEXITY_THRESHOLD ?= 15

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

mod-tidy:
	GOCACHE=$(GOCACHE) go mod tidy

mod-check:
	GOCACHE=$(GOCACHE) go mod tidy -diff
	GOCACHE=$(GOCACHE) go mod verify

vet:
	GOCACHE=$(GOCACHE) go vet ./...

build:
	GOCACHE=$(GOCACHE) go build -ldflags="-s -w" -o bin/strategist ./cmd/strategist

test:
	GOCACHE=$(GOCACHE) go test -race $$(GOCACHE=$(GOCACHE) go list ./... | grep -v '/testutil')

test-all: test spec integration

integration:
	GOCACHE=$(GOCACHE) go test -race -tags=integration ./tests/integration/...

spec:
	GOCACHE=$(GOCACHE) go test -race -tags=spec ./tests/spec/...

validate-expanded:
	GOCACHE=$(GOCACHE) go test ./internal/telemetry ./internal/embed
	GOCACHE=$(GOCACHE) go test -tags=spec ./tests/spec/...
	GOCACHE=$(GOCACHE) go test -tags=integration ./tests/integration/...

validate-all:
	GOCACHE=$(GOCACHE) go test ./cmd/strategist
	$(MAKE) validate-expanded

# test-lite runs the isolated test slices that do not require downloading new modules.
test-lite: test-telemetry-lite test-compile-cache test-domain-architecture

# test-telemetry-lite runs the telemetry subset that only depends on stdlib + local code.
test-telemetry-lite:
	GOCACHE=$(GOCACHE) go test -race internal/telemetry/schema.go internal/telemetry/policy_event.go internal/telemetry/mission_run.go internal/telemetry/mission_metrics.go internal/telemetry/policy_event_test.go internal/telemetry/mission_run_test.go internal/telemetry/mission_metrics_test.go

# test-compile-cache runs the compile cache tests without the rest of the compile package suite.
test-compile-cache:
	GOCACHE=$(GOCACHE) go test -race internal/compile/cache.go internal/compile/cache_test.go

# test-domain-architecture runs the dependency-isolation smoke test without the rest of the domain suite.
test-domain-architecture:
	GOCACHE=$(GOCACHE) go test -race internal/domain/architecture_test.go

ci-lint: fmt-check mod-check vet build quality-budget-gate

ci-test: test-all convergence-check contract-consistency-gate cover-gate

ci: ci-lint ci-test

lint: fmt-check
	$(GOLANGCI_LINT) run ./...
	@$(MAKE) complexity-report
	@$(MAKE) go-file-size-report

# complexity-report lists files that contain functions with cognitive complexity > 7.
complexity-report:
	@$(MAKE) install-gocognit
	@echo "=== Cognitive Complexity > 7 ==="
	@$(GOCOGNIT) -over 7 ./cmd ./internal \
		| awk '{split($$NF,a,":"); print a[1]}' \
		| sort -u \
		| sed 's|$(CURDIR)/||' \
		|| true
	@echo ""
	@$(GOCOGNIT) -over 7 ./cmd ./internal \
		| sort -t' ' -k1 -rn \
		| sed 's|$(CURDIR)/||' \
		|| true

# go-file-size-report lists primary Go source files over 200 lines.
go-file-size-report:
	@echo "=== Go Files > 200 Lines ==="
	@files=$$(find cmd internal -type f -name '*.go' \
		! -name '*_test.go' \
		! -path 'internal/embed/defaults/*' | sort); \
	results=$$(for f in $$files; do \
		lines=$$(wc -l < "$$f" | tr -d ' '); \
		if [ "$$lines" -gt 200 ]; then printf "%s %s\n" "$$f" "$$lines"; fi; \
	done | sort -k2,2nr -k1,1); \
	if [ -n "$$results" ]; then printf "%s\n" "$$results"; else echo "none"; fi

install-gocognit:
	@command -v $(GOCOGNIT) >/dev/null 2>&1 || GOCACHE=$(GOCACHE) go install github.com/uudashr/gocognit/cmd/gocognit@$(GOCOGNIT_VERSION)

quality-budget-gate: install-gocognit
	bash scripts/check-quality-budgets.sh "$(QUALITY_BUDGETS)" "$(GOCOGNIT)" "$(COMPLEXITY_THRESHOLD)"

install-govulncheck:
	GOCACHE=$(GOCACHE) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

vuln:
	$(GOVULNCHECK) ./...

vuln-ci: install-govulncheck vuln

bench:
	GOCACHE=$(GOCACHE) go test -bench=. -benchmem ./...

# cover shows per-package coverage (each package measured against itself).
cover:
	@mkdir -p $(COVERAGE_DIR)
	@for pkg in $(COVERAGE_PKGS); do \
		echo "=== $$pkg ==="; \
		GOCACHE=$(GOCACHE) go test -race -coverprofile=$(COVERAGE_PROFILE) -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null; \
		go tool cover -func=$(COVERAGE_PROFILE) | tail -1; \
	done

# cover-gate fails the build when a package falls below its manifest threshold.
cover-gate:
	bash scripts/check-coverage-gate.sh "$(COVERAGE_MANIFEST)" "$(COVERAGE_DIR)" "$(GOCACHE)"

# cover-html writes an HTML coverage report without opening a browser.
cover-html:
	@mkdir -p $(COVERAGE_DIR)
	GOCACHE=$(GOCACHE) go test -race -coverprofile=$(COVERAGE_PROFILE) -coverpkg=./internal/... ./internal/... ./tests/integration/...
	go tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "report written to $(COVERAGE_HTML)"

analysis-structure-gate:
	bash scripts/check-refined-structure.sh

docs-governance-gate:
	bash scripts/check-docs-governance.sh

contract-consistency-gate:
	bash scripts/check-contract-consistency.sh

convergence-check:
	@echo "Checking runtime/package-boundary convergence..."
	@grep -q '"skills", mc.ExpectedProvider' internal/dojo/checker_manifest.go \
		|| (echo "DRIFT: dojo/checker_manifest.go uses old provider path (not skills/<provider>/skill.yaml)"; exit 1)
	@grep -q '"skills", "brainstorming"' internal/dojo/checker_manifest_test.go \
		|| (echo "DRIFT: dojo/checker_manifest_test.go uses old provider path"; exit 1)
	@grep -q 'skills/<provider>/skill.yaml' internal/domain/types.go \
		|| (echo "DRIFT: internal/domain/types.go lost the canonical provider path skills/<provider>/skill.yaml"; exit 1)
	@test ! -d strategist \
		|| (echo "DRIFT: strategist/ exists — the authoring mirror was retired (W7a); author in internal/embed/defaults/"; exit 1)
	@test -d internal/embed/defaults/internal_skills \
		|| (echo "DRIFT: internal/embed/defaults/internal_skills/ missing — authoring tree broken"; exit 1)
	@echo "Convergence check: OK"

governance-check:
	@echo "Checking governance redirectors..."
	@for f in CLAUDE.md AGENTS.md GEMINI.md; do \
		grep -q "Governance fingerprint:" "$$f" || (echo "DRIFT: $$f missing governance fingerprint header"; exit 1); \
		grep -q "agent-instructions.md" "$$f" || (echo "DRIFT: $$f missing .sdd/agent-instructions.md reference"; exit 1); \
	done
	@echo "Governance redirectors: OK"

validate-fixtures:
	python3 -c "import yaml" || python3 -m pip install --user pyyaml
	bash tests/spec/run-tests.sh

install: build
	mkdir -p ~/.local/bin
	install -m 755 bin/strategist ~/.local/bin/strategist
	@echo "[Strategist] binary installed. Run: strategist install --wizard"

# The sync-embed target was removed in W7a (Option B): internal/embed/defaults/ is now
# the single authoring source embedded directly via go:embed — there is nothing to sync.

release-verify: ci-lint ci-test docs-governance-gate validate-fixtures release-reproducible-check

# release-check validates the GoReleaser config before a tag-triggered release.
release-check:
	$(GORELEASER) check

install-goreleaser:
	GOCACHE=$(GOCACHE) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

check-release-artifacts:
	bash scripts/check-release-artifacts.sh

check-release-assets:
	bash scripts/check-release-assets.sh "$(TAG)" dist/published.tsv

release-reproducible-check:
	bash scripts/check-reproducible-build.sh "$(GOCACHE)"

# release-test validates release config and local snapshot artifacts without publishing.
release-test: release-check snapshot check-release-artifacts

release-dry-run: install-goreleaser release-test

# release publishes to GitHub — requires GITHUB_TOKEN.
release:
	$(GORELEASER) release --clean

# snapshot builds release artifacts locally without publishing (no token needed).
snapshot:
	$(GORELEASER) release --snapshot --clean --skip=publish

clean:
	rm -rf bin/ dist/ coverage/ coverage.out coverage.html cover.out coverage_cmd.out

# compile-skill regenerates the compiled bootstrap artifacts in .strategist/.compiled/.
# Run after editing any file under .strategist/ to keep the fast-path active.
compile-skill:
	strategist compile --root .strategist

install-web:
	cd web/landing && npm ci

build-site: install-web
	cd web/landing && npm run build

build-all: build build-site

lint-web:
	cd web/landing && npm run lint

test-web:
	cd web/landing && npm run test

cover-web:
	cd web/landing && npm run cover

ci-web: install-web lint-web test-web build-site
