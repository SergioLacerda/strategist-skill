#!/usr/bin/env bash
# Fails unless a release tag is fit to publish from: strict vX.Y.Z form, an
# annotated tag object (not lightweight), and pointing at a commit reachable
# from the protected branch. Read-only: it only inspects git objects.
#
# Usage: check-release-tag.sh <tag> [<main-ref>] [<repo-dir>]
#   main-ref defaults to origin/main; repo-dir defaults to the current directory.
set -euo pipefail

tag="${1:-}"
main_ref="${2:-origin/main}"
repo="${3:-.}"

if [[ -z "$tag" ]]; then
  echo "::error::usage: check-release-tag.sh <tag> [<main-ref>] [<repo-dir>]" >&2
  exit 2
fi

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "::error::tag '$tag' is not a release tag of the form vX.Y.Z" >&2
  exit 1
fi

if ! git -C "$repo" rev-parse --verify --quiet "refs/tags/$tag" >/dev/null; then
  echo "::error::tag '$tag' not found in the checkout (fetch tags before running this check)" >&2
  exit 1
fi

kind="$(git -C "$repo" cat-file -t "refs/tags/$tag")"
if [[ "$kind" != "tag" ]]; then
  echo "::error::tag '$tag' is lightweight ($kind object); release tags must be annotated: git tag -a $tag -m \"...\"" >&2
  exit 1
fi

if ! git -C "$repo" rev-parse --verify --quiet "$main_ref" >/dev/null; then
  echo "::error::branch ref '$main_ref' not found; fetch it before running this check" >&2
  exit 1
fi

commit="$(git -C "$repo" rev-list -n 1 "refs/tags/$tag")"
if ! git -C "$repo" merge-base --is-ancestor "$commit" "$main_ref"; then
  echo "::error::tag '$tag' points at $commit, which is not reachable from $main_ref" >&2
  exit 1
fi

echo "release tag $tag ok: annotated, $commit reachable from $main_ref"
