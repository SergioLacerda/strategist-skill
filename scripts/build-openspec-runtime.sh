#!/usr/bin/env bash
# Maintainer tool: builds the pinned OpenSpec CLI into one self-contained,
# platform-independent bundle for external-skills-source/openspec-propose/runtime/.
#
# Node and pnpm are needed HERE only (a maintainer's machine), never in the
# Strategist build, tests, or on the client: the output is a plain file tree
# that the Go build embeds.
#
# Usage: scripts/build-openspec-runtime.sh <openspec-repo> <tag> <out-dir>
#   e.g. scripts/build-openspec-runtime.sh ~/dev/other/OpenSpec v1.13.0 /tmp/openspec-runtime
#
# The OpenSpec repo is only read (git archive); dependencies are installed from
# its own pnpm lockfile with --frozen-lockfile, so the result is reproducible.
set -euo pipefail

repo="${1:?openspec repo path}"
tag="${2:?git tag, e.g. v1.13.0}"
out="${3:?output directory}"

for tool in git node npx python3 tar; do
  command -v "$tool" >/dev/null || { echo "missing tool: $tool" >&2; exit 1; }
done

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

commit="$(git -C "$repo" rev-parse "$tag^{commit}")"
git -C "$repo" archive "$tag" | tar -x -C "$work"
cd "$work"

pnpm_version="$(python3 -c 'import json;print(json.load(open("package.json")).get("packageManager","pnpm@9").split("@")[1])')"
pnpm=(npx --yes "pnpm@${pnpm_version}")
"${pnpm[@]}" install --frozen-lockfile --ignore-scripts >/dev/null
node build.js >/dev/null

case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  esb_pkg=linux-x64 ;;
  Linux-aarch64) esb_pkg=linux-arm64 ;;
  Darwin-arm64)  esb_pkg=darwin-arm64 ;;
  Darwin-x86_64) esb_pkg=darwin-x64 ;;
  *) echo "unsupported build host: $(uname -s)-$(uname -m)" >&2; exit 1 ;;
esac
esbuild="$(ls -d node_modules/.pnpm/@esbuild+"${esb_pkg}"@*/node_modules/@esbuild/"${esb_pkg}"/bin/esbuild | head -1)"

# OpenSpec resolves its version and schemas relative to import.meta.url
# (../../package.json and ../../../schemas from dist/core/artifact-graph), so
# the single bundle keeps that depth inside the runtime tree.
bundle_rel="dist/core/artifact-graph/openspec.mjs"
# cli/index.js starts itself through its own "argv[1] is this module" guard,
# which is true inside the bundle, so the entry must only import it; calling
# runCli() as well would run every command twice.
printf "import './cli/index.js';\n" > dist/entry.mjs
rm -rf "$out"
mkdir -p "$out/$(dirname "$bundle_rel")"
"$esbuild" dist/entry.mjs --bundle --platform=node --format=esm --target=node20 --minify \
  --legal-comments=none --log-level=warning \
  --banner:js="import{createRequire as __cr}from'module';const require=__cr(import.meta.url);" \
  --outfile="$out/$bundle_rel"
cp -r schemas "$out/schemas"
cp package.json "$out/package.json"
cp package.json "$out/dist/package.json"

# Smoke test: the tree must run standalone, and one invocation must run the
# CLI exactly once (a doubled run prints the version twice).
smoke="$(node "$out/$bundle_rel" --version)"
if [[ "$smoke" != "$(python3 -c 'import json;print(json.load(open("package.json"))["version"])')" ]]; then
  echo "smoke test failed: --version printed: $smoke" >&2
  exit 1
fi

# Attribution for every production dependency that ends up in the bundle.
"${pnpm[@]}" licenses list --prod --json --long > licenses.json
python3 - "$out" <<'PY'
import glob, json, os, sys
out = sys.argv[1]
data = json.load(open("licenses.json"))
seen, lines, counts = set(), [], {}
for lic, pkgs in sorted(data.items()):
    for p in sorted(pkgs, key=lambda p: p["name"]):
        for path in p.get("paths", [])[:1]:
            files = sorted(f for f in glob.glob(os.path.join(path, "*")) if os.path.basename(f).upper().startswith(("LICENSE", "LICENCE", "COPYING")))
            key = (p["name"], tuple(p.get("versions", [])))
            if key in seen:
                continue
            seen.add(key)
            counts[lic] = counts.get(lic, 0) + 1
            body = open(files[0], encoding="utf-8", errors="replace").read().strip() if files else f"(no license file shipped; declared license: {lic})"
            lines.append(f"===== {p['name']} {','.join(p.get('versions', []))} ({lic}) =====\n{body}\n")
header = "Third-party licenses bundled in the OpenSpec runtime (production dependencies).\n" + \
         "Summary: " + ", ".join(f"{k}={v}" for k, v in sorted(counts.items())) + "\n\n"
open(os.path.join(out, "THIRD_PARTY_LICENSES.txt"), "w").write(header + "\n".join(lines))
PY
cp LICENSE "$out/LICENSE"

# Tree digest: must match runtimepayload.TreeDigest (sorted rel paths, each as
# "<path>\0<sha256(content)>\n", skipping runtime.build.yaml itself).
read -r tree_sha tree_bytes < <(python3 - "$out" <<'PY'
import hashlib, os, sys
root = sys.argv[1]
files = sorted(os.path.relpath(os.path.join(d, f), root).replace(os.sep, "/")
               for d, _, fs in os.walk(root) for f in fs)
h, size = hashlib.sha256(), 0
for rel in files:
    if rel == "runtime.build.yaml":
        continue
    data = open(os.path.join(root, rel), "rb").read()
    h.update(rel.encode() + b"\x00" + hashlib.sha256(data).hexdigest().encode() + b"\n")
    size += len(data)
print(h.hexdigest(), size)
PY
)
sha="$(sha256sum "$out/$bundle_rel" | cut -d' ' -f1)"
version="$(python3 -c 'import json;print(json.load(open("package.json"))["version"])')"
node_version="$(python3 -c 'import json;print(json.load(open("package.json"))["engines"]["node"])')"
cat > "$out/runtime.build.yaml" <<YAML
# Generated by scripts/build-openspec-runtime.sh - do not edit by hand.
upstream_repo: Fission-AI/OpenSpec
upstream_tag: ${tag}
upstream_commit: ${commit}
version: ${version}
requires_node: "${node_version}"
bundle: ${bundle_rel}
bundle_sha256: ${sha}
tree_sha256: ${tree_sha}
tree_bytes: ${tree_bytes}
YAML
echo "built OpenSpec ${version} (${tag} @ ${commit:0:12}) -> ${out}"
