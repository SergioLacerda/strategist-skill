#!/usr/bin/env bash
# Post-publish verification of a GitHub Release, run by the release workflow
# after every upload: downloads the published assets and proves them, going
# beyond the presence check in check-release-assets.sh:
#   - SHA256SUMS matches every asset, and the linux binary reports V<tag>;
#   - each asset's cosign bundle verifies against this repository's release
#     workflow identity for this tag;
#   - each asset has a GitHub build-provenance attestation from this repository;
#   - the CycloneDX SBOM asset is valid and non-empty.
# Usage: verify-published-release.sh <tag> [published.tsv]
# Env:   GITHUB_REPOSITORY (owner/name, set by Actions), GH_TOKEN for gh.
set -euo pipefail

tag="${1:-}"
published="${2:-dist/published.tsv}"
repo="${GITHUB_REPOSITORY:-SergioLacerda/strategist-skill}"
script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)

fail() { echo "::error::$*" >&2; exit 1; }

[[ -n "$tag" ]] || fail "usage: verify-published-release.sh <tag> [published.tsv]"
[[ -s "$published" ]] || fail "$published is missing or empty - run check-release-artifacts first"
for tool in gh cosign python3; do
  command -v "$tool" >/dev/null 2>&1 || fail "$tool is required to verify the published release"
done

dl="$(mktemp -d)"
trap 'rm -rf "$dl"' EXIT
gh release download "$tag" --repo "$repo" --dir "$dl"

# Published names paired with the downloaded files, for the shared binary check.
: > "$dl/downloaded.tsv"
while IFS=$'\t' read -r _ name; do
  [[ -n "$name" ]] || continue
  [[ -s "$dl/$name" ]] || fail "release $tag: asset '$name' was not downloadable"
  printf '%s\t%s\n' "$dl/$name" "$name" >> "$dl/downloaded.tsv"
done < "$published"

bash "$script_dir/check-release-binaries.sh" "$dl/downloaded.tsv" "$dl/SHA256SUMS" "${tag#v}"

identity="^https://github.com/${repo}/\\.github/workflows/release\\.yml@refs/tags/${tag}\$"
while IFS=$'\t' read -r _ name; do
  [[ -n "$name" ]] || continue
  [[ -s "$dl/$name.bundle" ]] || fail "release $tag: missing cosign bundle for '$name'"
  cosign verify-blob "$dl/$name" \
    --bundle "$dl/$name.bundle" \
    --certificate-identity-regexp "$identity" \
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com" >/dev/null \
    || fail "release $tag: cosign verification failed for '$name'"
  gh attestation verify "$dl/$name" --repo "$repo" >/dev/null \
    || fail "release $tag: build provenance attestation failed for '$name'"
done < "$published"

sbom="$(find "$dl" -maxdepth 1 -type f -iname '*sbom*.json' | head -n 1)"
[[ -n "$sbom" ]] || fail "release $tag: no SBOM asset found"
python3 - "$sbom" <<'PY' || fail "release $tag: SBOM is not a valid, non-empty CycloneDX document"
import json, sys
doc = json.load(open(sys.argv[1], encoding="utf-8"))
assert doc.get("bomFormat") == "CycloneDX", "bomFormat is not CycloneDX"
assert doc.get("components"), "no components"
PY

echo "published release $tag verified: checksums, version, cosign, attestation, SBOM"
