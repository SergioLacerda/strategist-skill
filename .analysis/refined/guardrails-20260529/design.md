# Design: Guardrails em 3 Sprints
**Mission ID:** guardrails-20260529
**Date:** 2026-05-29

---

## Sprint 1 — CI + Linters

### 1.1 Edição: `.github/workflows/test.yml`

Inserir quatro novos steps **antes** do step `Build` existente, nesta ordem:

```yaml
      - name: Format check
        run: test -z "$(gofmt -l .)"

      - name: Module hygiene
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum
          go mod verify

      - name: Vulnerability check
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
```

**Decisão de ordenação:** Format e module hygiene antes de lint/test — falham rápido com custo zero.
`govulncheck` após tests porque é o step mais lento e só faz sentido quando o código está correto.

**Decisão sobre go-version:** manter `"1.22"` no `setup-go` (não alterar neste sprint).
`govulncheck` exige Go ≥ 1.18 — compatível.

### 1.2 Edição: `.golangci.yaml`

Adicionar à seção `linters.enable`:
```yaml
    - misspell      # typos em comentários e strings
    - dupl          # blocos duplicados (threshold padrão: 150 tokens)
    - unconvert     # conversões desnecessárias
    - ineffassign   # atribuições cujo resultado nunca é lido
    - gocritic      # análise estática complementar ao revive
    - contextcheck  # context.Background() em service/usecase
```

Adicionar à seção `linters.settings`:
```yaml
    contextcheck:
      # NewInstaller.Install usa context.Background() como bootstrap CLI — intencional.
      # Qualquer uso dentro de lógica de negócio deve ser sinalizado.
      # Sem exclusão de arquivo: o linter vai reportar installer.go:99.
      # A exclusão é feita na seção exclusions para ser explícita.
    dupl:
      threshold: 100  # tokens; padrão 150 é muito permissivo para funções pequenas
```

Adicionar à seção `exclusions.rules`:
```yaml
      - path: internal/install/installer\.go
        linters:
          - contextcheck
        text: "passes a context to"
```

**Decisão sobre dupl threshold:** os helpers `writeGzJSON`/`readGzJSON` têm ~20-30 linhas cada.
Com threshold 100 tokens eles vão ser flagrados. Isso é comportamento desejado — deixar o linter
apontar a duplicação para decisão consciente de refatorar ou suprimir com `//nolint:dupl`.

### 1.3 Edição: `Makefile`

Adicionar target após `lint`:
```makefile
vuln:
	govulncheck ./...
```

Adicionar `vuln` à linha `.PHONY`:
```makefile
.PHONY: build test lint vuln cover cover-gate cover-html install-local release clean
```

---

## Sprint 2 — Enforcement Arquitetural

### 2.1 Edição: `.golangci.yaml` — depguard

Adicionar à seção `linters.enable`:
```yaml
    - depguard
```

Adicionar à seção `linters.settings`:
```yaml
    depguard:
      rules:
        domain-isolation:
          list-mode: lax  # allow by default, deny explicitly
          deny:
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/compile"
              desc: "domain must not import compile"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/install"
              desc: "domain must not import install"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/stale"
              desc: "domain must not import stale"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/embed"
              desc: "domain must not import embed"
          files:
            - "**/internal/domain/**"
        compile-lateral:
          list-mode: lax
          deny:
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/install"
              desc: "compile must not import install"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/stale"
              desc: "compile must not import stale"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/embed"
              desc: "compile must not import embed"
          files:
            - "**/internal/compile/**"
        install-lateral:
          list-mode: lax
          deny:
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/compile"
              desc: "install must not import compile"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/stale"
              desc: "install must not import stale"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/embed"
              desc: "install must not import embed"
          files:
            - "**/internal/install/**"
        stale-lateral:
          list-mode: lax
          deny:
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/compile"
              desc: "stale must not import compile"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/install"
              desc: "stale must not import install"
            - pkg: "github.com/SergioLacerda/strategist-skill/internal/embed"
              desc: "stale must not import embed"
          files:
            - "**/internal/stale/**"
```

**Decisão de estratégia:** `list-mode: lax` (allow-by-default + deny-list) é mais estável que
`strict` (deny-by-default). Com strict qualquer nova dependência padrão (fmt, os) precisa ser
explicitamente permitida — manutenção cara. O lax captura violações arquiteturais sem ruído.

**Decisão sobre embed:** `internal/embed` não tem imports internos hoje. Não precisa de regra
de lateral isolation — se adquirir dependências internas no futuro, `depguard` vai pegar.

### 2.2 Edição: `internal/domain/architecture_test.go`

Renomear o arquivo para `internal/domain/arch_test.go` e expandir
`TestDomainIsolation` com uma segunda função que valida cruzamentos laterais:

```go
// TestLateralIsolation verifica que pacotes internos não importam uns aos outros.
// Cada pacote de negócio deve depender apenas de internal/domain — nunca de um par.
func TestLateralIsolation(t *testing.T) {
    t.Parallel()

    cases := []struct {
        pkg      string
        forbidden []string
    }{
        {
            pkg: "github.com/SergioLacerda/strategist-skill/internal/compile",
            forbidden: []string{
                "github.com/SergioLacerda/strategist-skill/internal/install",
                "github.com/SergioLacerda/strategist-skill/internal/stale",
                "github.com/SergioLacerda/strategist-skill/internal/embed",
            },
        },
        {
            pkg: "github.com/SergioLacerda/strategist-skill/internal/install",
            forbidden: []string{
                "github.com/SergioLacerda/strategist-skill/internal/compile",
                "github.com/SergioLacerda/strategist-skill/internal/stale",
                "github.com/SergioLacerda/strategist-skill/internal/embed",
            },
        },
        {
            pkg: "github.com/SergioLacerda/strategist-skill/internal/stale",
            forbidden: []string{
                "github.com/SergioLacerda/strategist-skill/internal/compile",
                "github.com/SergioLacerda/strategist-skill/internal/install",
                "github.com/SergioLacerda/strategist-skill/internal/embed",
            },
        },
    }

    for _, tc := range cases {
        tc := tc
        t.Run(tc.pkg, func(t *testing.T) {
            t.Parallel()
            out, err := exec.Command("go", "list", "-deps", tc.pkg).CombinedOutput()
            if err != nil {
                t.Fatalf("go list -deps failed: %v\n%s", err, out)
            }
            deps := string(out)
            for _, forbidden := range tc.forbidden {
                if strings.Contains(deps, forbidden) {
                    t.Errorf("%s must not import %s", tc.pkg, forbidden)
                }
            }
        })
    }
}
```

**Decisão sobre local do arquivo:** manter em `internal/domain/` (não criar `tests/arch_test.go`)
porque os testes de arquitetura são ownership do pacote domain — o pacote que define os contratos
é quem deve guardá-los.

### 2.3 Criação: `strategist/contracts/architecture-rules.yaml`

```yaml
id: architecture-dependency-direction
version: "1.0"
type: mandate
severity: high

description: >
  Define a direção permitida de imports entre pacotes internos.
  Violações indicam acoplamento indevido e devem ser bloqueadas no CI.

allowed_directions:
  - from: cmd/strategist
    to: [internal/compile, internal/install, internal/stale, internal/embed]
    rationale: cmd é o ponto de wiring; pode importar qualquer pacote interno
  - from: internal/*
    to: [internal/domain]
    rationale: todos os pacotes de negócio dependem dos tipos e contratos do domain

forbidden_directions:
  - from: internal/domain
    to: [internal/compile, internal/install, internal/stale, internal/embed]
    reason: domain é camada pura — sem dependências de infraestrutura
  - from: internal/compile
    to: [internal/install, internal/stale, internal/embed]
    reason: pacotes de negócio não devem se acoplar lateralmente
  - from: internal/install
    to: [internal/compile, internal/stale, internal/embed]
    reason: pacotes de negócio não devem se acoplar lateralmente
  - from: internal/stale
    to: [internal/compile, internal/install, internal/embed]
    reason: pacotes de negócio não devem se acoplar lateralmente

enforcement:
  - tool: depguard (golangci-lint) — bloqueia no CI
  - tool: TestLateralIsolation (go test) — bloqueia no CI
```

---

## Sprint 3 — Governance + Skill Root Fix

### 3.1–3.3 Criação: `strategist/contracts/`

Três novos arquivos de mandate no diretório `strategist/contracts/`:

**`no-hack-without-evidence.md`:**
```markdown
# NO HACK WITHOUT EVIDENCE
id: no-hack-without-evidence
severity: high

Proibido por padrão — qualquer exceção exige os 5 itens obrigatórios:
1. diagnosis (o que foi investigado)
2. evidence (por que a abordagem normal não funciona)
3. explicit trade-off (o que se perde com o hack)
4. temporary marker (// HACK: <reason> com issue/task associada)
5. follow-up task (issue registrada para resolver a raiz)

Comportamentos proibidos sem evidência:
- suprimir erros sem diagnose
- enfraquecer testes para passar
- recover() ou error handling silencioso genérico
- `_ = err`
- alterar contratos públicos sem escalação
- adicionar abstrações sem evidência
- bypassar camadas arquiteturais
- desabilitar linters/testes/checks
- introduzir estado global mutável
- adicionar dependências sem aprovação
```

**`test-integrity.md`:**
```markdown
# TEST INTEGRITY
id: test-integrity
severity: high

Proibido modificar testes para fazer o código passar.
A ordem é: código se adapta ao teste, nunca o contrário.

Proibido:
- enfraquecer assertion para passar (assert.True → assert.NotNil)
- remover caso de teste sem justificativa documentada
- atualizar golden/snapshot sem diff explicado
- teste que não falha quando o comportamento que testa quebra
- teste dependente de ordem de execução
- sleep arbitrário em teste (usar testify Eventually ou channels)
- mock que torna o teste insensível a mudanças reais de comportamento
```

**`scope-locking.md`:**
```markdown
# SCOPE LOCKING
id: scope-locking
severity: medium

Toda mudança deve declarar escopo antes de iniciar.
Expansões de escopo descobertas durante execução requerem nova aprovação.

Regras:
- Sniper executa apenas o que está em tasks.md aprovado
- Qualquer arquivo fora do escopo declarado requer pausa + mini approval
- Melhorias de oportunidade vão para um novo item em todo/, não são executadas inline
- "enquanto estou aqui" é scope expansion — requer gate
```

### 3.4 Fix: Skill Root Resolution

**Problema:** `~/.claude/skills/strategist/SKILL.md` aponta para `source: ~/.strategist/`.
Esse path não existe — `strategist install` instala em `<target>/.strategist/`, não em `~/.strategist/`.

**Solução:** adicionar um comando `strategist install-global` (ou flag `--global`) que instala
os defaults em `~/.strategist/`. Quando invocado globalmente (fora de um projeto), o agente
resolve o skill root de `~/.strategist/` em vez de falhar silenciosamente.

**Arquivos afetados:**
- `cmd/strategist/main.go` ou arquivo de comandos Cobra — novo subcomando `install-global`
- `internal/install/installer.go` — reutilizar `Install()` com target `os.UserHomeDir()`
- `~/.claude/skills/strategist/SKILL.md` — atualizado automaticamente pelo `install-global`

**Comportamento esperado pós-fix:**
```
strategist install-global
→ extrai defaults em ~/.strategist/
→ atualiza ~/.claude/skills/strategist/SKILL.md com source: ~/.strategist
→ emite: [Strategist] global install complete — skill root: ~/.strategist/
```

**Alternativa descartada:** symlink `~/.strategist/ → <project>/.strategist/`. Frágil —
o symlink se quebra se o projeto for movido ou clonado em outro path.
