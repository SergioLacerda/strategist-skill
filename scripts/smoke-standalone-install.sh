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
if [[ -n "$sysroot" ]]; then
  # PATHEXT is what lets Windows resolve "node" to node.exe. The exclusion is
  # read by the MSYS "env" process that spawns the binary (not by the binary),
  # so it is exported; it stops Git Bash rewriting the already-native PATH into
  # a POSIX/mixed list that the Go binary cannot search.
  export MSYS2_ENV_CONV_EXCL=PATH
  clean_env+=(SystemRoot="$sysroot" TEMP="$nwork" TMP="$nwork" PATHEXT=".COM;.EXE;.BAT;.CMD")
fi

echo "smoke: host Node $node_version at $npath"
# Prove what a native child really receives, so a PATH mangled by the shell is
# visible in the job log instead of surfacing as "node not found" later.
echo "smoke: child PATH=$("${clean_env[@]}" "$node" -p 'process.env.PATH')"

# fail_with_log reports the command's own error line first (cobra prints it
# above the usage text, which a plain tail would show instead), then the whole
# log, so a CI failure is diagnosable from the job output alone.
fail_with_log() {
  echo "$1 failed:" >&2
  grep -E '^(Error|\[Strategist\].*(failed|error))' "$2" >&2 || true
  echo "---- full log ----" >&2
  cat "$2" >&2
  exit 1
}

# Accept the wizard defaults (the Ranked option is pre-selected).
printf '\n%.0s' $(seq 80) | "${clean_env[@]}" "$bin" install --wizard --target "$nwork" >"$work/install.log" 2>&1 \
  || fail_with_log install "$work/install.log"

test -f "$work/.strategist/openspec/config.yaml" || { echo "missing openspec/config.yaml" >&2; exit 1; }
test -f "$work/.strategist/weapon-runtime/openspec-propose/openspec/dist/core/artifact-graph/openspec.mjs" || { echo "missing embedded OpenSpec runtime" >&2; exit 1; }

(cd "$work" && "${clean_env[@]}" "$bin" check --json >"$work/check.json" 2>"$work/check.log") || true
status="$(node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{console.log(JSON.parse(s).status)}catch(e){console.log("unparseable")}})' <"$work/check.json")"
[[ "$status" == "ready" ]] || { cat "$work/check.json" >>"$work/check.log"; fail_with_log "check (status=$status)" "$work/check.log"; }
echo "standalone smoke OK: install + check ready with embedded OpenSpec and host Node"
