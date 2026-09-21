#!/usr/bin/env bash
# Acceptance smoke for the standalone build: a binary with the embedded
# OpenSpec bundle must install the Ranked provider and pass `strategist check`
# with only the supported host Node on PATH (OpenSpec itself is absent).
#
# Runs on Linux, macOS and Windows (Git Bash). Usage:
#   scripts/smoke-standalone-install.sh [path-to-binary]
# Without an argument it builds the binary itself.
set -euo pipefail
cd "$(dirname "$0")/.."

node="$(command -v node || true)"
[[ -n "$node" ]] || { echo "Node.js >=20.19.0 is required for this smoke" >&2; exit 1; }
node_version="$($node --version)"
[[ "$node_version" =~ ^v(2[1-9]|20\.(1[9]|[2-9][0-9])|[3-9][0-9])\. ]] || { echo "unsupported host Node: $node_version (need >=20.19.0)" >&2; exit 1; }

# Native path spelling for the binary under test (Git Bash on Windows hands
# native executables MSYS paths otherwise).
native() { if command -v cygpath >/dev/null 2>&1; then cygpath -m "$1"; else printf '%s' "$1"; fi; }

bin="${1:-}"
build_dir=""
if [[ -z "$bin" ]]; then
  build_dir="$(mktemp -d)"
  bin="$build_dir/strategist$(go env GOEXE)"
  CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$bin" ./cmd/strategist
fi

# The script changes directory below, so the binary must be an absolute path.
bin="$(cd "$(dirname "$bin")" && pwd)/$(basename "$bin")"

work="$(mktemp -d)"
empty_path="$(mktemp -d)"
trap 'rm -rf "$work" "$empty_path" ${build_dir:+"$build_dir"}' EXIT
nwork="$(native "$work")"
npath="$(native "$(dirname "$node")")"

# A clean environment: PATH contains only Node, proving OpenSpec is resolved
# from the embedded bundle rather than from the host.
# SystemRoot is the one variable Windows processes (Node) need to start and is
# always present on a real client, so it is passed through.
clean_env=(env -i HOME="$nwork" USERPROFILE="$nwork" PATH="$npath")
sysroot="${SYSTEMROOT:-${SystemRoot:-}}"
[[ -z "$sysroot" ]] || clean_env+=(SystemRoot="$sysroot" TEMP="$nwork" TMP="$nwork")

# Accept the wizard defaults (the Ranked option is pre-selected).
printf '\n%.0s' $(seq 80) | "${clean_env[@]}" "$bin" install --wizard --target "$nwork" >"$work/install.log" 2>&1 \
  || { echo "install failed:" >&2; tail -8 "$work/install.log" >&2; exit 1; }

test -f "$work/.strategist/openspec/config.yaml" || { echo "missing openspec/config.yaml" >&2; exit 1; }
test -f "$work/.strategist/weapon-runtime/openspec-propose/openspec/dist/core/artifact-graph/openspec.mjs" || { echo "missing embedded OpenSpec runtime" >&2; exit 1; }

status="$(cd "$work" && "${clean_env[@]}" "$bin" check --json 2>/dev/null \
  | node -e 'let s=""; process.stdin.on("data",d=>s+=d).on("end",()=>console.log(JSON.parse(s).status))')"
[[ "$status" == "ready" ]] || { echo "check status: $status" >&2; exit 1; }
echo "standalone smoke OK: install + check ready with embedded OpenSpec and host Node"
