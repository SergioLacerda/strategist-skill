#!/usr/bin/env bash
# Regression fixture: `make install`'s identity check and runtime-synchronization
# call (make/release.mk) must always invoke the binary at its known installed
# absolute path ($HOME/.local/bin/strategist$(EXE)), never a PATH-resolved
# `strategist` — see .analysis/refined/20260928-make-install-runtime-sync/design.md.
#
# Method: place a decoy `strategist` (and `strategist<GOEXE>`) earlier on PATH
# than anything else, point HOME at a sandbox so the binary-install location is
# isolated, then run `make install` for real. If the recipe ever regresses to a
# bare `strategist` call, PATH resolution picks up the decoy and it leaves a
# marker; this test fails on that marker. If the recipe stays correct (as it
# does today), the decoy is never invoked and the sandboxed absolute-path
# binary is proven to exist and run.
#
# Side effect: `make install`'s own `--target "$(CURDIR)"` argument is not
# sandboxed (make/release.mk is out of scope for this test to modify), so this
# script does refresh this checkout's real .strategist/ runtime — the same
# side effect running `make install` yourself would have. .strategist/ is
# generated and gitignored; this is expected, not a defect.
#
# HOME is sandboxed for the install *location* (~/.local/bin and the
# Claude/Codex/Gemini shim/seed probes), but GOPATH is pinned to the real one
# so `go build` (build-standalone's dependency) reuses the existing module
# cache instead of re-downloading the toolchain and every dependency into a
# throwaway HOME/go — which is also read-only and cannot be cleaned up by a
# plain `rm -rf` once populated.
#
# Runs on Linux, macOS and Windows (Git Bash), matching
# scripts/smoke-standalone-install.sh's platform coverage. Usage:
#   scripts/test-make-install-absolute-path.sh
set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v go >/dev/null 2>&1; then
  echo "skip: go is required to build the standalone binary for this fixture" >&2
  exit 0
fi
if ! command -v make >/dev/null 2>&1; then
  echo "skip: make is required to run the install target under test" >&2
  exit 0
fi

exe="$(go env GOEXE)"
real_gopath="$(go env GOPATH)"

sandbox_home="$(mktemp -d)"
decoy_dir="$(mktemp -d)"
log="$(mktemp)"
# The Go module cache under GOPATH/pkg/mod is read-only by design; if anything
# ever does write under sandbox_home despite the GOPATH pin above, a plain
# `rm -rf` silently leaves it behind. chmod first so cleanup is unconditional.
trap 'chmod -R u+w "$sandbox_home" 2>/dev/null; rm -rf "$sandbox_home" "$decoy_dir" "$log"' EXIT

marker="$sandbox_home/decoy-invoked.marker"

# write_decoy creates an executable at decoy_dir/$1 that records its own
# invocation (argv) to $marker and exits non-zero, so an accidental real use
# of the decoy in place of the genuine binary fails loudly rather than
# silently appearing to succeed.
write_decoy() {
  local path="$decoy_dir/$1"
  cat >"$path" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$0 \$*" >> "$marker"
echo "[decoy] this is a fixture decoy, not the real strategist binary" >&2
exit 1
EOF
  chmod +x "$path"
}

write_decoy "strategist"
if [[ -n "$exe" ]]; then
  # Windows (GOEXE=".exe"): also shadow the exact name make/release.mk builds,
  # "strategist$(EXE)". A no-op on Linux/macOS, where GOEXE is empty.
  write_decoy "strategist$exe"
fi

set +e
HOME="$sandbox_home" GOPATH="$real_gopath" PATH="$decoy_dir:$PATH" make install >"$log" 2>&1
status=$?
set -e

if [[ -f "$marker" ]]; then
  echo "FAIL: make install invoked a PATH-resolved 'strategist' instead of the absolute installed path" >&2
  echo "---- decoy invocation(s) ----" >&2
  cat "$marker" >&2
  echo "---- make install log ----" >&2
  cat "$log" >&2
  exit 1
fi

real_bin="$sandbox_home/.local/bin/strategist$exe"
if [[ $status -ne 0 ]]; then
  echo "FAIL: make install exited $status without invoking the decoy (not a PATH-resolution regression, but the absolute-path invocation itself failed)" >&2
  cat "$log" >&2
  exit 1
fi
if [[ ! -x "$real_bin" ]]; then
  echo "FAIL: make install reported success but $real_bin was not installed" >&2
  cat "$log" >&2
  exit 1
fi

echo "make-install-absolute-path OK: decoy never invoked, $real_bin installed and used directly"
