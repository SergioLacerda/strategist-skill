#!/usr/bin/env bash
set -euo pipefail

tag="${1:-}"
published="${2:-dist/published.tsv}"

if [[ -z "$tag" ]]; then
  echo "::notice::skipping published-release asset gate because TAG is not set; local snapshot artifacts were already validated"
  exit 0
fi

if [[ ! -s "$published" ]]; then
  echo "::error::$published is missing or empty - run check-release-artifacts first" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "::error::gh is required to inspect GitHub Release assets" >&2
  exit 1
fi

assets="$(gh release view "$tag" --json assets --jq '.assets[].name')"
missing=0

while IFS=$'\t' read -r _ name; do
  for want in "$name" "${name}.bundle"; do
    if ! grep -qxF "$want" <<<"$assets"; then
      echo "::error::release $tag is missing asset '$want'" >&2
      missing=1
    fi
  done
done < "$published"

if ! grep -qxF "SHA256SUMS" <<<"$assets"; then
  echo "::error::release $tag is missing SHA256SUMS (bootstrap.sh requires it)" >&2
  missing=1
fi

exit "$missing"
