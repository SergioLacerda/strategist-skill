.PHONY: \
	fmt fmt-check mod-tidy mod-check vet build \
	test test-all integration spec eval golden leveling-integration \
	validate-expanded validate-all \
	test-lite test-telemetry-lite test-compile-cache test-domain-architecture \
	bench validate-fixtures treasure-chest-tmp

# External Weapon source and versioned embedded payloads may contain source
# files for the provider's own module. They are artifacts, not packages of the
# Strategist host module; keep host-owned Go gates from treating them as local
# dependencies. The generated list still covers every tracked/untracked Go
# package under the host-owned roots.
HOST_GO_DIRS := $(shell find cmd internal treasure-chest -type f -name '*.go' ! -path 'internal/embed/defaults/*' -exec dirname {} \; | sort -u)
HOST_GO_PACKAGES := $(addprefix ./,$(HOST_GO_DIRS))
TREASURE_CHEST_TMP_BIN := bin/treasure-chest-tmp$(EXE)

fmt:
	git ls-files -co --exclude-standard -z '*.go' | xargs -0r -n1 sh -c 'if test -f "$$1"; then gofmt -w "$$1"; fi' sh

fmt-check:
	@files="$$(git ls-files -co --exclude-standard -z '*.go' | xargs -0r -n1 sh -c 'if test -f "$$1"; then gofmt -l "$$1"; fi' sh)"; \
	if [ -n "$$files" ]; then \
		echo "fmt-check: unformatted Go files detected:" >&2; \
		printf '%s\n' "$$files" | sed 's/^/  - /' >&2; \
		echo "fmt-check: run 'make fmt' to apply formatting" >&2; \
		exit 1; \
	else \
		echo "fmt-check: all Go files are formatted"; \
	fi

mod-tidy:
	GOCACHE="$(GOCACHE)" go mod tidy

mod-check:
	GOCACHE="$(GOCACHE)" go mod tidy -diff
	GOCACHE="$(GOCACHE)" go mod verify

vet:
	GOCACHE="$(GOCACHE)" go vet $(HOST_GO_PACKAGES)

build: generate-embedded
	GOCACHE="$(GOCACHE)" go build -ldflags="-s -w -X main.Version=$$(git describe --tags --dirty --always 2>/dev/null || echo dev)" -o "$(STRATEGIST_BIN)" ./cmd/strategist
	GOCACHE="$(GOCACHE)" go run ./cmd/strategist plugins prepare-embedded --check
	"$(STRATEGIST_BIN)" version --build

treasure-chest-tmp:
	GOCACHE="$(GOCACHE)" go build -o "$(TREASURE_CHEST_TMP_BIN)" ./cmd/treasure-chest-tmp
	@echo "[Strategist] temporary Weapon built: $(TREASURE_CHEST_TMP_BIN)"

test:
	GOCACHE="$(GOCACHE)" go test -race $(HOST_GO_PACKAGES)

test-all: test spec integration

# leveling-integration is a deterministic, provider-free matrix for the
# LEVELING budget and its first Ranger consumer. GOPROXY=off makes missing
# dependencies fail instead of reaching the network; GOTOOLCHAIN=auto permits
# the repository's required Go patch release when it is already installed.
LEVELING_GOCACHE ?= /tmp/strategist-leveling-integration-cache
leveling-integration:
	GOCACHE="$(LEVELING_GOCACHE)" GOPROXY=off GOTOOLCHAIN=auto go test -count=1 ./internal/tools/leveling ./internal/application/leveling ./internal/feats/initiative ./internal/telemetry ./internal/domain ./cmd/strategist/leveling
	GOCACHE="$(LEVELING_GOCACHE)" GOPROXY=off GOTOOLCHAIN=auto go test -count=1 ./cmd/strategist -run 'Test(RangerPrecheck|EmitRoleLevel|ResolveMissionLeveling|StartInitiativeConsultation)'

integration:
	GOCACHE="$(GOCACHE)" go test -race -tags=integration ./tests/integration/...

spec:
	GOCACHE="$(GOCACHE)" go test -race -tags=spec ./tests/spec/...

eval:
	GOCACHE="$(GOCACHE)" go test -race -tags=eval ./tests/evals/...

# ---------------------------------------------------------------------------
# External-call checks (LIVE). Deliberately NOT in .PHONY, in no dependency chain
# and not reachable from test, test-all, ci, ci-test or build: nothing a plain
# `make` target does ever calls an external endpoint.
#
# The sanctioned way to run a check against the real provider is the DOJO, which
# prints a consent term (endpoint, data class, calls, bytes, estimated usage) and
# needs an explicit --consent:
#     strategist dojo calibrate jev-live --dry-run     # then --consent
#     strategist dojo calibrate jev-pilot --dry-run    # then --consent
# The two recipes below remain for developers who want the Go-level live tests;
# they refuse to run unless STRATEGIST_LIVE_CONSENT=yes is set explicitly.
# ---------------------------------------------------------------------------
live-jev:
	@test "$(STRATEGIST_LIVE_CONSENT)" = "yes" || { echo "refused: this calls the real provider; prefer 'strategist dojo calibrate jev-live --dry-run', or set STRATEGIST_LIVE_CONSENT=yes"; exit 1; }
	STRATEGIST_LIVE_CONSENT=yes GOCACHE="$(GOCACHE)" go test -tags=live -count=1 -v ./internal/integration/live/...

live-calibration:
	@test "$(STRATEGIST_LIVE_CONSENT)" = "yes" || { echo "refused: this calls the real provider; prefer 'strategist dojo calibrate jev-pilot --dry-run', or set STRATEGIST_LIVE_CONSENT=yes"; exit 1; }
	STRATEGIST_LIVE_CONSENT=yes GOCACHE="$(GOCACHE)" go test -tags=live -count=1 -v -timeout 30m -run TestLiveCalibrationPilot ./internal/integration/live/...

# golden runs the deterministic artifact snapshot suite. Wired into ci-test
# (see the top-level Makefile) per docs/adr/0026-deterministic-golden-testing.md's
# CI-wiring follow-up and .analysis/refined/20260830-skill-gaps-triage/tasks.md
# Task 6 — the cli-help subtest alone costs ~90s (a cold
# `go run ./cmd/strategist --help` per invocation), which fits comfortably inside
# the `test` job's 15-minute timeout. Use `-run <pattern>` or `-update` (never in
# CI — see tests/evals/golden/golden.go) for local iteration.
golden:
	GOCACHE="$(GOCACHE)" go test -race -tags=golden ./tests/evals/golden/...

# eval-promptfoo runs the Promptfoo-based artifact quality review config. Standalone and
# manual — not wired into eval/test/test-all/ci-test/ci (see DEC-2 in
# .analysis/archived/20260804-promptfoo-ci-adapter-adr.md). Requires a local LM Studio
# endpoint; operator-verify apiBaseUrl in promptfoo/promptfooconfig.yaml first. Guarded by
# a reachability preflight (see .analysis/refined/20260804-eval-promptfoo-guard/design.md)
# so a missing local server fails fast with a clear message instead of a raw fetch error.
PROMPTFOO_LM_STUDIO_URL ?= http://localhost:1234/v1

eval-promptfoo:
	@curl -sf -m 3 "$(PROMPTFOO_LM_STUDIO_URL)/models" >/dev/null || \
	  (echo "eval-promptfoo: no LM Studio (or compatible) server detected at $(PROMPTFOO_LM_STUDIO_URL) - start it, or override PROMPTFOO_LM_STUDIO_URL, then rerun." >&2 && exit 1)
	cd promptfoo && npx promptfoo eval

validate-expanded:
	GOCACHE="$(GOCACHE)" go test ./internal/telemetry ./internal/embed
	GOCACHE="$(GOCACHE)" go test -tags=spec ./tests/spec/...
	GOCACHE="$(GOCACHE)" go test -tags=integration ./tests/integration/...

validate-all:
	GOCACHE="$(GOCACHE)" go test ./cmd/strategist
	$(MAKE) validate-expanded

# test-lite runs the isolated test slices that do not require downloading new modules.
test-lite: test-telemetry-lite test-compile-cache test-domain-architecture

# test-telemetry-lite runs the telemetry subset that only depends on stdlib + local code.
test-telemetry-lite:
	GOCACHE="$(GOCACHE)" go test -race internal/telemetry/schema.go internal/telemetry/policy_event.go internal/telemetry/mission_run.go internal/telemetry/mission_run_marks.go internal/telemetry/mission_metrics.go internal/telemetry/mission_token_usage.go internal/telemetry/jsonl.go internal/telemetry/policy_event_test.go internal/telemetry/mission_run_test.go internal/telemetry/mission_metrics_test.go

# test-compile-cache runs the compile cache tests without the rest of the compile package suite.
test-compile-cache:
	GOCACHE="$(GOCACHE)" go test -race internal/compile/cache.go internal/compile/cache_test.go

# test-domain-architecture runs the dependency-isolation smoke test without the rest of the domain suite.
test-domain-architecture:
	GOCACHE="$(GOCACHE)" go test -race internal/domain/architecture_test.go

bench:
	GOCACHE="$(GOCACHE)" go test -bench=. -benchmem $(HOST_GO_PACKAGES)

validate-fixtures:
	python3 -c "import yaml" || python3 -m pip install --user pyyaml
	bash tests/spec/run-tests.sh
