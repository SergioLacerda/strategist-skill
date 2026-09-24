#!/usr/bin/env bash
# The release-dry-run job installs GoReleaser at GORELEASER_VERSION (Makefile);
# the release job must run the same version, or a passing dry run proves nothing
# about the real release.
# Usage: check-goreleaser-pin.sh [Makefile] [release-workflow]
set -euo pipefail

makefile="${1:-Makefile}"
workflow="${2:-.github/workflows/release.yml}"

dry_run="$(sed -n 's/^GORELEASER_VERSION[[:space:]]*?\{0,1\}=[[:space:]]*\(v[0-9][0-9.]*\).*$/\1/p' "$makefile" | head -n 1)"
release="$(awk '/goreleaser\/goreleaser-action@/{found=1} found && /version:/{gsub(/[" ]/,"",$2); print $2; exit}' "$workflow")"

if [[ -z "$dry_run" || -z "$release" ]]; then
  echo "::error::could not read the GoReleaser version from $makefile ('$dry_run') or $workflow ('$release')" >&2
  exit 1
fi
if [[ "$dry_run" != "$release" ]]; then
  echo "::error::GoReleaser version drift: $makefile pins $dry_run, $workflow runs $release" >&2
  exit 1
fi
echo "goreleaser pin ok ($release)"
