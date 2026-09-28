#!/usr/bin/env bash
# Offline test for scripts/check-skill-for-hire-pin.sh: a fixture pin note and
# fixture upstream API responses stand in for the real note and the network.
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
checker="$script_dir/check-skill-for-hire-pin.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

atlas=f8671ffdec326ca0000998321e949775ce6afc7f4fe6a0e881c21ab410329f02
chest=03eebd0e1111111111111111111111111111111111111111111111111115aad3

cat > "$tmp/pin.md" <<NOTE
## Source repository

| Field | Value | Source |
|---|---|---|
| Repository | https://github.com/example/skill-for-hire | KF-001 |
| Tag | \`v0.0.1\` (informational) | KF-001 |
| Tag commit | \`21a9c8e\` | KF-001 |

## Per-package record

| Field | \`atlas\` 0.1.0 | \`treasure-chest\` 0.1.0 |
|---|---|---|
| Tarball | \`atlas-0.1.0.tar.gz\` | \`treasure-chest-0.1.0.tar.gz\` |
| Tarball sha256 | \`$atlas\` | INCOMPLETE: only the recorded prefix \`03eebd0e...15aad3\`; not verified |

- Release \`immutable\` flag: \`false\` (GitHub release API, KF-005).
NOTE

mkdir "$tmp/api"
upstream() { # <tag-commit-sha> <atlas-digest> <chest-digest> <immutable>
  printf '{\n  "sha": "%s",\n  "commit": {}\n}\n' "$1" > "$tmp/api/commit.json"
  cat > "$tmp/api/release.json" <<JSON
{
  "name": "v0.0.1",
  "immutable": $4,
  "assets": [
    {
      "name": "atlas-0.1.0.tar.gz",
      "digest": "sha256:$2"
    },
    {
      "name": "treasure-chest-0.1.0.tar.gz",
      "digest": "sha256:$3"
    }
  ]
}
JSON
}

run() { SFH_FIXTURE_DIR="$tmp/api" bash "$checker" "${1:-$tmp/pin.md}"; }
expect_fail() { # <label> [pin-note]
  if run "${2:-}" >"$tmp/out" 2>&1; then
    echo "FAIL: $1 should fail" >&2; cat "$tmp/out" >&2; exit 1
  fi
}

upstream 21a9c8ef384d5cbb91a21ca41bd6054e52620438 "$atlas" "$chest" false
run >"$tmp/out" 2>&1 || { echo "FAIL: matching pin should pass" >&2; cat "$tmp/out" >&2; exit 1; }
grep -q 'skill-for-hire pin: OK' "$tmp/out" || { echo "FAIL: missing OK summary" >&2; exit 1; }

upstream deadbeef384d5cbb91a21ca41bd6054e52620438 "$atlas" "$chest" false
expect_fail "moved tag commit"

upstream 21a9c8ef384d5cbb91a21ca41bd6054e52620438 "${atlas%?}0" "$chest" false
expect_fail "changed atlas digest"

upstream 21a9c8ef384d5cbb91a21ca41bd6054e52620438 "$atlas" "ffffffff${chest#????????}" false
expect_fail "changed treasure-chest prefix"

upstream 21a9c8ef384d5cbb91a21ca41bd6054e52620438 "$atlas" "$chest" true
expect_fail "flipped immutable flag"

upstream 21a9c8ef384d5cbb91a21ca41bd6054e52620438 "$atlas" "$chest" false
sed 's/21a9c8e/0000000/' "$tmp/pin.md" > "$tmp/altered.md"
expect_fail "altered value in a scratch copy of the note" "$tmp/altered.md"

set +e
SFH_FIXTURE_DIR="$tmp/api" bash "$checker" "$tmp/missing.md" >/dev/null 2>&1
rc=$?
set -e
[[ "$rc" -eq 2 ]] || { echo "FAIL: a missing pin note should exit 2, got $rc" >&2; exit 1; }

echo "OK: skill-for-hire pin check"
