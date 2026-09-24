#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check-continue-on-error.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

mkdir "$tmp/ok" "$tmp/none" "$tmp/uncatalogued" "$tmp/far" "$tmp/partial"

cat > "$tmp/ok/w.yml" <<'YAML'
jobs:
  a:
    steps:
      # continue-on-error-catalog: owner=maintainer reason=evidence run remove-when=green once
      - name: x
        continue-on-error: true
        run: true
YAML

cat > "$tmp/none/w.yml" <<'YAML'
jobs:
  a:
    steps:
      - run: true
        # continue-on-error: true   (a commented mention is not a use)
YAML

cat > "$tmp/uncatalogued/w.yml" <<'YAML'
jobs:
  a:
    steps:
      - name: x
        continue-on-error: true
        run: true
YAML

{
  echo "      # continue-on-error-catalog: owner=m reason=r remove-when=w"
  for i in $(seq 1 20); do echo "      # filler $i"; done
  echo "      - continue-on-error: true"
} > "$tmp/far/w.yml"

cat > "$tmp/partial/w.yml" <<'YAML'
jobs:
  a:
    steps:
      # continue-on-error-catalog: owner=maintainer
      - continue-on-error: true
YAML

bash "$checker" "$tmp/ok" >/dev/null 2>&1 || { echo "FAIL: catalogued entry should pass" >&2; exit 1; }
bash "$checker" "$tmp/none" >/dev/null 2>&1 || { echo "FAIL: no entries should pass" >&2; exit 1; }
for d in uncatalogued far partial; do
  if bash "$checker" "$tmp/$d" >/dev/null 2>&1; then
    echo "FAIL: $d should fail" >&2
    exit 1
  fi
done

echo "OK: continue-on-error catalog check"
