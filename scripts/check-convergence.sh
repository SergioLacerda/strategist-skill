#!/usr/bin/env bash
set -euo pipefail

echo "Checking runtime/package-boundary convergence..."

grep -q 'filepath.Join(strategistDir, "plugins", "catalog.yaml")' internal/dojo/checker_manifest.go \
  || { echo "DRIFT: dojo/checker_manifest.go is not reading the plugin catalog authority"; exit 1; }

if grep -qE '"skills".*"skill.yaml"' internal/dojo/checker_manifest.go; then
  echo "DRIFT: dojo/checker_manifest.go still reads a provider compatibility view"
  exit 1
fi

grep -q 'filepath.Join(strategistDir, "plugins", "catalog.yaml")' internal/dojo/checker_manifest_test.go \
  || { echo "DRIFT: dojo/checker_manifest_test.go does not exercise the plugin catalog authority"; exit 1; }

grep -q 'cataloged/custom Weapons' internal/domain/types.go \
  || { echo "DRIFT: internal/domain/types.go lost catalog-first RoleSlotMap resolution"; exit 1; }

test ! -d strategist \
  || { echo "DRIFT: strategist/ exists — the authoring mirror was retired (W7a); author in internal/embed/defaults/"; exit 1; }

test -d internal/embed/defaults/internal_skills \
  || { echo "DRIFT: internal/embed/defaults/internal_skills/ missing — authoring tree broken"; exit 1; }

# "OK" here means the contract/byte drift class checked above (a handful of
# literal path/symbol strings staying in sync across code and docs) found
# nothing — see docs/drift-detection-matrix.md for what this script does and
# does not check.
echo "Convergence check: OK"
