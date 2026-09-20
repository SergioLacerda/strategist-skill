.PHONY: \
	install release-verify release-check install-goreleaser \
	check-release-artifacts check-release-assets release-reproducible-check \
	release-test release-dry-run release snapshot clean compile-skill \
	embed-skills embed-skills-check build-standalone standalone-smoke install-lite

# install puts a STANDALONE binary in ~/.local/bin: it embeds the private runtime
# of the Ranked provider (openspec-propose), so `strategist install --wizard`
# works on a machine with no OpenSpec or Node. It fetches the pinned Node by
# digest on the first run (needs network and python; cached afterwards).
# Use install-lite for a binary without the runtime (resolves openspec from PATH).
install: build-standalone
	mkdir -p "$$HOME/.local/bin" && install -m 755 bin/strategist "$$HOME/.local/bin/strategist"
	@# Prove the binary now on disk is the standalone one; a stale copy (or another
	@# strategist earlier on PATH) would otherwise fail later with a misleading error.
	@"$$HOME/.local/bin/strategist" version --build | grep -q "runtime payload: embedded" || { echo "[Strategist] ERROR: the installed binary has no embedded runtime; check 'command -v strategist' and 'strategist version --build'" >&2; exit 1; }
	@"$$HOME/.local/bin/strategist" version --build
	@command -v strategist >/dev/null 2>&1 && [ "$$(command -v strategist)" != "$$HOME/.local/bin/strategist" ] && echo "[Strategist] WARNING: 'strategist' on PATH is $$(command -v strategist), not $$HOME/.local/bin/strategist" >&2 || true
	@echo "[Strategist] standalone binary installed. Run: strategist install --wizard"

install-lite: build
	mkdir -p "$$HOME/.local/bin" && install -m 755 bin/strategist "$$HOME/.local/bin/strategist"
	@echo "[Strategist] binary installed WITHOUT the embedded runtime (needs openspec on PATH for the Ranked provider)."

# The sync-embed target was removed in W7a (Option B): internal/embed/defaults/ is now
# the single authoring source embedded directly via go:embed — there is nothing to sync.

# embed-skills ingests external-skills-source/ into the embedded plugin
# catalog (ADR-0032's pre-build ingestion pattern, generalized to role-slot
# skills — see .analysis/done/20260913-embedded-skill-directory-catalog).
# Run after adding/editing anything under external-skills-source/, then
# commit the regenerated catalog.yaml, skill.yaml mirrors, and lock file.
embed-skills: build
	./bin/strategist plugins prepare-embedded

# embed-skills-check fails non-zero on drift instead of writing — the CI gate
# that catches an external-skills-source/ change that was never followed by
# `make embed-skills`.
embed-skills-check: build
	./bin/strategist plugins prepare-embedded --check

release-verify: ci-lint ci-test docs-governance-gate validate-fixtures vuln-ci release-reproducible-check embed-skills-check

# release-check validates the GoReleaser config before a tag-triggered release.
release-check:
	$(GORELEASER) check

install-goreleaser:
	GOCACHE=$(GOCACHE) go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

check-release-artifacts:
	bash scripts/check-release-artifacts.sh

check-release-assets:
	bash scripts/check-release-assets.sh "$(TAG)" dist/published.tsv

# Also covers the standalone build that embeds the runtime payload (fetches the
# pinned Node for the host by digest, so it needs network).
release-reproducible-check:
	REPRODUCIBLE_PAYLOAD=1 bash scripts/check-reproducible-build.sh "$(GOCACHE)"

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

# build-standalone builds bin/strategist with the private runtime payload for
# the host target embedded (fetches the pinned Node first; needs network unless
# .cache/node-runtime is pre-seeded). Ordinary `make build` never embeds it.
PYTHON ?= $(shell command -v python3 2>/dev/null || command -v python 2>/dev/null)

build-standalone:
	@test -n "$(PYTHON)" || { echo "python3 (or python) is required to fetch the pinned Node; use 'make install-lite' to skip the embedded runtime" >&2; exit 1; }
	$(PYTHON) scripts/fetch-node-runtime.py --host
	GOCACHE=$(GOCACHE) CGO_ENABLED=0 go build -tags strategist_payload -trimpath -ldflags="-s -w -X main.Version=$$(git describe --tags --dirty --always 2>/dev/null || echo dev)" -o bin/strategist ./cmd/strategist

# standalone-smoke proves a payload build installs and passes check with an
# empty PATH (no host openspec or node).
standalone-smoke:
	./scripts/smoke-standalone-install.sh
