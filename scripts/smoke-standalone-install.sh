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
nbin="$(native "$bin")"
npath="$(dirname "$node")"

# clean_run runs a command in a clean environment whose PATH contains only the
# host Node's directory, proving OpenSpec is resolved from the embedded bundle
# rather than from the host. Node builds the environment and spawns the child
# itself, so the shell never rewrites it: Git Bash on Windows treats a PATH like
# C:/dir or C:\dir as a POSIX path list, splits it at the drive colon and hands
# the native child a mangled PATH (C;D:\dir), so neither `env -i PATH=...` nor
# MSYS2_ENV_CONV_EXCL is reliable there. The launcher is a real Node process, so
# stdin passes through to the command.
clean_run() {
  "$node" -e '
    const { spawnSync } = require("child_process");
    const path = require("path");
    const [home, ...cmd] = process.argv.slice(1);
    const env = { HOME: home, USERPROFILE: home, PATH: path.dirname(process.execPath) };
    if (process.platform === "win32") {
      // The variables a Windows process needs to start; always present on a client.
      Object.assign(env, { SystemRoot: process.env.SystemRoot, TEMP: home, TMP: home, PATHEXT: ".COM;.EXE;.BAT;.CMD" });
    }
    const r = spawnSync(cmd[0], cmd.slice(1), { env, stdio: "inherit" });
    if (r.error) { console.error(r.error.message); process.exit(1); }
    process.exit(r.status === null ? 1 : r.status);
  ' "$nwork" "$@"
}

echo "smoke: host Node $node_version at $npath"
# Prove what a native child really receives, so a PATH mangled on the way is
# visible in the job log instead of surfacing as "node not found" later.
echo "smoke: child PATH=$(clean_run "$node" -p 'process.env.PATH')"

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
printf '\n%.0s' $(seq 80) | clean_run "$nbin" install --wizard --target "$nwork" >"$work/install.log" 2>&1 \
  || fail_with_log install "$work/install.log"

test -f "$work/.strategist/openspec/config.yaml" || { echo "missing openspec/config.yaml" >&2; exit 1; }
test -f "$work/.strategist/weapon-runtime/openspec-propose/openspec/dist/core/artifact-graph/openspec.mjs" || { echo "missing embedded OpenSpec runtime" >&2; exit 1; }

(cd "$work" && clean_run "$nbin" check --json >"$work/check.json" 2>"$work/check.log") || true
status="$(node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{console.log(JSON.parse(s).status)}catch(e){console.log("unparseable")}})' <"$work/check.json")"
[[ "$status" == "ready" ]] || { cat "$work/check.json" >>"$work/check.log"; fail_with_log "check (status=$status)" "$work/check.log"; }
echo "standalone smoke OK: install + check ready with embedded OpenSpec and host Node"
