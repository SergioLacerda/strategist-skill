.PHONY: build test test-all integration spec validate-expanded validate-all test-lite test-telemetry-lite test-compile-cache test-domain-architecture lint complexity-report vuln bench cover cover-gate cover-html analysis-structure-gate docs-governance-gate governance-check convergence-check install sync-embed release snapshot clean compile-skill build-site build-all install-web lint-web test-web cover-web

GOCACHE ?= /tmp/go-build-cache

GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
GOVULNCHECK   := $(shell which govulncheck 2>/dev/null || echo $(shell go env GOPATH)/bin/govulncheck)
GOCOGNIT      := $(shell which gocognit 2>/dev/null || echo $(shell go env GOPATH)/bin/gocognit)
GORELEASER    := $(shell which goreleaser 2>/dev/null || echo $(shell go env GOPATH)/bin/goreleaser)
COVERAGE_PKGS := internal/stale internal/compile internal/install internal/embed internal/telemetry cmd/strategist

build:
	go build -ldflags="-s -w" -o bin/strategist ./cmd/strategist

test:
	GOCACHE=$(GOCACHE) go test -race $$(go list ./... | grep -v '/testutil')

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

lint:
	gofmt -w .
	$(GOLANGCI_LINT) run ./...
	@$(MAKE) complexity-report

# complexity-report lists files that contain functions with cognitive complexity > 7.
# Informational only — does not fail the build.
complexity-report:
	@command -v $(GOCOGNIT) >/dev/null 2>&1 || go install github.com/uudashr/gocognit/cmd/gocognit@latest
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

vuln:
	$(GOVULNCHECK) ./...

bench:
	go test -bench=. -benchmem ./...

# cover shows per-package coverage (each package measured against itself).
cover:
	@for pkg in $(COVERAGE_PKGS); do \
		echo "=== $$pkg ==="; \
		go test -race -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null; \
		go tool cover -func=coverage.out | tail -1; \
	done

# cover-gate fails the build if any internal package is below 90%.
# Note: internal/domain is excluded (pure type declarations — no executable statements).
cover-gate:
	@fail=0; \
	for pkg in $(COVERAGE_PKGS); do \
		pct=$$(go test -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null \
			| grep -o '[0-9.]*%' | tail -1 | tr -d '%'); \
		printf "%-30s %s%%\n" "$$pkg" "$$pct"; \
		ok=$$(awk -v p="$$pct" 'BEGIN{print (p+0 >= 90)}'); \
		if [ "$$ok" != "1" ]; then echo "  FAIL: $$pct% < 90%"; fail=1; fi; \
	done; \
	exit $$fail

# cover-html writes coverage.html without opening a browser.
cover-html:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./tests/integration/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "report written to coverage.html"

analysis-structure-gate:
	bash scripts/check-refined-structure.sh

docs-governance-gate:
	bash scripts/check-docs-governance.sh

convergence-check:
	@echo "Checking runtime/package-boundary convergence..."
	@grep -q 'skills.*ExpectedProvider\|"skills".*ExpectedProvider' internal/dojo/checker.go \
		|| (echo "DRIFT: dojo/checker.go uses old provider path (not skills/<provider>/skill.yaml)"; exit 1)
	@grep -q '"skills".*"brainstorming"\|skills.*brainstorming' internal/dojo/checker_test.go \
		|| (echo "DRIFT: dojo/checker_test.go uses old provider path"; exit 1)
	@grep -q '"skills".*providerID\|skills.*providerID' cmd/strategist/initiative.go \
		|| (echo "DRIFT: initiative.go uses old provider path (not skills/<provider>/skill.yaml)"; exit 1)
	@echo "Convergence check: OK"

governance-check:
	@echo "Checking governance redirectors..."
	@for f in CLAUDE.md AGENTS.md GEMINI.md; do \
		grep -q "Governance fingerprint:" "$$f" || (echo "DRIFT: $$f missing governance fingerprint header"; exit 1); \
		grep -q "agent-instructions.md" "$$f" || (echo "DRIFT: $$f missing .sdd/agent-instructions.md reference"; exit 1); \
	done
	@echo "Governance redirectors: OK"

install: build
	mkdir -p ~/.local/bin
	install -m 755 bin/strategist ~/.local/bin/strategist
	@echo "[Strategist] binary installed. Run: strategist install --wizard"

# sync-embed copies updated YAML artifacts from strategist/ to internal/embed/defaults/
# so the next `make build` embeds the latest versions.
# Run this whenever you edit files under strategist/ that should ship in the binary.
# After sync-embed, run: make build && ./bin/strategist install --target <project>
sync-embed:
	@echo "[Strategist] syncing strategist/ → internal/embed/defaults/"
	rsync -a --delete \
		--exclude 'active.schema.yaml' \
		--exclude 'mission-result.schema.yaml' \
		--exclude 'roles.schema.yaml' \
		--exclude 'slot-output.schema.yaml' \
		strategist/schemas/ internal/embed/defaults/schemas/
	rsync -a --delete \
		--exclude 'default.yaml' \
		strategist/roles/ internal/embed/defaults/roles/
	rsync -a --delete strategist/templates/ internal/embed/defaults/templates/
	rsync -a --delete strategist/personas/ internal/embed/defaults/personas/
	@if [ -d strategist/output-profiles ]; then rsync -a --delete strategist/output-profiles/ internal/embed/defaults/output-profiles/; fi
	rsync -a --delete strategist/internal_skills/ internal/embed/defaults/internal_skills/
	rsync -a --delete strategist/contracts/narrative/ internal/embed/defaults/contracts/narrative/
	rsync -a --delete strategist/contracts/machine/   internal/embed/defaults/contracts/machine/
	rsync -a          strategist/contracts/index.yaml  internal/embed/defaults/contracts/index.yaml
	rsync -a --delete strategist/SKILL.md internal/embed/defaults/SKILL.md
	rsync -a --delete strategist/protocol.md internal/embed/defaults/protocol.md
	rsync -a --delete strategist/skill.yaml internal/embed/defaults/skill.yaml
	rsync -a --delete strategist/treasure-chests.yaml internal/embed/defaults/treasure-chests.yaml
	@echo "[Strategist] sync done. Next: make build && ./bin/strategist install --target <project>"

# release publishes to GitHub — requires GITHUB_TOKEN.
release:
	$(GORELEASER) release --clean

# snapshot builds release artifacts locally without publishing (no token needed).
snapshot:
	$(GORELEASER) release --snapshot --clean --skip=publish

clean:
	rm -rf bin/ dist/ coverage.out coverage.html

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
