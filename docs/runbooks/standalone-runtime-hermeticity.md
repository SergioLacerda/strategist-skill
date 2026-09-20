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
   **Residual (unverified on Windows):** behavior for 8.3 short names and
   drive-letter case is not yet exercised on a Windows runner; the
   same-directory fallback is the intended backstop. Owner: mission
   `20260920-drift-b-semantic-root-preflight`, task 1.1.
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

## Private OpenSpec runtime (Drift A resolution)

A release binary is built with `-tags strategist_payload` and carries the whole
runtime of the Ranked provider `openspec-propose`; a client installs and runs
it without OpenSpec, Node, `npm`, or anything on `PATH`.

- **What ships.** The prebuilt OpenSpec bundle lives in
  `external-skills-source/openspec-propose/runtime/` (one JavaScript file plus
  schemas and third-party licences, built by `scripts/build-openspec-runtime.sh`
  from the pinned upstream tag and never edited by hand). The pinned Node for
  each target is fetched at build time by `scripts/fetch-node-runtime.py`,
  verified against the digests in `runtime.lock.yaml`, and only the node
  executable and its licence are embedded.
- **Install.** `strategist install` verifies every component digest before
  writing anything, materializes the runtime under
  `.strategist/weapon-runtime/openspec-propose/`, runs `init` and
  `context --json` from it with a `PATH` limited to that directory, and records
  the launcher and each component's version and digest in `ranked-runtimes.yaml`.
  A missing, corrupt, or wrong-target payload fails the install with a full
  rollback and never falls back to a host executable.
- **Check.** When the state records a private runtime, `strategist check` runs
  the healthcheck through it and blocks with `ranked_runtime_executable_missing`
  if it is gone or with `ranked_runtime_state_invalid` if the recorded paths
  leave `weapon-runtime/`.
- **Certification.** The pinned OpenSpec and Node versions (`runtime.version`,
  `runtime.node_version` in the provider contract) are part of the certification
  digest. Workspaces installed before a runtime version change report
  `ranked_runtime_digest_mismatch` until `strategist install` is run again.
- **Ordinary builds** (`make build`, `go test`) do not embed a payload. Without
  one, install resolves `openspec` from `PATH` and reports the cataloged
  `ranked_runtime_executable_missing` diagnostic when it is absent (stage (a)).
  Use `make build-standalone` for a payload build and `make standalone-smoke`
  to prove install and check succeed with an empty `PATH`.
- **Windows.** The payload paths and the Windows environment allow-list
  (`SystemRoot`, `TEMP`, `TMP`, `ComSpec`, `PATHEXT`) are covered by
  platform-neutral tests. Tests that fake an executable with a POSIX shell
  script are skipped on Windows (`testutil.RequirePOSIXShell`); the real exec
  path is proven there by `scripts/smoke-standalone-install.sh`, which the
  `test-windows` CI job runs against a payload build with an empty `PATH`.
  That job is `continue-on-error` until a first green run on a real Windows
  runner. The Windows binary was also exercised under Wine (install, `init`
  through the embedded `node.exe`, and `check` all succeeded with no OpenSpec on
  the system), which is supporting evidence, not a substitute for the CI run.
