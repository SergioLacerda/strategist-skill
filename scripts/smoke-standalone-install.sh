#!/usr/bin/env bash
# Acceptance smoke for the standalone payload build: a binary built with
# -tags strategist_payload must install the Ranked provider and pass
# `strategist check` on a machine with NO openspec/node on PATH.
#
# Usage: scripts/smoke-standalone-install.sh [path-to-binary]
# Without an argument it fetches the host Node and builds the binary itself.
set -euo pipefail
cd "$(dirname "$0")/.."

bin="${1:-}"
if [[ -z "$bin" ]]; then
  python3 scripts/fetch-node-runtime.py --host
  bin="$(mktemp -d)/strategist"
  CGO_ENABLED=0 go build -tags strategist_payload -trimpath -ldflags='-s -w' -o "$bin" ./cmd/strategist
fi

work="$(mktemp -d)"
empty_path="$(mktemp -d)"
trap 'rm -rf "$work" "$empty_path"' EXIT

# Accept the wizard defaults (the Ranked option is pre-selected); env -i with an
# empty PATH proves nothing is resolved from the host.
printf '\n%.0s' $(seq 80) | env -i HOME="$work" PATH="$empty_path" "$bin" install --wizard --target "$work" >"$work/install.log" 2>&1 \
  || { echo "install failed:" >&2; tail -5 "$work/install.log" >&2; exit 1; }

test -f "$work/.strategist/openspec/config.yaml" || { echo "missing openspec/config.yaml" >&2; exit 1; }
test -d "$work/.strategist/weapon-runtime/openspec-propose" || { echo "missing private runtime" >&2; exit 1; }

status="$(cd "$work" && env -i HOME="$work" PATH="$empty_path" "$bin" check --json 2>/dev/null \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["status"])')"
[[ "$status" == "ready" ]] || { echo "check status: $status" >&2; exit 1; }
echo "standalone smoke OK: install + check ready with an empty PATH"
