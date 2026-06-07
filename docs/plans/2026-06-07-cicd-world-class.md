# CI/CD World Class Engineering — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Paralelizar o pipeline de CI em 4 jobs independentes e fechar o supply chain com Dependabot para Go modules + SLSA L2 build provenance.

**Architecture:** Três arquivos modificados — `test.yml` reescrito com 4 jobs paralelos + concurrency group, `dependabot.yml` com segundo entry para `gomod`, `release.yml` com step de atestado SLSA após goreleaser. Sem nova lógica Go — mudanças puramente em YAML de CI.

**Tech Stack:** GitHub Actions, golangci-lint-action v9, actions/attest-build-provenance v4, Dependabot gomod ecosystem.

---

## Task 1: Dependabot para Go modules

**Files:**
- Modify: `.github/dependabot.yml`

**Contexto:** O `dependabot.yml` atual só rastreia `github-actions`. Go modules (`go.mod`) não recebem PRs automáticos de atualização.

**Step 1: Adicionar o entry gomod**

Abrir `.github/dependabot.yml`. O arquivo atual tem:

```yaml
version: 2

updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "06:00"
    commit-message:
      prefix: "chore(deps)"
    groups:
      actions:
        patterns: ["*"]
```

Substituir pelo conteúdo abaixo (adiciona segundo entry, mantém o existente intacto):

```yaml
version: 2

updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "06:00"
    commit-message:
      prefix: "chore(deps)"
    groups:
      actions:
        patterns: ["*"]

  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "06:00"
    commit-message:
      prefix: "chore(deps)"
    groups:
      go-deps:
        patterns: ["*"]
```

**Step 2: Verificar YAML válido**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/dependabot.yml'))" && echo "OK"
```

Esperado: `OK`

**Step 3: Commit**

```bash
git add .github/dependabot.yml
git commit -m "chore: add dependabot tracking for Go modules"
```

---

## Task 2: Reescrever test.yml com 4 jobs paralelos

**Files:**
- Modify: `.github/workflows/test.yml`

**Contexto:** O arquivo atual tem 1 job com 13 steps sequenciais. Qualquer falha no início (ex: lint) bloqueia todos os steps restantes por minutos. A reescrita paraleliza em 4 jobs independentes que rodam simultaneamente.

**Novos SHAs necessários:**

| Action | Versão | SHA |
|--------|--------|-----|
| `golangci/golangci-lint-action` | v9.2.1 | `db582008a42febd596419635a5abc9d9815daa9c` |

As demais actions (`checkout`, `setup-go`) permanecem com os SHAs atuais.

**Step 1: Substituir o conteúdo completo de test.yml**

```yaml
name: Test

on:
  push:
    branches: ["**"]
  pull_request:

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}

permissions:
  contents: read

jobs:
  lint:
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v5
        with:
          go-version-file: "go.mod"
          cache: true

      - name: Format check
        run: test -z "$(gofmt -l .)"

      - name: Module hygiene
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum
          go mod verify

      - name: Vet
        run: go vet ./...

      - name: Lint
        uses: golangci/golangci-lint-action@db582008a42febd596419635a5abc9d9815daa9c # v9.2.1
        with:
          version: v2.1.6

      - name: Build
        run: go build ./...

  test:
    runs-on: ubuntu-latest
    timeout-minutes: 15

    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v5
        with:
          go-version-file: "go.mod"
          cache: true

      - name: Lightweight tests
        run: make test-lite

      - name: Test (with race detector)
        run: go test -race $(go list ./... | grep -v '/testutil')

      - name: Integration tests
        run: go test -tags=integration -race ./tests/integration/...

      - name: Coverage gate (>90% per package)
        run: |
          fail=0
          # internal/domain excluded: pure type declarations, no executable statements.
          for pkg in internal/stale internal/compile internal/install internal/embed cmd/strategist; do
            pct=$(go test -coverprofile=/tmp/cov.out -coverpkg=./$pkg/... ./$pkg/... 2>/dev/null \
                  | grep -o '[0-9.]*%' | tail -1 | tr -d '%')
            echo "$pkg: ${pct}%"
            ok=$(awk -v p="$pct" 'BEGIN{print (p+0 >= 90)}')
            if [ "$ok" != "1" ]; then
              echo "  FAIL: ${pct}% < 90%"
              fail=1
            fi
          done
          exit $fail

  security:
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v5
        with:
          go-version-file: "go.mod"
          cache: true

      - name: Vulnerability check
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
          govulncheck ./...

  validate:
    runs-on: ubuntu-latest
    timeout-minutes: 10

    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v5
        with:
          go-version-file: "go.mod"
          cache: true

      - name: Fixture and schema validation
        run: |
          pip install pyyaml --quiet
          bash tests/spec/run-tests.sh

      - name: Analysis refined structure gate
        run: make analysis-structure-gate

      - name: Docs governance gate
        run: make docs-governance-gate
```

**Step 2: Verificar YAML válido**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml'))" && echo "OK"
```

Esperado: `OK`

**Step 3: Confirmar que os 4 jobs estão presentes**

```bash
grep "^  [a-z]*:$" .github/workflows/test.yml
```

Esperado:
```
  lint:
  test:
  security:
  validate:
```

**Step 4: Commit**

```bash
git add .github/workflows/test.yml
git commit -m "ci: parallelize test workflow into 4 independent jobs

- lint: format, mod hygiene, vet, golangci-lint-action, build
- test: unit + race + integration + coverage gate
- security: govulncheck
- validate: fixture/schema, analysis gate, docs governance gate
- add concurrency group with cancel-in-progress for non-main branches
- add timeout-minutes per job
- replace go install golangci-lint with golangci-lint-action (cached)"
```

**Step 5: Push e verificar na UI do GitHub**

```bash
git push
```

Abrir a tab Actions no GitHub. Verificar que o run mostra 4 jobs rodando em paralelo (não em sequência). Todos devem aparecer simultaneamente na visualização de jobs.

---

## Task 3: SLSA L2 Build Provenance no release.yml

**Files:**
- Modify: `.github/workflows/release.yml`

**Contexto:** O release atual gera binários via goreleaser e um SBOM CycloneDX. Falta o atestado de provenance — quem, quando, e qual workflow gerou os binários. O `actions/attest-build-provenance` gera isso via OIDC sem infra extra.

**SHA necessário:**

| Action | Versão | SHA |
|--------|--------|-----|
| `actions/attest-build-provenance` | v4.1.0 | `a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32` |

**Step 1: Substituir o conteúdo completo de release.yml**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write
  id-token: write      # OIDC token para SLSA signing
  attestations: write  # publicar atestado no GitHub

jobs:
  release:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
        with:
          fetch-depth: 0

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c # v5
        with:
          go-version-file: "go.mod"
          cache: true

      - name: Run goreleaser
        uses: goreleaser/goreleaser-action@9c156ee5a69193a9a85c4b91df1644a3a6e8f5ab # v6
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Generate SBOM
        uses: anchore/sbom-action@e22c389904149dbc22b58101806040fa8d37a610 # v0.24.0
        with:
          artifact-name: strategist-${{ github.ref_name }}-sbom.cdx.json
          output-file: strategist-sbom.cdx.json
          format: cyclonedx-json
          upload-release-assets: true
          github-token: ${{ secrets.GITHUB_TOKEN }}

      - name: Attest build provenance
        uses: actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32 # v4.1.0
        with:
          subject-path: dist/strategist_*
```

**Nota sobre `subject-path`:** O goreleaser coloca os binários em `dist/`. O glob `dist/strategist_*` cobre todos os binários cross-platform gerados. Se o nome do binário no `.goreleaser.yml` for diferente, ajuste o glob.

**Step 2: Verificar nome do binário no goreleaser**

```bash
grep -A3 "builds:" .goreleaser.yml 2>/dev/null | head -10
# ou
ls dist/ 2>/dev/null | head -10
```

Se o prefixo for diferente de `strategist_`, ajustar o `subject-path` no step acima.

**Step 3: Verificar YAML válido**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "OK"
```

Esperado: `OK`

**Step 4: Confirmar as 3 permissões presentes**

```bash
grep -E "contents:|id-token:|attestations:" .github/workflows/release.yml
```

Esperado:
```
  contents: write
  id-token: write
  attestations: write
```

**Step 5: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add SLSA L2 build provenance attestation to release workflow

actions/attest-build-provenance generates a signed SLSA provenance
statement for goreleaser binaries via GitHub OIDC. Complements the
existing CycloneDX SBOM (what was built) with provenance (how and
when it was built).

Verify after next release:
  gh attestation verify <binary> --repo SergioLacerda/strategist-skill"
```

---

## Task 4: Branch Protection — Required Status Checks

**Files:** Nenhum arquivo — configuração via GitHub UI ou `gh` CLI.

**Contexto:** Com 4 jobs no lugar de 1, o branch protection de `main` precisa ser atualizado. Se estava exigindo o job antigo `test`, agora precisa exigir os 4 novos: `lint`, `test`, `security`, `validate`.

**Step 1: Verificar configuração atual via CLI**

```bash
gh api repos/SergioLacerda/strategist-skill/branches/main/protection \
  --jq '.required_status_checks.contexts'
```

Se retornar erro 404, branch protection ainda não está configurada — pular para o Step 3.

**Step 2: Atualizar via CLI**

```bash
gh api repos/SergioLacerda/strategist-skill/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"contexts":["lint","test","security","validate"]}' \
  --field enforce_admins=false \
  --field required_pull_request_reviews=null \
  --field restrictions=null
```

**Step 3: Verificar via UI (alternativa)**

Settings → Branches → Branch protection rules → `main` → Required status checks → adicionar: `lint`, `test`, `security`, `validate` → remover: `test` (o antigo job unificado, se existia).

---

## Verificação Final

Após os 4 tasks commitados e pushed:

**1. Abrir um PR de teste (branch develop → main)**

Verificar na tab Actions que o workflow `Test` mostra 4 jobs rodando em paralelo.

**2. Verificar tempo total**

O job mais lento (`test`) deve completar em ~2–3min. Compare com o tempo anterior.

**3. Verificar que golangci-lint-action usa cache**

No log do job `lint`, procurar: `Restored golangci-lint cache` ou similar — confirma que o cache está ativo.

**4. Na próxima semana**

O Dependabot deve abrir PRs agrupados cobrindo Go modules (`chore(deps): bump go-deps group`).

**5. No próximo release tag**

Após `git tag v<next> && git push --tags`, o workflow `Release` deve completar e mostrar um atestado de provenance na aba "Attestations" do release no GitHub.

Verificar: `gh attestation verify <binário-baixado> --repo SergioLacerda/strategist-skill`
