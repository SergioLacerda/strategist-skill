# Standalone runtime evidence

Standalone installation has four separate readiness dimensions. They must not
be collapsed into a single green `strategist check` result.

1. **Static readiness** — catalog, role affinity, provider risk and
   certification metadata are structurally valid.
2. **Persisted runtime** — `plugins.lock` and `ranked-runtimes.yaml` identify
   the selected slot/provider, certification digest and declared runtime.
3. **Live healthcheck** — the provider command runs from the physical
   `.strategist/openspec` directory and reports the containing `.strategist`
   directory as its semantic OpenSpec root. The comparison is independent of
   path spelling: a relative, dotted, trailing-slash or symlinked runtime root
   is resolved to its physical directory before it is compared with the
   absolute `root.path` OpenSpec reports. A different directory is still
   rejected (`ranked_runtime_root_mismatch`).
4. **Containment** — configuration is directly at
   `.strategist/openspec/config.yaml`; repository-root or nested
   `openspec/config.yaml` is not an accepted substitute, and provider
   subprocesses receive a runtime-scoped environment.

The declared runtime root (`runtime.root` in the provider contract) is a
canonical slash-separated path under `.strategist/`. Validation normalizes the
platform separator before checking cleanliness, so the same declaration is
accepted on Windows and Linux; absolute paths, `..` segments, drive letters and
non-canonical spellings (`./`, `//`, trailing slash) are rejected on every
platform.

The boundary is fail-closed: malformed healthcheck output, missing or
mismatched roots, invalid runtime contracts, digest mismatches, and escaped
runtime symlinks block readiness. The installer must not lazily initialize a
different root or silently fall back to another provider.

## Verification

Use isolated HOME/XDG directories and writable `GOCACHE`, `GOPATH`, and
coverage directories when running the hermetic integration suite. The CLI
harness passes an explicit environment allowlist; credentials, provider
configuration, and unrelated host variables are excluded unless a test adds
them deliberately.

The fake OpenSpec provider records cwd, arguments, provider-scoped HOME/XDG
values, and a write ledger. Successful writes must remain under the temporary
`.strategist` runtime and must include the direct `openspec/config.yaml`; the
workspace root and nested `.strategist/openspec/openspec` path are not valid
destinations. Host decoys are seeded and must remain byte-identical.

A passing static check without the persisted-runtime and live-healthcheck
evidence is not a Ranked invocation certification. The fixture proves process
containment and contract behavior; it is not certification of a live external
OpenSpec installation.
