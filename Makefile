.PHONY: build test lint vuln bench cover cover-gate cover-html install-local release snapshot clean

GOLANGCI_LINT := $(shell which golangci-lint 2>/dev/null || echo $(shell go env GOPATH)/bin/golangci-lint)
GOVULNCHECK   := $(shell which govulncheck 2>/dev/null || echo $(shell go env GOPATH)/bin/govulncheck)
GORELEASER    := $(shell which goreleaser 2>/dev/null || echo $(shell go env GOPATH)/bin/goreleaser)

build:
	go build -ldflags="-s -w" -o bin/strategist ./cmd/strategist

test:
	go test -race $$(go list ./... | grep -v '/testutil')

lint:
	$(GOLANGCI_LINT) run ./...

vuln:
	$(GOVULNCHECK) ./...

bench:
	go test -bench=. -benchmem ./...

# cover shows per-package coverage (each package measured against itself).
cover:
	@for pkg in internal/stale internal/compile internal/install internal/embed; do \
		echo "=== $$pkg ==="; \
		go test -race -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null; \
		go tool cover -func=coverage.out | tail -1; \
	done

# cover-gate fails the build if any internal package is below 90%.
# Note: internal/domain is excluded (pure type declarations — no executable statements).
cover-gate:
	@fail=0; \
	for pkg in internal/stale internal/compile internal/install internal/embed internal/telemetry cmd/strategist; do \
		pct=$$(go test -coverprofile=coverage.out -coverpkg=./$$pkg/... ./$$pkg/... 2>/dev/null \
			| grep -o '[0-9.]*%' | tail -1 | tr -d '%'); \
		printf "%-30s %s%%\n" "$$pkg" "$$pct"; \
		ok=$$(awk -v p="$$pct" 'BEGIN{print (p+0 >= 90)}'); \
		if [ "$$ok" != "1" ]; then echo "  FAIL: $$pct% < 90%"; fail=1; fi; \
	done; \
	exit $$fail

# cover-html writes coverage.html without opening a browser.
cover-html:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./tests/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "report written to coverage.html"

install-local: build
	install -m 755 bin/strategist ~/.local/bin/strategist

# release publishes to GitHub — requires GITHUB_TOKEN.
release:
	$(GORELEASER) release --clean

# snapshot builds release artifacts locally without publishing (no token needed).
snapshot:
	$(GORELEASER) release --snapshot --clean --skip=publish

clean:
	rm -rf bin/ dist/ coverage.out
