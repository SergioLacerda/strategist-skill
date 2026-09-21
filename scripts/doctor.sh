#!/usr/bin/env bash
# Developer environment check (`make doctor`). It names each fault and its
# remedy instead of letting a hook or a build fail with a message about
# something else. Exit 1 when a FAIL is found; WARN never fails.
#
# Test knobs: DOCTOR_MIN_TMP_MB (default 2048).
set -uo pipefail
cd "$(dirname "$0")/.."

fail=0
warns=0
ok()      { printf '  ok    %s\n' "$*"; }
warning() { printf '  WARN  %s\n' "$*"; warns=$((warns + 1)); }
bad()     { printf '  FAIL  %s\n' "$*"; fail=$((fail + 1)); }

# version_lt A B: true when A sorts strictly before B as a version.
version_lt() { [[ "$1" != "$2" && "$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -1)" == "$1" ]]; }

required_go="$(awk '/^go /{print $2; exit}' go.mod)"
echo "strategist doctor (go.mod requires Go ${required_go})"

# 1. Go toolchain
if ! command -v go >/dev/null 2>&1; then
  bad "go: not found on PATH; install Go ${required_go} or newer"
else
  local_go="$(GOTOOLCHAIN=local go env GOVERSION 2>/dev/null | sed 's/^go//; s/[^0-9.].*$//')"
  if version_lt "${local_go}" "${required_go}"; then
    warning "go: the installed Go is ${local_go}, older than go.mod's ${required_go}; inside the repository the go command downloads ${required_go} automatically (needs network), so run 'go version' in the repository to see the effective one"
  else
    ok "go: ${local_go}"
  fi
fi

# 2. golangci-lint: same discovery order as the pre-commit hook
lint=""
if [[ -x ./bin/golangci-lint ]]; then
  lint="./bin/golangci-lint"
else
  lint="$(command -v golangci-lint 2>/dev/null || true)"
  if [[ -z "${lint}" ]] && command -v go >/dev/null 2>&1; then
    candidate="$(go env GOPATH 2>/dev/null)/bin/golangci-lint"
    [[ -x "${candidate}" ]] && lint="${candidate}"
  fi
fi
if [[ -z "${lint}" ]]; then
  warning "golangci-lint: not installed; the pre-commit hook skips lint (install: GOTOOLCHAIN=go${required_go} go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@<version pinned in .github/workflows/test.yml>)"
else
  lint_out="$("${lint}" --version 2>&1 | head -1)"
  built="$(sed -n 's/.*built with go\([0-9][0-9.]*\).*/\1/p' <<<"${lint_out}")"
  pin="$(grep -A3 'golangci-lint-action' .github/workflows/test.yml 2>/dev/null | sed -n 's/.*version: *v\{0,1\}\([0-9][0-9.]*\).*/\1/p' | head -1)"
  have="$(sed -n 's/.*version \([0-9][0-9.]*\).*/\1/p' <<<"${lint_out}" | head -1)"
  if [[ -n "${built}" ]] && version_lt "${built}" "${required_go}"; then
    bad "golangci-lint: built with go${built}, older than go.mod's ${required_go}, so it refuses to load the config; rebuild: GOTOOLCHAIN=go${required_go} go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v${pin:-latest}"
  elif [[ -n "${pin}" && -n "${have}" && "${have}" != "${pin}" ]]; then
    warning "golangci-lint: ${have} differs from the CI pin ${pin}; results may differ from CI"
  else
    ok "golangci-lint: ${have:-unknown} built with go${built:-unknown}"
  fi
fi

# 3. python: build scripts need a working python (a Windows Store stub exists but cannot run)
py="$(command -v python3 2>/dev/null || command -v python 2>/dev/null || true)"
if [[ -z "${py}" ]]; then
  bad "python: neither python3 nor python found; 'make install' needs it to fetch the pinned Node (use 'make install-lite' to skip)"
elif ! "${py}" -c 'import sys; sys.exit(0 if sys.version_info[0] >= 3 else 1)' >/dev/null 2>&1; then
  bad "python: '${py}' cannot run as Python 3 (a Windows Store stub or a broken shim?); install Python 3 or pass PYTHON=<path> to make"
else
  ok "python: ${py}"
fi

# 4. the strategist on PATH: shadowing and the embedded runtime
if command -v strategist >/dev/null 2>&1; then
  on_path="$(command -v strategist)"
  build_out="$(strategist version --build 2>&1 || true)"
  if grep -q 'runtime payload: none' <<<"${build_out}"; then
    warning "strategist: ${on_path} has no embedded runtime (runtime payload: none); the Ranked provider then needs openspec on PATH. Run 'make install' for a standalone binary"
  elif grep -q 'runtime payload: embedded' <<<"${build_out}"; then
    ok "strategist: ${on_path} (embedded runtime)"
  else
    warning "strategist: ${on_path} does not support 'version --build'; it predates the standalone runtime, run 'make install'"
  fi
  installed="${HOME}/.local/bin/strategist$(go env GOEXE 2>/dev/null)"
  if [[ -e "${installed}" && "${on_path}" != "${installed}" ]]; then
    warning "strategist: PATH resolves ${on_path}, not ${installed}; the binary you install is shadowed"
  fi
else
  ok "strategist: not on PATH (nothing to shadow)"
fi

# 5. temp space: read-only module caches under a temporary HOME fill small tmpfs mounts
tmp_dir="${TMPDIR:-/tmp}"
min_mb="${DOCTOR_MIN_TMP_MB:-2048}"
free_mb="$(df -Pm "${tmp_dir}" 2>/dev/null | awk 'NR==2{print $4}')"
if [[ -n "${free_mb}" && "${free_mb}" -lt "${min_mb}" ]]; then
  warning "temp: only ${free_mb} MB free in ${tmp_dir} (< ${min_mb} MB); tests and builds can fail with 'disk quota exceeded'. Leftover 'tmp.*/go/pkg/mod' directories are read-only: chmod -R u+w them before rm -rf"
else
  ok "temp: ${free_mb:-unknown} MB free in ${tmp_dir}"
fi

# 6. pre-commit hook
if [[ -d .git ]]; then
  if [[ -x .git/hooks/pre-commit ]]; then
    if grep -q 'check-embedded-drift' .git/hooks/pre-commit; then
      ok "git hook: pre-commit is installed (with the embedded catalog drift check)"
    else
      warning "git hook: the installed pre-commit lacks the embedded catalog drift check; run 'make install-hooks'"
    fi
  else
    warning "git hook: .git/hooks/pre-commit is missing or not executable; commits skip gofmt, vet, build and lint"
  fi
fi

echo
if (( fail > 0 )); then
  echo "doctor: ${fail} problem(s), ${warns} warning(s)"
  exit 1
fi
echo "doctor: no problems (${warns} warning(s))"
