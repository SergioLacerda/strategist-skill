#!/usr/bin/env python3
"""Build-time step: fetch the pinned Node for one or more targets.

Reads external-skills-source/openspec-propose/runtime.lock.yaml, downloads each
official archive, verifies its sha256 against the lock, and repackages ONLY the
node executable and LICENSE (no npm, headers or symlinks) into
internal/runtimepayload/bundled/nodepayload/<target>/ for the strategist_payload build.
Output is deterministic (fixed mtimes and ownership).

Usage: scripts/fetch-node-runtime.py <target>...   (e.g. linux-amd64 windows-amd64)
       scripts/fetch-node-runtime.py --host        (the current machine's target)
       scripts/fetch-node-runtime.py --all         (every target pinned in the lock; release builds)
Environment: NODE_RUNTIME_CACHE overrides the download cache (default .cache/node-runtime).
Offline builds: pre-seed the cache with the archives named in the lock.
"""
import gzip, hashlib, io, os, platform, re, sys, tarfile, urllib.request, zipfile

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
LOCK = os.path.join(ROOT, "external-skills-source/openspec-propose/runtime.lock.yaml")
OUT = os.path.join(ROOT, "internal/runtimepayload/bundled/nodepayload")
CACHE = os.environ.get("NODE_RUNTIME_CACHE", os.path.join(ROOT, ".cache/node-runtime"))


def parse_lock(path: str) -> dict:
    """Minimal reader for runtime.lock.yaml (the stdlib has no YAML parser and
    build hosts are not guaranteed to have PyYAML). It understands exactly the
    shape of that file and rejects anything else."""
    node, targets, current = {}, {}, None
    for raw in open(path, encoding="utf-8"):
        line = raw.split("#", 1)[0].rstrip()
        if not line.strip():
            continue
        indent = len(line) - len(line.lstrip())
        key, _, value = line.strip().partition(":")
        value = value.strip()
        if indent == 2 and key in ("version", "base_url") and value:
            node[key] = value
        elif indent == 4 and not value:
            current = key
            targets[current] = {}
        elif indent == 6 and current and key in ("file", "sha256") and value:
            targets[current][key] = value
    node["targets"] = targets
    if not node.get("version") or not node.get("base_url") or not targets:
        sys.exit(f"{path}: not a runtime lock this tool understands")
    for name, spec in targets.items():
        if not re.fullmatch(r"[0-9a-f]{64}", spec.get("sha256", "")) or not spec.get("file"):
            sys.exit(f"{path}: target {name} needs file and a 64-hex sha256")
    return {"node": node}


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def download(url: str, want: str, dest: str) -> bytes:
    if os.path.exists(dest):
        data = open(dest, "rb").read()
        if sha256(data) == want:
            return data
        os.remove(dest)  # stale or corrupt cache entry
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    with urllib.request.urlopen(url, timeout=120) as resp:  # noqa: S310 - pinned https URL from the lock
        data = resp.read()
    got = sha256(data)
    if got != want:
        sys.exit(f"digest mismatch for {url}: expected {want}, got {got}")
    open(dest, "wb").write(data)
    return data


def unix_members(data: bytes):
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:xz") as tf:
        found = {}
        for m in tf.getmembers():
            parts = m.name.split("/")
            if m.isfile() and len(parts) == 3 and parts[1:] == ["bin", "node"]:
                found["bin/node"] = tf.extractfile(m).read()
            elif m.isfile() and len(parts) == 2 and parts[1] == "LICENSE":
                found["LICENSE"] = tf.extractfile(m).read()
    return found


def windows_members(data: bytes):
    found = {}
    with zipfile.ZipFile(io.BytesIO(data)) as zf:
        for name in zf.namelist():
            parts = name.split("/")
            if len(parts) == 2 and parts[1] in ("node.exe", "LICENSE"):
                found[parts[1]] = zf.read(name)
    return found


def pack_tar_gz(members: dict) -> bytes:
    raw = io.BytesIO()
    with tarfile.open(fileobj=raw, mode="w", format=tarfile.PAX_FORMAT) as tf:
        for name in sorted(members):
            info = tarfile.TarInfo(name)
            info.size, info.mtime, info.uid, info.gid, info.uname, info.gname = len(members[name]), 0, 0, 0, "", ""
            info.mode = 0o755 if name == "bin/node" else 0o644
            tf.addfile(info, io.BytesIO(members[name]))
    out = io.BytesIO()
    with gzip.GzipFile(fileobj=out, mode="wb", mtime=0, compresslevel=9) as gz:
        gz.write(raw.getvalue())
    return out.getvalue()


def pack_zip(members: dict) -> bytes:
    out = io.BytesIO()
    with zipfile.ZipFile(out, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for name in sorted(members):
            info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
            info.external_attr = 0o644 << 16
            zf.writestr(info, members[name], zipfile.ZIP_DEFLATED, 9)
    return out.getvalue()


def host_target() -> str:
    os_name = {"Linux": "linux", "Darwin": "darwin", "Windows": "windows"}[platform.system()]
    arch = {"x86_64": "amd64", "AMD64": "amd64", "aarch64": "arm64", "arm64": "arm64"}[platform.machine()]
    return f"{os_name}-{arch}"


def build(target: str, lock: dict) -> None:
    node = lock["node"]
    spec = node["targets"].get(target)
    if spec is None:
        sys.exit(f"target {target!r} is not pinned in runtime.lock.yaml")
    data = download(f"{node['base_url']}/{spec['file']}", spec["sha256"], os.path.join(CACHE, spec["file"]))
    windows = target.startswith("windows-")
    members = windows_members(data) if windows else unix_members(data)
    want = ("node.exe", "LICENSE") if windows else ("bin/node", "LICENSE")
    missing = [m for m in want if m not in members]
    if missing:
        sys.exit(f"{spec['file']}: missing {missing} in archive")
    packed = pack_zip(members) if windows else pack_tar_gz(members)
    name = "node.zip" if windows else "node.tar.gz"
    out_dir = os.path.join(OUT, target)
    os.makedirs(out_dir, exist_ok=True)
    for stale in os.listdir(out_dir):
        os.remove(os.path.join(out_dir, stale))
    open(os.path.join(out_dir, name), "wb").write(packed)
    info = {
        "version": node["version"], "target": target, "file": name,
        "format": "zip" if windows else "tar.gz",
        "sha256": sha256(packed), "size": len(packed),
        "source_file": spec["file"], "source_sha256": spec["sha256"],
    }
    with open(os.path.join(out_dir, "node.info.yaml"), "w") as f:
        f.write("# Generated by scripts/fetch-node-runtime.py - do not edit.\n")
        for key in sorted(info):
            f.write(f"{key}: {info[key]}\n")
    print(f"node {node['version']} {target}: {len(packed)} bytes -> {os.path.relpath(out_dir, ROOT)}")


def main() -> None:
    args = sys.argv[1:]
    if not args:
        sys.exit(__doc__)
    lock = parse_lock(LOCK)
    if args == ["--host"]:
        targets = [host_target()]
    elif args == ["--all"]:
        targets = sorted(lock["node"]["targets"])
    else:
        targets = args
    for target in targets:
        build(target, lock)


if __name__ == "__main__":
    main()
