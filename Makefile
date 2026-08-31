.PHONY: ci-lint ci-test ci

# Fail fast with one clear, actionable message when make cannot reach a POSIX
# shell — e.g. invoked directly from PowerShell/cmd.exe on Windows instead of
# Git Bash/WSL. Without this, the variable/recipe lines below (which/awk/test)
# fail one at a time with opaque `CreateProcess(...) failed` errors and no
# indication of the actual cause. See CONTRIBUTING.md § Prerequisites and
# .analysis/refined/20260811-windows-make/design.md. The first CreateProcess
# failure line (if any) still comes from `make` itself and cannot be
# suppressed — this guard adds the explanation right after it.
ifneq ($(shell test -z "" && echo posix-shell-ok),posix-shell-ok)
$(error make requires a POSIX shell. On Windows, run make from Git Bash or WSL -- never directly from PowerShell or cmd.exe. See CONTRIBUTING.md, section Prerequisites)
endif

GOCACHE ?= /tmp/go-build-cache

# `go env GOPATH` prints a backslash-separated path on Windows (e.g.
# C:\Users\User\go). Make substitutes that text literally into the recipe
# lines below *before* the shell parses them, so an unquoted backslash there
# is stripped by the shell's own escape handling (C:\Users\User\go becomes
# C:UsersUsergo) -- not a live command-substitution result, but literal
# source text being re-parsed. tr normalizes it to forward slashes once,
# here, so nothing downstream ever sees a backslash to mis-parse. Octal \134
# is the portable spelling of a literal backslash across tr implementations.
GOPATH_BIN          := $(shell go env GOPATH | tr '\134' '/')/bin

GOLANGCI_LINT       := $(shell which golangci-lint 2>/dev/null || echo $(GOPATH_BIN)/golangci-lint)
GOVULNCHECK         := $(shell which govulncheck 2>/dev/null || echo $(GOPATH_BIN)/govulncheck)
GOCOGNIT            := $(shell which gocognit 2>/dev/null || echo $(GOPATH_BIN)/gocognit)
GORELEASER          := $(shell which goreleaser 2>/dev/null || echo $(GOPATH_BIN)/goreleaser)
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
include make/docs.mk

ci-lint: fmt-check mod-check vet build quality-budget-gate

ci-test: test-all golden convergence-check contract-consistency-gate cover-gate docs-generated-gate

ci: ci-lint ci-test
