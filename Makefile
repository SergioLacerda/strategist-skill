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
# Keep golangci-lint's analysis cache alongside the isolated Go build cache.
# This avoids reusing entries written by a different system/toolchain version
# (and keeps local CI-like runs writable in restricted environments).
GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache

# `go env GOPATH` prints a backslash-separated path on Windows (e.g.
# C:\Users\User\go). Make substitutes that text literally into the recipe
# lines below *before* the shell parses them, so an unquoted backslash there
# is stripped by the shell's own escape handling (C:\Users\User\go becomes
# C:UsersUsergo) -- not a live command-substitution result, but literal
# source text being re-parsed. tr normalizes it to forward slashes once,
# here, so nothing downstream ever sees a backslash to mis-parse. Octal \134
# is the portable spelling of a literal backslash across tr implementations.
GOPATH_BIN          := $(shell go env GOPATH | tr '\134' '/')/bin
LOCAL_BIN           := $(CURDIR)/bin

# Executable suffix of the build host (".exe" on Windows, empty elsewhere). Every
# target that names the strategist binary uses it, so a Windows build produces
# strategist.exe, which cmd and PowerShell can resolve. Override with EXE=.
EXE                 ?= $(shell go env GOEXE)
STRATEGIST_BIN      := bin/strategist$(EXE)

ifneq ($(wildcard $(LOCAL_BIN)/golangci-lint),)
GOLANGCI_LINT       := $(LOCAL_BIN)/golangci-lint
else
GOLANGCI_LINT       := $(shell which golangci-lint 2>/dev/null || echo $(GOPATH_BIN)/golangci-lint)
endif

# Pinned to go.mod's own `toolchain` line so AST/SSA-based tools (golangci-lint's
# bundled go/types checker, govulncheck's x/tools SSA builder) always analyze
# against the exact Go version they were built for. GOTOOLCHAIN=auto only
# upgrades when the system `go` is OLDER than this pin -- a system `go` that
# races ahead of it (e.g. a distro shipping a very recent/prerelease point
# release) is used as-is otherwise, which can make these tools panic on AST
# shapes they weren't built to understand (observed: golangci-lint failing to
# type-check stdlib packages; govulncheck's ssa builder panicking with
# "unexpected expr: *ast.KeyValueExpr"). An explicit, non-"auto" GOTOOLCHAIN
# value always switches to (downloading if needed) exactly that version, in
# either direction, sidestepping the skew regardless of what the system `go` is.
PINNED_GOTOOLCHAIN  := $(shell awk '/^toolchain /{print $$2}' go.mod)
GOVULNCHECK         := $(shell which govulncheck 2>/dev/null || echo $(GOPATH_BIN)/govulncheck)
GOCOGNIT            := $(shell which gocognit 2>/dev/null || echo $(GOPATH_BIN)/gocognit)
GORELEASER          := $(shell which goreleaser 2>/dev/null || echo $(GOPATH_BIN)/goreleaser)
GOVULNCHECK_VERSION ?= v1.8.0
GOCOGNIT_VERSION    ?= v1.2.1
GORELEASER_VERSION  ?= v2.12.2
COVERAGE_MANIFEST   := scripts/coverage-packages.tsv
COVERAGE_EXEMPTIONS := scripts/coverage-exemptions.tsv
COVERAGE_PKGS       := $(shell awk 'NF && $$1 !~ /^#/ {print $$1}' $(COVERAGE_MANIFEST))
COVERAGE_DIR        ?= coverage
COVERAGE_PROFILE    := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML       := $(COVERAGE_DIR)/coverage.html
QUALITY_BUDGETS     := scripts/quality-budgets.tsv
COMPLEXITY_THRESHOLD ?= 7

include make/go.mk
include make/quality.mk
include make/governance.mk
include make/release.mk
include make/web.mk
include make/docs.mk

ci-lint: lint-status fmt-check mod-check vet build quality-budget-gate

ci-test: test-all golden convergence-check contract-consistency-gate coverage-manifest-check cover-gate docs-generated-gate docs-links-gate docs-index-ownership-gate mutation-role-weapon

ci: ci-lint ci-test
