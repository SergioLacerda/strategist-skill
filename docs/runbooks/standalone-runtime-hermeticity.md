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

## Embedded OpenSpec runtime (Drift A resolution)

Every build path (`go install`, `make build`, release archives, and native
Windows builds) carries the same digest-verified OpenSpec bundle. Ranked
execution requires the client's host Node.js `>=20.19.0`; OpenSpec and `npm`
are never installed separately.

- **What ships.** The prebuilt OpenSpec bundle lives in
  `external-skills-source/openspec-propose/runtime/` (one JavaScript file plus
  schemas and third-party licences, built by `scripts/build-openspec-runtime.sh`
  from the pinned upstream tag and never edited by hand). Node archives are not
  fetched or embedded by the Strategist build.
- **Install.** `strategist install` always verifies and materializes the
  embedded OpenSpec bundle under `.strategist/weapon-runtime/openspec-propose/`.
  It resolves `node` once from the host PATH, requires Node.js `>=20.19.0`,
  records its absolute path, then runs the internal bundle with that path. It
  never resolves `openspec` or `npm` from PATH.
- **Check.** When the state records a Ranked runtime, `strategist check` runs
  the healthcheck through it after re-verifying the bundle digest, and blocks
  with `ranked_runtime_executable_missing` if Node is gone (see the reason
  codes below for legacy records, unsafe paths and altered bundles).
- **Certification.** The embedded OpenSpec version (`runtime.version` in the
  provider contract) is part of the certification digest. Workspaces installed before a runtime version change report
  `ranked_runtime_digest_mismatch` until `strategist install` is run again.
- **Pins.** The embedded OpenSpec version must match the provider contract at
  install (`ranked_runtime_pin_mismatch`). The host Node is checked against the
  `>=20.19.0` floor on every healthcheck; an optional certification pin is
  reported as non-blocking version skew.
- **Builds** (`go install`, `make build`, plain `go build`, and release
  archives) all embed OpenSpec and require only a supported host Node. Clients
  do not install OpenSpec, npm, curl, or a second global CLI.
- **Troubleshooting `ranked_runtime_executable_missing`.** On a client this
  means the Node executable recorded at install is absent. Install Node.js
  `>=20.19.0` and rerun `strategist install --wizard`; no global OpenSpec/npm
  installation is required. `strategist version --build` reports the embedded
  OpenSpec version and host-Node minimum.
- **Windows.** The recorded absolute Node path and slash-normalized private
  script are covered by platform-neutral tests. Git Bash and native Windows
  smoke tests keep OpenSpec absent from PATH while allowing the host Node
  directory, proving the same runtime contract on the client.

## Known limitations and diagnostics

These describe the behavior of the current release. Each item names how to
recognise it; the hardening package
`20260920-standalone-runtime-hardening-sweep` tracks the fixes.

- **Silent install does not materialize the runtime.** `strategist install`
  without `--wizard` binds the refinement provider in `custom` mode. Even though
  every binary embeds the OpenSpec bundle, it creates no `.strategist/weapon-runtime/`, and
  `strategist check` can still report `ready` on a machine with no `openspec`.
  Use `strategist install --wizard` and keep the pre-selected Ranked option for
  the refinement slot. Diagnostic: `.strategist/weapon-runtime/openspec-propose/`
  and `ranked-runtimes.yaml` exist only after a Ranked install.
- **Stale certification digest.** After the binary changes a provider's
  certification digest (for example when the embedded OpenSpec version
  changes), `strategist check` reports `ranked_runtime_digest_mismatch` with
  `expected=`, `observed=` and a `remedy=`. `strategist upgrade` re-records and
  re-materializes the runtime from the installed catalog; `strategist install
  --wizard` also repairs it.
- **Reason-code catalog.** Every `ranked_runtime_*` reason code `strategist
  check` emits has an entry with an action in `machine/errors.yaml`; a spec
  test fails when a new code is emitted without one.
- **Runtime-state reason codes.** `strategist check` reports the Ranked
  runtime under the `dependencies` readiness dimension. A record from an earlier
  release (any schema other than `strategist-ranked-runtime/v2`, including a
  private-Node `"mode": "payload"` record) blocks with
  `ranked_runtime_state_legacy`; `strategist upgrade` rewrites it with the
  absolute host Node and removes the private Node directory. A relative Node
  path, a launcher outside `weapon-runtime/<provider>/openspec/` or a record
  without the OpenSpec bundle digest blocks with `ranked_runtime_state_invalid`.
- **Bundle integrity.** Before running Node, check recomputes the tree digest
  of `weapon-runtime/<provider>/openspec/` and compares it with the digest
  recorded at install: an absent bundle or launcher blocks with
  `ranked_runtime_bundle_missing`, any changed, added or removed file with
  `ranked_runtime_bundle_tampered`, and the bundle is not executed. Repair with
  `strategist upgrade` or `strategist install --wizard`.
- **Windows binary name.** A cross-build of `go build -o bin/strategist` for
  Windows produced a file without `.exe`; `make build`, `make install` and
  `make build-standalone` use that name. It runs from Git Bash but is not
  resolved by `cmd` or PowerShell. Use a release binary
  (`strategist-windows-*.exe`) or rename the file to `strategist.exe`. Whether
  a native Windows build behaves the same has not been confirmed.
- **Healthcheck time limit.** Ranked healthchecks give the runtime 10 seconds.
  The Node cold path measured about 0.8 seconds on Linux; the host Node cold
  start on Windows is unmeasured. A timeout appears as
  `ranked_runtime_healthcheck_failed`, indistinguishable from a failing command.
  Every `strategist check` starts the runtime once.
- **Reading a binary.** `strategist version --build` prints the commit,
  platform and the line
  `runtime: embedded OpenSpec <version>; host Node >=20.19.0 required`. No
  build embeds Node.
- **Development environment.** `golangci-lint` must be built with a Go at least
  as new as `go.mod` (the pre-commit hook otherwise stops with a Go-version
  error), and the local Go may be older than the `go.mod` patch level (the
  repository then switches toolchain automatically, so check `go version`
  inside the repository). Running `make` or `go` with a temporary `HOME` leaves
  a read-only module cache that `rm -rf` cannot delete and can exhaust a small
  `/tmp`. Any Go change alters the catalog `host_api_digest`; run
  `strategist plugins prepare-embedded` before committing.
