#!/usr/bin/env bash
set -euo pipefail

manifest="${1:-dist/artifacts.json}"
published="${2:-dist/published.tsv}"
checksums="${3:-dist/SHA256SUMS}"

if [[ ! -f "$manifest" ]]; then
  echo "::error::$manifest not found - run 'make release-test' or 'make snapshot' before checking release artifacts" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "::error::python3 is required to parse $manifest" >&2
  exit 1
fi

python3 - "$manifest" > "$published" <<'PY'
import json
import sys

manifest = sys.argv[1]
with open(manifest, encoding="utf-8") as handle:
    artifacts = json.load(handle)

published = [
    artifact
    for artifact in artifacts
    if artifact.get("type") in {"Archive", "Binary"}
    and artifact.get("extra", {}).get("Format", "") != ""
]

if not published:
    seen = set()
    published = []
    for artifact in artifacts:
        path = artifact.get("path")
        if artifact.get("type") in {"Archive", "Binary"} and path not in seen:
            seen.add(path)
            published.append(artifact)

for artifact in published:
    print(f"{artifact.get('path', '')}\t{artifact.get('name', '')}")
PY

if [[ ! -s "$published" ]]; then
  echo "::error::goreleaser published no binaries or archives - $manifest had no \"Archive\" or \"Binary\" entries" >&2
  python3 - "$manifest" >&2 <<'PY' || true
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    for artifact in json.load(handle):
        print(f"{artifact.get('type', '')}\t{artifact.get('name', '')}")
PY
  exit 1
fi

while IFS=$'\t' read -r path name; do
  if [[ ! -s "$path" ]]; then
    echo "::error::published artifact '$name' is missing or empty at '$path'" >&2
    exit 1
  fi
  echo "  $name -> $path"
done < "$published"

echo "Published $(wc -l < "$published") artifact(s)."

if [[ ! -s "$checksums" ]]; then
  echo "::error::$checksums is missing or empty - bootstrap.sh and provenance attestation require it" >&2
  exit 1
fi
