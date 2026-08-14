.PHONY: \
	fmt fmt-check mod-tidy mod-check vet build \
	test test-all integration spec eval eval-promptfoo golden \
	validate-expanded validate-all \
	test-lite test-telemetry-lite test-compile-cache test-domain-architecture \
	bench validate-fixtures

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

eval:
	GOCACHE=$(GOCACHE) go test -race -tags=eval ./tests/evals/...

# golden runs the deterministic artifact snapshot suite. Opt-in, not part of
# test/test-all/ci-test/ci — the cli-help subtest alone costs ~90s (a cold
# `go run ./cmd/strategist --help` per invocation). See
# docs/adr/0026-deterministic-golden-testing.md. Use `-run <pattern>` or
# `-update` (never in CI — see tests/evals/golden/golden.go) as needed.
golden:
	GOCACHE=$(GOCACHE) go test -race -tags=golden ./tests/evals/golden/...

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

bench:
	GOCACHE=$(GOCACHE) go test -bench=. -benchmem ./...

validate-fixtures:
	python3 -c "import yaml" || python3 -m pip install --user pyyaml
	bash tests/spec/run-tests.sh
