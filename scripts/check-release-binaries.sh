#!/usr/bin/env bash
# Checks the release binaries listed in a published.tsv (path<TAB>asset-name):
#   1. every asset has a SHA256SUMS line and the file's digest matches it;
#   2. SHA256SUMS lists no asset that is not published;
#   3. the binary built for this host runs `version --build`, prints a
#      `V...` version line and reports the embedded OpenSpec runtime;
#   4. when an expected version is given, that line is exactly V<expected>
#      (tag <-> embedded version <-> release consistency).
# A GoReleaser snapshot embeds a SNAPSHOT version that displays as `Vdev`, so
# the dry run omits the expected version; the post-publish check supplies it.
# Usage: check-release-binaries.sh <published.tsv> <SHA256SUMS> [expected-version]
set -euo pipefail

published="${1:-dist/published.tsv}"
sums="${2:-dist/SHA256SUMS}"
expected="${3:-}"

fail() { echo "::error::$*" >&2; exit 1; }

[[ -s "$published" ]] || fail "$published is missing or empty"
[[ -s "$sums" ]] || fail "$sums is missing or empty"

declare -A listed=()
while IFS= read -r name; do listed["$name"]=1; done < <(awk 'NF>=2 {print $2}' "$sums")

host_bin=""
host_name=""
case "$(uname -s)" in Linux) os=linux ;; Darwin) os=darwin ;; *) os="" ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) arch="" ;; esac
[[ -n "$os" && -n "$arch" ]] && host_name="strategist-$os-$arch"

declare -A seen=()
while IFS=$'\t' read -r path name; do
  [[ -n "$name" ]] || continue
  [[ -s "$path" ]] || fail "published artifact '$name' is missing or empty at '$path'"
  want="$(awk -v n="$name" '$2==n {print $1}' "$sums")"
  [[ -n "$want" ]] || fail "asset '$name' has no line in $sums"
  got="$(sha256sum "$path" | awk '{print $1}')"
  [[ "$want" == "$got" ]] || fail "asset '$name' digest $got does not match $sums ($want)"
  seen["$name"]=1
  if [[ "$name" == "$host_name" ]]; then host_bin="$path"; fi
done < "$published"

for name in "${!listed[@]}"; do
  [[ -n "${seen[$name]:-}" ]] || fail "$sums lists '$name', which is not a published asset"
done

if [[ -z "$host_bin" ]]; then
  echo "::notice::no published binary for this host (${host_name:-unsupported}); skipping the run check"
  echo "release binaries ok: ${#seen[@]} checksum(s) verified"
  exit 0
fi

out="$("$host_bin" version --build)" || fail "'$host_bin version --build' failed"
first="$(printf '%s\n' "$out" | head -n 1)"
[[ "$first" =~ ^V ]] || fail "unexpected version line '$first' from $host_bin"
grep -q "embedded OpenSpec" <<<"$out" || fail "$host_bin does not report the embedded OpenSpec runtime"
if [[ -n "$expected" && "$first" != "V$expected" ]]; then
  fail "embedded version '$first' does not match the expected V$expected"
fi

echo "release binaries ok: ${#seen[@]} checksum(s) verified; $host_name runs as '$first'"
