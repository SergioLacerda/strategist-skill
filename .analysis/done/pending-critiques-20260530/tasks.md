# Tasks — Pending Critiques Sprint 1
**Mission ID:** pending-critiques-20260530
**Phase:** Sniper (controlled)

## T1 — Fix release.yml go-version
- **File:** `.github/workflows/release.yml`
- **Change:** `go-version: "1.22"` → `go-version-file: "go.mod"`

## T2 — Create validate.go
- **File:** `cmd/strategist/validate.go`
- **Pattern:** same `var validateRoot string` injectable variable as other commands
- **Checks:**
  1. `active.yaml` — exists, valid YAML, has `mode` and `roles_config`
  2. `personas/` — directory exists, has ≥1 `.yaml`
  3. Each `personas/*.yaml` — has `tone_directive` and `phase_labels`
  4. Each `roles/*.yaml` — has `discovery`, `refinement`, `execution` keys
  5. `knowledge.index.yaml` — if exists, valid YAML
- **Print:** `[Strategist] validate OK — X check(s) passed` on success

## T3 — Add TestValidateCmd_* to cmd_test.go
- `TestValidateCmd_Success` — MinimalRoot + minimal valid personas/roles
- `TestValidateCmd_MissingRoot` — non-existent dir
- `TestValidateCmd_MissingActiveYAML` — dir exists but no active.yaml
- `TestValidateCmd_InvalidMode` — active.yaml with mode: invalid_mode
- `TestValidateCmd_MissingSlot` — role file without `discovery` key
- `TestValidateCmd_DefaultRoot` — empty validateRoot triggers default

## T4 — Register validateCmd in root.go

## T5 — go test -race + coverage gate
