.PHONY: install-actionlint install-zizmor actionlint shellcheck zizmor workflow-lint ci-script-test

# Static analysis of the CI itself: workflow syntax (actionlint), workflow
# security (zizmor), shell scripts (shellcheck) and release-tool pin drift.
# Kept out of ci-lint on purpose: these tools come from outside the module
# graph, so they run as their own CI job until they have a stable track record
# (docs/adr/0052-cicd-enforcement-policy.md, decision 6).
ACTIONLINT_VERSION ?= v1.7.12
ZIZMOR_VERSION     ?= 1.30.1
SHELLCHECK_PY      ?= 0.11.0.1
ACTIONLINT         := $(shell which actionlint 2>/dev/null || echo $(GOPATH_BIN)/actionlint)
SHELL_SCRIPTS      := $(wildcard scripts/*.sh scripts/hooks/*) bootstrap.sh .github/setup-precommit-hook.sh

install-actionlint:
	GOCACHE="$(GOCACHE)" go install github.com/rhysd/actionlint/cmd/actionlint@$(ACTIONLINT_VERSION)

# CI installs zizmor with pipx (preinstalled on the runner image; a plain
# `pip install --user` is refused there by PEP 668).
install-zizmor:
	pipx install "zizmor==$(ZIZMOR_VERSION)"

actionlint:
	@test -x "$(ACTIONLINT)" || { echo "actionlint not found: run 'make install-actionlint'" >&2; exit 1; }
	"$(ACTIONLINT)" .github/workflows/*.yml

# shellcheck: warning severity and above. The runner image ships shellcheck;
# locally: pip install shellcheck-py==$(SHELLCHECK_PY)
shellcheck:
	@command -v shellcheck >/dev/null 2>&1 || { echo "shellcheck not found (pip install shellcheck-py==$(SHELLCHECK_PY))" >&2; exit 1; }
	shellcheck -S warning $(SHELL_SCRIPTS)

# zizmor runs offline unless GH_TOKEN is set (CI sets it for the online audits).
# locally: pip install zizmor==$(ZIZMOR_VERSION)
zizmor:
	@command -v zizmor >/dev/null 2>&1 || { echo "zizmor not found (pip install zizmor==$(ZIZMOR_VERSION))" >&2; exit 1; }
	zizmor .github/workflows

workflow-lint: actionlint shellcheck zizmor
	bash scripts/check-goreleaser-pin.sh
	bash scripts/test-check-goreleaser-pin.sh

# ci-script-test exercises the CI reporting scripts against fixtures and a
# stubbed gh (no network).
ci-script-test:
	bash scripts/test-report-job-durations.sh
