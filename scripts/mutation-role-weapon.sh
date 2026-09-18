#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go_cache="${GOCACHE:-/tmp/go-build-cache}"
go_mod_cache="${GOMODCACHE:-/tmp/go-mod-cache}"
report="$(mktemp "${TMPDIR:-/tmp}/role-weapon-mutation.XXXXXX")"
trap 'rm -f "$report"' EXIT

run_mutation_suite() {
	local package_path="$1"
	local test_name="$2"

	GOCACHE="$go_cache" GOMODCACHE="$go_mod_cache" \
		go test "$package_path" -run "^${test_name}$" -count=1 -v \
		| tee -a "$report"
}

cd "$repo_root"
echo "role-weapon mutation gate: running critical fixture mutants"
run_mutation_suite ./internal/rolevalidation TestRoleWeaponCriticalMutants
run_mutation_suite ./internal/install TestWizardCriticalMutants

summary_count="$(grep -Ec 'critical mutants killed: [0-9]+/[0-9]+' "$report")"
if [ "$summary_count" -ne 2 ]; then
	echo "role-weapon mutation gate: FAIL missing critical-mutant summary" >&2
	exit 1
fi

echo "role-weapon mutation gate: PASS"
grep -E 'critical mutants killed: [0-9]+/[0-9]+' "$report"
