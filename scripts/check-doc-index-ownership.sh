#!/usr/bin/env bash
set -euo pipefail

root=${1:-.}
contract="$root/docs/index-ownership.tsv"
index="$root/docs/README.md"

failures=0
entries=0

fail() {
  printf 'DOC_INDEX_OWNERSHIP_INVALID %s\n' "$1" >&2
  failures=$((failures + 1))
}

if [[ ! -f "$contract" ]]; then
  fail "missing contract: docs/index-ownership.tsv"
fi
if [[ ! -f "$index" ]]; then
  fail "missing index: docs/README.md"
fi

if [[ "$failures" -eq 0 ]]; then
  seen_paths=()
  seen_purposes=()
  line_number=0

  while IFS= read -r line || [[ -n "$line" ]]; do
    line_number=$((line_number + 1))
    [[ -z "${line//[[:space:]]/}" || "$line" == \#* ]] && continue

    IFS=$'\t' read -r -a fields <<< "$line"
    if [[ "${#fields[@]}" -ne 4 ]]; then
      fail "line $line_number must contain path, purpose, owner, and policy"
      continue
    fi

    path=${fields[0]}
    purpose=${fields[1]}
    owner=${fields[2]}
    policy=${fields[3]}
    entries=$((entries + 1))

    if [[ "$path" == docs/generated/* || "$path" == docs/adr/* || "$path" == docs/runbooks/* ]]; then
      fail "line $line_number uses an explicitly excluded path: $path"
    fi
    if [[ "$owner" != "docs/README.md" || "$policy" != "required" ]]; then
      fail "line $line_number must require docs/README.md ownership: $path"
    fi
    for seen_path in "${seen_paths[@]}"; do
      if [[ "$seen_path" == "$path" ]]; then
        fail "duplicate canonical path: $path"
      fi
    done
    for seen_purpose in "${seen_purposes[@]}"; do
      if [[ "$seen_purpose" == "$purpose" ]]; then
        fail "duplicate canonical purpose: $purpose"
      fi
    done
    seen_paths+=("$path")
    seen_purposes+=("$purpose")

    if [[ ! -f "$root/$path" ]]; then
      fail "canonical path does not exist: $path"
      continue
    fi

    link_path=${path#docs/}
    if ! grep -Fq "]($link_path)" "$index"; then
      fail "canonical path is not linked by docs/README.md: $path"
    fi
  done < "$contract"

  if [[ "$entries" -eq 0 ]]; then
    fail "contract contains no canonical documentation entries"
  fi
fi

if [[ "$failures" -ne 0 ]]; then
  exit 1
fi

printf 'OK: docs index ownership valid (%d canonical surfaces; generated, ADR, and runbook indexes excluded)\n' "$entries"
