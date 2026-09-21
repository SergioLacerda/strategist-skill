# Standalone runtime evidence

For local provider onboarding, keep the static and live dimensions separate:
`strategist provider validate <source>` may prove package/adapter contract shape,
but it reports `live_invocation: unknown` until an authorized runtime probe
succeeds. Use `strategist provider add <source> --slot refinement|execution` to
stage and bind a provider transactionally; discovery remains native Ranger.

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
- **Pins.** A private runtime whose component versions differ from the
  contract's `runtime.version` / `runtime.node_version` is rejected at install
  (`ranked_runtime_pin_mismatch`). A host `openspec` whose `--version` differs
  from the pin only produces the advisory `ranked_runtime_version_skew` (log at
  install, Ready-with-reason at check). `strategist version --build` shows a
  binary's commit, platform and whether a payload is embedded.
- **Ordinary builds** (`make build`, `make install-lite`, plain `go build`,
  `go test`) do not embed a payload. Without one, install resolves `openspec`
  from `PATH` and reports the cataloged `ranked_runtime_executable_missing`
  diagnostic when it is absent. `make install` and `make build-standalone`
  build with the payload (they fetch the pinned Node by digest on first use and
  need network and python); `make standalone-smoke` proves install and check
  succeed with an empty `PATH`.
- **Troubleshooting `ranked_runtime_executable_missing`.** On a client this
  almost always means the binary was built without the payload (typically
  `make build` or a `go build`, or a binary installed before `make install`
  started embedding it). Run `strategist version --build`: `runtime payload:
  none` confirms it. Install a release binary, or rebuild with
  `make install`.
- **Windows.** The payload paths and the Windows environment allow-list
  (`SystemRoot`, `TEMP`, `TMP`, `ComSpec`, `PATHEXT`) are covered by
  platform-neutral tests. Tests that fake an executable with a POSIX shell
  script are skipped on Windows (`testutil.RequirePOSIXShell`); the real exec
  path is proven there by `scripts/smoke-standalone-install.sh`, which the
  `test-windows` CI job runs against a payload build with an empty `PATH`.
  The job is blocking. On 2026-09-20 it passed on a GitHub `windows-latest`
  runner, including the smoke: a payload build installed and passed
  `strategist check` with an empty `PATH` and no OpenSpec on the machine. The
  Windows binary was also exercised under Wine earlier as supporting evidence.
  Not yet covered: macOS, and Windows on arm64 (both compile only).

## Known limitations and diagnostics

These describe the behavior of the current release. Each item names how to
recognise it; the hardening package
`20260920-standalone-runtime-hardening-sweep` tracks the fixes.

- **Silent install does not materialize the runtime.** `strategist install`
  without `--wizard` binds the refinement provider in `custom` mode. Even with an
  embedded payload it creates no `.strategist/weapon-runtime/`, and
  `strategist check` can still report `ready` on a machine with no `openspec`.
  Use `strategist install --wizard` and keep the pre-selected Ranked option for
  the refinement slot. Diagnostic: `.strategist/weapon-runtime/openspec-propose/`
  and `ranked-runtimes.yaml` exist only after a Ranked install.
- **`strategist upgrade` does not repair a stale ranked runtime.** After the
  binary changes a provider's certification digest (for example when the pinned
  runtime versions change), `strategist check` reports
  `ranked_runtime_digest_mismatch` and `strategist upgrade` still ends with
  "Upgrade complete". The only known repair is to rerun
  `strategist install --wizard`. Diagnostic: the message lists `expected=` and
  `observed=` digests and no remedy.
- **Ranked reason codes without a catalog entry.** Only
  `ranked_runtime_executable_missing`, `ranked_runtime_pin_mismatch` and
  `ranked_runtime_version_skew` have entries in `machine/errors.yaml`. The
  others (`binding_missing`, `state_missing`, `state_invalid`, `root_missing`,
  `root_mismatch`, `healthcheck_failed`, `contract_invalid`) surface only their
  raw detail text.
- **Windows binary name.** A cross-build of `go build -o bin/strategist` for
  Windows produced a file without `.exe`; `make build`, `make install` and
  `make build-standalone` use that name. It runs from Git Bash but is not
  resolved by `cmd` or PowerShell. Use a release binary
  (`strategist-windows-*.exe`) or rename the file to `strategist.exe`. Whether
  a native Windows build behaves the same has not been confirmed.
- **Healthcheck time limit.** Ranked healthchecks give the runtime 10 seconds.
  The Node cold path measured about 0.8 seconds on Linux; the first launch of a
  freshly extracted `node.exe` on Windows is unmeasured. A timeout appears as
  `ranked_runtime_healthcheck_failed`, indistinguishable from a failing command.
  Every `strategist check` starts the runtime once.
- **Reading a binary.** `strategist version --build` prints the commit,
  platform and whether the runtime payload is embedded. `runtime payload: none`
  on a client means the binary was built without `make install` /
  `make build-standalone`.
- **Development environment.** `golangci-lint` must be built with a Go at least
  as new as `go.mod` (the pre-commit hook otherwise stops with a Go-version
  error), and the local Go may be older than the `go.mod` patch level (the
  repository then switches toolchain automatically, so check `go version`
  inside the repository). Running `make` or `go` with a temporary `HOME` leaves
  a read-only module cache that `rm -rf` cannot delete and can exhaust a small
  `/tmp`. Any Go change alters the catalog `host_api_digest`; run
  `strategist plugins prepare-embedded` before committing.
