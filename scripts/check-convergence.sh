#!/usr/bin/env bash
set -euo pipefail

echo "Checking runtime/package-boundary convergence..."

grep -q '"skills", mc.ExpectedProvider' internal/dojo/checker_manifest.go \
  || { echo "DRIFT: dojo/checker_manifest.go uses old provider path (not skills/<provider>/skill.yaml)"; exit 1; }

grep -q '"skills", "brainstorming"' internal/dojo/checker_manifest_test.go \
  || { echo "DRIFT: dojo/checker_manifest_test.go uses old provider path"; exit 1; }

grep -q 'skills/<provider>/skill.yaml' internal/domain/types.go \
  || { echo "DRIFT: internal/domain/types.go lost the canonical provider path skills/<provider>/skill.yaml"; exit 1; }

test ! -d strategist \
  || { echo "DRIFT: strategist/ exists — the authoring mirror was retired (W7a); author in internal/embed/defaults/"; exit 1; }

test -d internal/embed/defaults/internal_skills \
  || { echo "DRIFT: internal/embed/defaults/internal_skills/ missing — authoring tree broken"; exit 1; }

echo "Convergence check: OK"
