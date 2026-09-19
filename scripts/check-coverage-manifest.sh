#!/usr/bin/env bash
set -euo pipefail

manifest="${1:-scripts/coverage-packages.tsv}"
exemptions="${2:-scripts/coverage-exemptions.tsv}"
go_cache="${3:-${GOCACHE:-/tmp/go-build-cache}}"

fail=0
diagnostics=()
declare -A manifest_packages=()
declare -A exemption_packages=()

fail_msg() {
  diagnostics+=("$1 $2")
  fail=1
}

if [[ ! -f "$manifest" ]]; then
  fail_msg COVERAGE_MANIFEST_INVALID "manifest not found: $manifest"
fi
if [[ ! -f "$exemptions" ]]; then
  fail_msg COVERAGE_EXEMPTION_INVALID "allowlist not found: $exemptions"
fi
if (( fail != 0 )); then
  printf '%s\n' "${diagnostics[@]}" | LC_ALL=C sort -u >&2
  exit "$fail"
fi

while IFS=$'\t' read -r pkg minimum reason extra; do
  [[ -z "${pkg:-}" || "$pkg" == \#* ]] && continue
  if [[ -n "${extra:-}" || -z "${minimum:-}" || -z "${reason:-}" ]]; then
    fail_msg COVERAGE_MANIFEST_INVALID "row must contain package, numeric threshold, and reason: $pkg"
    continue
  fi
  if [[ ! "$pkg" =~ ^(cmd|internal|treasure-chest)(/[^[:space:]]+)?$ ]]; then
    fail_msg COVERAGE_MANIFEST_INVALID "out-of-scope package: $pkg"
    continue
  fi
  if [[ ! "$minimum" =~ ^[0-9]+([.][0-9]+)?$ ]] || ! awk -v value="$minimum" 'BEGIN { exit !(value >= 0 && value <= 100) }'; then
    fail_msg COVERAGE_MANIFEST_INVALID "invalid threshold for $pkg: $minimum"
  fi
  if [[ -n "${manifest_packages[$pkg]+x}" ]]; then
    fail_msg COVERAGE_MANIFEST_INVALID "duplicate package: $pkg"
  fi
  manifest_packages["$pkg"]="$minimum"
done < "$manifest"

while IFS=$'\t' read -r pkg owner reason extra; do
  [[ -z "${pkg:-}" || "$pkg" == \#* ]] && continue
  if [[ -n "${extra:-}" || -z "${owner:-}" || -z "${reason:-}" ]]; then
    fail_msg COVERAGE_EXEMPTION_INVALID "row must contain package, owner, and reason: $pkg"
    continue
  fi
  if [[ ! "$pkg" =~ ^(cmd|internal|treasure-chest)(/[^[:space:]]+)?$ ]]; then
    fail_msg COVERAGE_EXEMPTION_INVALID "out-of-scope package: $pkg"
    continue
  fi
  if [[ -n "${exemption_packages[$pkg]+x}" ]]; then
    fail_msg COVERAGE_EXEMPTION_INVALID "duplicate package: $pkg"
  fi
  if [[ -n "${manifest_packages[$pkg]+x}" ]]; then
    fail_msg COVERAGE_EXEMPTION_INVALID "package appears in both manifest and exemptions: $pkg"
  fi
  exemption_packages["$pkg"]="$owner"
done < "$exemptions"

module_path="$(GOCACHE="$go_cache" go list -m -f '{{.Path}}')" || {
  fail_msg COVERAGE_INVENTORY_DISCOVERY_FAILED "unable to determine Go module path"
  exit "$fail"
}

mapfile -t discovered < <(GOCACHE="$go_cache" go list ./cmd/... ./internal/... ./treasure-chest/... | sed "s#^${module_path}/##" | LC_ALL=C sort)
if (( ${PIPESTATUS[0]} != 0 )); then
  fail_msg COVERAGE_INVENTORY_DISCOVERY_FAILED "unable to discover production Go packages"
  exit "$fail"
fi

declare -A discovered_packages=()
for pkg in "${discovered[@]}"; do
  [[ -n "$pkg" ]] && discovered_packages["$pkg"]=1
done

while IFS= read -r pkg; do
  if [[ -z "${discovered_packages[$pkg]+x}" ]]; then
    if [[ -n "${manifest_packages[$pkg]+x}" ]]; then
      fail_msg COVERAGE_MANIFEST_STALE "package is not discovered: $pkg"
    else
      fail_msg COVERAGE_EXEMPTION_STALE "package is not discovered: $pkg"
    fi
  fi
done < <(printf '%s\n' "${!manifest_packages[@]}" "${!exemption_packages[@]}" | LC_ALL=C sort -u)

for pkg in "${discovered[@]}"; do
  if [[ -z "${manifest_packages[$pkg]+x}" && -z "${exemption_packages[$pkg]+x}" ]]; then
    fail_msg COVERAGE_INVENTORY_OMITTED "discovered package has no manifest row or reviewed exemption: $pkg"
  fi
done

if (( fail == 0 )); then
  echo "coverage manifest complete: ${#discovered_packages[@]} packages (${#manifest_packages[@]} gated, ${#exemption_packages[@]} exempt)"
else
  printf '%s\n' "${diagnostics[@]}" | LC_ALL=C sort -u >&2
fi
exit "$fail"
