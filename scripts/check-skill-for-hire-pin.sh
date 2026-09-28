#!/usr/bin/env bash
# Read-only check of the skill-for-hire upstream pin against the live upstream.
# Compares the tag commit, the per-tarball sha256 digests and the release
# `immutable` flag recorded in the pin note with what the GitHub API reports.
# Writes nothing. Everything the upstream returns is untrusted data: it is only
# compared, never executed. Needs network access and curl (no gh, no jq).
#
# Usage: check-skill-for-hire-pin.sh [pin-note]
#   pin-note defaults to the note under .analysis/pending/02-external-skills-adapters/
# Env:   SFH_FIXTURE_DIR  read commit.json / release.json from this directory
#                         instead of the network (used by the offline test)
# Exit:  0 pin matches upstream, 1 mismatch, 2 usage / unreadable note / fetch error
set -euo pipefail

note="${1:-.analysis/pending/02-external-skills-adapters/skill-for-hire-upstream-pin.md}"
mismatches=0

die() { echo "[skill-for-hire-pin] ERROR: $*" >&2; exit 2; }
mismatch() { echo "[skill-for-hire-pin] MISMATCH: $*" >&2; mismatches=$((mismatches + 1)); }

[[ -r "$note" ]] || die "pin note not readable: $note (it lives under the gitignored .analysis/)"

# cell <row-label> <column>: the n-th cell (1 = first value column) of the table row starting with "| <label> |".
cell() {
  awk -F'|' -v label="$1" -v col="$2" '
    { key = $2; gsub(/^ +| +$/, "", key) }
    key == label { print $(col + 2); exit }' "$note"
}
first_code() { { grep -o '`[^`]*`' || true; } | head -n 1 | tr -d '`'; }

repo=$(cell Repository 1 | sed -E 's#.*github\.com/([^ /]+/[^ /]+).*#\1#' | tr -d ' ')
tag=$(cell Tag 1 | first_code)
tag_commit=$(cell "Tag commit" 1 | first_code)
[[ "$repo" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || die "no repository recorded in $note"
[[ -n "$tag" && -n "$tag_commit" ]] || die "tag or tag commit not recorded in $note"
noted_immutable=$(grep -oE 'immutable` flag: `(true|false)`' "$note" | grep -oE '(true|false)' | head -n 1 || true)

fetch() { # <fixture-name> <api-path>
  if [[ -n "${SFH_FIXTURE_DIR:-}" ]]; then
    cat "$SFH_FIXTURE_DIR/$1.json"
  else
    curl -fsS --max-time 20 -H 'Accept: application/vnd.github+json' "https://api.github.com/repos/$repo/$2"
  fi
}
commit_json=$(fetch commit "commits/$tag") || die "could not fetch the commit of tag $tag from $repo"
release_json=$(fetch release "releases/tags/$tag") || die "could not fetch release $tag from $repo"

up_commit=$(grep -m1 '"sha"' <<<"$commit_json" | sed -E 's/.*"sha": *"([0-9a-f]+)".*/\1/')
[[ -n "$up_commit" ]] || die "no commit sha in the upstream response for tag $tag"
if [[ "$up_commit" == "$tag_commit"* ]]; then
  echo "[skill-for-hire-pin] tag $tag commit: ok ($tag_commit)"
else
  mismatch "tag $tag commit: note has $tag_commit, upstream has $up_commit"
fi

# name<TAB>digest pairs of the release assets (pretty-printed API JSON).
assets=$(awk '
  /"name":/   { n = $0; sub(/.*"name": *"/, "", n); sub(/".*/, "", n) }
  /"digest":/ { d = $0; sub(/.*"digest": *"sha256:/, "", d); sub(/".*/, "", d); print n "\t" d }' <<<"$release_json")

col=1
while :; do
  tarball=$(cell Tarball "$col" | first_code)
  [[ -n "$tarball" ]] || break
  recorded=$(cell "Tarball sha256" "$col" | tr -d ' ')
  up_digest=$(awk -F'\t' -v t="$tarball" '$1 == t { print $2; exit }' <<<"$assets")
  if [[ -z "$up_digest" ]]; then
    mismatch "$tarball: no such asset (or no digest) in upstream release $tag"
  elif full=$(printf '%s' "$recorded" | grep -oE '`[0-9a-f]{64}`' | tr -d '`') && [[ -n "$full" ]]; then
    if [[ "$full" == "$up_digest" ]]; then
      echo "[skill-for-hire-pin] $tarball sha256: ok (full digest)"
    else
      mismatch "$tarball sha256: note has $full, upstream has $up_digest"
    fi
  elif part=$(printf '%s' "$recorded" | grep -oE '`[0-9a-f]+\.\.\.[0-9a-f]+`' | tr -d '`') && [[ -n "$part" ]]; then
    prefix=${part%%...*}; suffix=${part##*...}
    if [[ "$up_digest" == "$prefix"* && "$up_digest" == *"$suffix" ]]; then
      echo "[skill-for-hire-pin] $tarball sha256: ok (partial record $part; the note never held the full digest)"
    else
      mismatch "$tarball sha256: note has partial $part, upstream has $up_digest"
    fi
  else
    echo "[skill-for-hire-pin] $tarball sha256: skipped (no digest recorded in the note; upstream has $up_digest)"
  fi
  col=$((col + 1))
done
[[ "$col" -gt 1 ]] || die "no tarball recorded in $note"

up_immutable=$(grep -m1 '"immutable"' <<<"$release_json" | sed -E 's/.*"immutable": *(true|false).*/\1/')
if [[ -z "$noted_immutable" ]]; then
  echo "[skill-for-hire-pin] immutable flag: not recorded in the note (upstream: ${up_immutable:-unknown})"
elif [[ "$noted_immutable" == "$up_immutable" ]]; then
  echo "[skill-for-hire-pin] immutable flag: ok ($up_immutable)"
else
  mismatch "immutable flag: note has $noted_immutable, upstream has ${up_immutable:-unknown}"
fi

if [[ "$mismatches" -gt 0 ]]; then
  echo "[skill-for-hire-pin] $mismatches mismatch(es): the upstream moved or the note is stale; review before trusting the pin" >&2
  exit 1
fi
echo "[skill-for-hire-pin] skill-for-hire pin: OK ($repo $tag)"
