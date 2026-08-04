.PHONY: ci-lint ci-test ci

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

include make/go.mk
include make/quality.mk
include make/governance.mk
include make/release.mk
include make/web.mk

ci-lint: fmt-check mod-check vet build quality-budget-gate

ci-test: test-all convergence-check contract-consistency-gate cover-gate

ci: ci-lint ci-test
