#!/usr/bin/env bash
# Acceptance smoke for the standalone payload build: a binary built with
# -tags strategist_payload must install the Ranked provider and pass
# `strategist check` on a machine with NO openspec/node on PATH.
#
# Runs on Linux, macOS and Windows (Git Bash). Usage:
#   scripts/smoke-standalone-install.sh [path-to-binary]
# Without an argument it fetches the host Node and builds the binary itself.
set -euo pipefail
cd "$(dirname "$0")/.."

py="$(command -v python3 || command -v python || true)"
[[ -n "$py" ]] || { echo "python is required (build-time scripts only)" >&2; exit 1; }

# Native path spelling for the binary under test (Git Bash on Windows hands
# native executables MSYS paths otherwise).
native() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }

bin="${1:-}"
if [[ -z "$bin" ]]; then
  "$py" scripts/fetch-node-runtime.py --host
  bin="$(mktemp -d)/strategist$(go env GOEXE)"
  CGO_ENABLED=0 go build -tags strategist_payload -trimpath -ldflags='-s -w' -o "$bin" ./cmd/strategist
fi

# The script changes directory below, so the binary must be an absolute path.
bin="$(cd "$(dirname "$bin")" && pwd)/$(basename "$bin")"

work="$(mktemp -d)"
empty_path="$(mktemp -d)"
trap 'rm -rf "$work" "$empty_path"' EXIT
nwork="$(native "$work")"
npath="$(native "$empty_path")"

# A clean environment: an empty PATH proves nothing is resolved from the host.
# SystemRoot is the one variable Windows processes (Node) need to start and is
# always present on a real client, so it is passed through.
clean_env=(env -i HOME="$nwork" USERPROFILE="$nwork" PATH="$npath")
sysroot="${SYSTEMROOT:-${SystemRoot:-}}"
[[ -z "$sysroot" ]] || clean_env+=(SystemRoot="$sysroot" TEMP="$nwork" TMP="$nwork")

# Accept the wizard defaults (the Ranked option is pre-selected).
printf '\n%.0s' $(seq 80) | "${clean_env[@]}" "$bin" install --wizard --target "$nwork" >"$work/install.log" 2>&1 \
  || { echo "install failed:" >&2; tail -8 "$work/install.log" >&2; exit 1; }

test -f "$work/.strategist/openspec/config.yaml" || { echo "missing openspec/config.yaml" >&2; exit 1; }
test -d "$work/.strategist/weapon-runtime/openspec-propose" || { echo "missing private runtime" >&2; exit 1; }

status="$(cd "$work" && "${clean_env[@]}" "$bin" check --json 2>/dev/null \
  | "$py" -c 'import json,sys; print(json.load(sys.stdin)["status"])')"
[[ "$status" == "ready" ]] || { echo "check status: $status" >&2; exit 1; }
echo "standalone smoke OK: install + check ready with an empty PATH"
