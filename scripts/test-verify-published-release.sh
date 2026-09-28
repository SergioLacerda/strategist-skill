#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
verifier="$script_dir/verify-published-release.sh"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

case "$(uname -s)" in Linux) os=linux ;; Darwin) os=darwin ;; *) echo "OK: skipped (unsupported host)"; exit 0 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; *) echo "OK: skipped (unsupported arch)"; exit 0 ;; esac
host="strategist-$os-$arch"

# A fake release: one host binary, its bundle, SHA256SUMS and an SBOM.
release="$tmp/release"
mkdir -p "$release" "$tmp/bin"
printf '#!/bin/sh\nprintf "%%s\\n" "V1.2.3" "runtime: embedded OpenSpec 1.13.0"\n' > "$release/$host"
chmod +x "$release/$host"
printf 'bundle' > "$release/$host.bundle"
(cd "$release" && sha256sum "$host" > SHA256SUMS)
printf '{"bomFormat":"CycloneDX","components":[{"name":"x"}]}' > "$release/strategist-sbom.cdx.json"
printf 'dist/x\t%s\n' "$host" > "$tmp/published.tsv"

# Stubs for the network tools; FAIL_COSIGN / FAIL_ATTEST switch them off.
cat > "$tmp/bin/gh" <<STUB
#!/bin/sh
case "\$1 \$2" in
  "release download") dir=; while [ \$# -gt 0 ]; do [ "\$1" = "--dir" ] && dir=\$2; shift; done; cp -r "$release"/. "\$dir" ;;
  "attestation verify") [ -z "\${FAIL_ATTEST:-}" ] ;;
  *) exit 1 ;;
esac
STUB
cat > "$tmp/bin/cosign" <<'STUB'
#!/bin/sh
[ -z "${FAIL_COSIGN:-}" ]
STUB
chmod +x "$tmp/bin/gh" "$tmp/bin/cosign"

verify() { PATH="$tmp/bin:$PATH" bash "$verifier" "$1" "$tmp/published.tsv" >/dev/null 2>&1; }

verify v1.2.3 || { echo "FAIL: a valid published release should pass" >&2; exit 1; }
if verify v1.2.4; then echo "FAIL: a tag that differs from the embedded version should fail" >&2; exit 1; fi
if FAIL_COSIGN=1 verify v1.2.3; then echo "FAIL: a cosign failure should fail" >&2; exit 1; fi
if FAIL_ATTEST=1 verify v1.2.3; then echo "FAIL: an attestation failure should fail" >&2; exit 1; fi

rm "$release/$host.bundle"
if verify v1.2.3; then echo "FAIL: a missing cosign bundle should fail" >&2; exit 1; fi
printf 'bundle' > "$release/$host.bundle"

printf '{"bomFormat":"SPDX"}' > "$release/strategist-sbom.cdx.json"
if verify v1.2.3; then echo "FAIL: a non-CycloneDX SBOM should fail" >&2; exit 1; fi

echo "OK: published release verification"
