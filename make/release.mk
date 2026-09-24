.PHONY: \
	install release-verify release-check install-goreleaser \
	check-release-artifacts check-release-assets release-reproducible-check \
	release-test release-dry-run release snapshot clean compile-skill \
	release-tag-check release-tag-test release-script-test check-release-binaries verify-published-release \
	embed-skills embed-skills-check build-standalone standalone-smoke install-lite doctor install-hooks

# install puts a standalone binary in ~/.local/bin. It embeds the OpenSpec
# bundle; Ranked execution uses the client's validated host Node >=20.19.0.
# Use install-lite for a binary without the runtime (resolves openspec from PATH).
install: build-standalone
	mkdir -p "$$HOME/.local/bin" && install -m 755 "$(STRATEGIST_BIN)" "$$HOME/.local/bin/strategist$(EXE)"
	@# Prove the binary now on disk is the standalone one; a stale copy (or another
	@# strategist earlier on PATH) would otherwise fail later with a misleading error.
	@"$$HOME/.local/bin/strategist$(EXE)" version --build | grep -q "runtime: embedded OpenSpec" || { echo "[Strategist] ERROR: the installed binary has no embedded OpenSpec bundle; check 'command -v strategist' and 'strategist version --build'" >&2; exit 1; }
	@"$$HOME/.local/bin/strategist$(EXE)" version --build
	@command -v strategist >/dev/null 2>&1 && [ "$$(command -v strategist)" != "$$HOME/.local/bin/strategist$(EXE)" ] && echo "[Strategist] WARNING: 'strategist' on PATH is $$(command -v strategist), not $$HOME/.local/bin/strategist$(EXE)" >&2 || true
	@echo "[Strategist] standalone binary installed. Run: strategist install --wizard"

install-lite: build
	mkdir -p "$$HOME/.local/bin" && install -m 755 "$(STRATEGIST_BIN)" "$$HOME/.local/bin/strategist$(EXE)"
	@echo "[Strategist] binary installed WITHOUT the embedded runtime (needs openspec on PATH for the Ranked provider)."

# The sync-embed target was removed in W7a (Option B): internal/embed/defaults/ is now
# the single authoring source embedded directly via go:embed — there is nothing to sync.

# embed-skills ingests external-skills-source/ into the embedded plugin
# catalog (ADR-0032's pre-build ingestion pattern, generalized to role-slot
# skills — see .analysis/done/20260913-embedded-skill-directory-catalog).
# Run after adding/editing anything under external-skills-source/, then
# commit the regenerated catalog.yaml, skill.yaml mirrors, and lock file.
embed-skills: build
	"./$(STRATEGIST_BIN)" plugins prepare-embedded

# embed-skills-check fails non-zero on drift instead of writing — the CI gate
# that catches an external-skills-source/ change that was never followed by
# `make embed-skills`.
embed-skills-check: build
	"./$(STRATEGIST_BIN)" plugins prepare-embedded --check

release-verify: ci-lint ci-test docs-governance-gate validate-fixtures vuln-ci release-reproducible-check embed-skills-check

# release-tag-check fails unless TAG is an annotated vX.Y.Z tag whose commit is
# reachable from origin/main (see scripts/check-release-tag.sh). The release
# workflow runs it first in the verify job; locally: make release-tag-check TAG=v1.2.3
release-tag-check:
	bash scripts/check-release-tag.sh "$(TAG)" "$(or $(MAIN_REF),origin/main)"

# release-tag-test exercises that check against throwaway repositories.
release-tag-test:
	bash scripts/test-check-release-tag.sh

# release-script-test exercises the post-build / post-publish verification scripts
# against fixtures and stubbed gh/cosign (no network, nothing published).
release-script-test:
	bash scripts/test-check-release-binaries.sh
	bash scripts/test-verify-published-release.sh

# check-release-binaries proves the built artifacts: checksums match SHA256SUMS
# and the host binary runs with its embedded runtime. Pass VERSION=x.y.z to also
# require that exact embedded version (a snapshot embeds SNAPSHOT and shows Vdev).
check-release-binaries:
	bash scripts/check-release-binaries.sh dist/published.tsv dist/SHA256SUMS $(VERSION)

# verify-published-release proves an already-published GitHub Release: checksums,
# embedded version == tag, cosign bundles, build attestations and the SBOM.
# Needs gh, cosign and python3; used by the release workflow after publishing.
verify-published-release:
	bash scripts/verify-published-release.sh "$(TAG)" dist/published.tsv

# release-check validates the GoReleaser config before a tag-triggered release.
release-check:
	"$(GORELEASER)" check

# The module proxy occasionally drops a large download mid-stream (for example
# "stream error ... INTERNAL_ERROR"); the module cache keeps what already
# arrived, so a retry of the same pinned version is cheap and changes nothing.
install-goreleaser:
	@for attempt in 1 2 3; do \
	  GOCACHE="$(GOCACHE)" go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) && exit 0; \
	  echo "install-goreleaser: attempt $$attempt failed" >&2; \
	  [ "$$attempt" -lt 3 ] && sleep $$((attempt * 10)); \
	done; \
	echo "install-goreleaser: failed after 3 attempts" >&2; exit 1

check-release-artifacts:
	bash scripts/check-release-artifacts.sh

check-release-assets:
	bash scripts/check-release-assets.sh "$(TAG)" dist/published.tsv

# Also covers the single host-Node runtime build.
release-reproducible-check:
	bash scripts/check-reproducible-build.sh "$(GOCACHE)"

# release-test validates release config and local snapshot artifacts without publishing.
release-test: release-check snapshot check-release-artifacts check-release-binaries

release-dry-run: install-goreleaser release-test

# release publishes to GitHub — requires GITHUB_TOKEN.
release:
	"$(GORELEASER)" release --clean

# snapshot builds release artifacts locally without publishing (no token needed).
snapshot:
	"$(GORELEASER)" release --snapshot --clean --skip=publish

clean:
	rm -rf bin/ dist/ coverage/ coverage.out coverage.html cover.out coverage_cmd.out

# compile-skill regenerates the compiled bootstrap artifacts in .strategist/.compiled/.
# Run after editing any file under .strategist/ to keep the fast-path active.
compile-skill:
	strategist compile --root .strategist

# build-standalone is retained as the explicit release/install target, but it
# now has the same deterministic embedded OpenSpec build as `make build`.
build-standalone: build

# standalone-smoke proves the embedded OpenSpec bundle installs and passes
# check using the supported host Node prerequisite.
standalone-smoke:
	./scripts/smoke-standalone-install.sh

# doctor checks the developer environment (Go and golangci-lint versions against
# go.mod and the CI pin, python, a stale strategist on PATH, temp space, the
# pre-commit hook) and names the remedy for each fault.
doctor:
	@bash scripts/doctor.sh

# install-hooks copies the tracked pre-commit hook into .git/hooks (git does not
# version that directory). An existing different hook is kept as pre-commit.bak.
install-hooks:
	@test -d .git || { echo "not a git checkout (.git missing)" >&2; exit 1; }
	@if [ -f .git/hooks/pre-commit ] && ! cmp -s scripts/hooks/pre-commit .git/hooks/pre-commit; then cp -f .git/hooks/pre-commit .git/hooks/pre-commit.bak && echo "[Strategist] previous hook saved as .git/hooks/pre-commit.bak"; fi
	@cp -f scripts/hooks/pre-commit .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
	@echo "[Strategist] pre-commit hook installed from scripts/hooks/pre-commit"
