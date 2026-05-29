# Tasks: Guardrails em 3 Sprints
**Mission ID:** guardrails-20260529
**Date:** 2026-05-29

---

## Sprint 1 — CI + Linters

### T1.1 — Adicionar Format Check ao CI
**Arquivo:** `.github/workflows/test.yml`
**Ação:** inserir step antes do step `Build`:
```yaml
      - name: Format check
        run: test -z "$(gofmt -l .)"
```

### T1.2 — Adicionar Module Hygiene ao CI
**Arquivo:** `.github/workflows/test.yml`
**Ação:** inserir step antes do step `Build`:
```yaml
      - name: Module hygiene
        run: |
          go mod tidy
          git diff --exit-code go.mod go.sum
          go mod verify
```

### T1.3 — Adicionar govulncheck ao CI
**Arquivo:** `.github/workflows/test.yml`
**Ação:** inserir step antes do step `Build`:
```yaml
      - name: Vulnerability check
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
```

### T1.4 — Adicionar target `vuln` ao Makefile
**Arquivo:** `Makefile`
**Ação 1:** adicionar à linha `.PHONY`:
```
.PHONY: build test lint vuln cover cover-gate cover-html install-local release clean
```
**Ação 2:** adicionar target após `lint:`:
```makefile
vuln:
	govulncheck ./...
```

### T1.5 — Adicionar linters ao `.golangci.yaml`
**Arquivo:** `.golangci.yaml`
**Ação 1:** adicionar à seção `linters.enable`:
```yaml
    - misspell
    - dupl
    - unconvert
    - ineffassign
    - gocritic
    - contextcheck
```
**Ação 2:** adicionar à seção `linters.settings`:
```yaml
    dupl:
      threshold: 100
```
**Ação 3:** adicionar à seção `exclusions.rules`:
```yaml
      - path: internal/install/installer\.go
        linters:
          - contextcheck
        text: "passes a context to"
```

---

## Sprint 2 — Enforcement Arquitetural

### T2.1 — Adicionar depguard ao `.golangci.yaml`
**Arquivo:** `.golangci.yaml`
**Ação 1:** adicionar `- depguard` à seção `linters.enable`
**Ação 2:** adicionar à seção `linters.settings` o bloco completo `depguard:` com as 4 regras
(domain-isolation, compile-lateral, install-lateral, stale-lateral) conforme design.md §2.1

### T2.2 — Expandir TestLateralIsolation em `internal/domain/architecture_test.go`
**Arquivo:** `internal/domain/architecture_test.go`
**Ação:** adicionar função `TestLateralIsolation` conforme design.md §2.2
(manter `TestDomainIsolation` existente, apenas adicionar a nova função no mesmo arquivo)

### T2.3 — Criar mandate de arquitetura
**Arquivo:** `strategist/contracts/architecture-rules.yaml` (criar)
**Ação:** criar arquivo com conteúdo conforme design.md §2.3

---

## Sprint 3 — Governance + Skill Root Fix

### T3.1 — Criar mandate NO HACK WITHOUT EVIDENCE
**Arquivo:** `strategist/contracts/no-hack-without-evidence.md` (criar)
**Ação:** criar arquivo com conteúdo conforme design.md §3.1

### T3.2 — Criar mandate TEST INTEGRITY
**Arquivo:** `strategist/contracts/test-integrity.md` (criar)
**Ação:** criar arquivo com conteúdo conforme design.md §3.2

### T3.3 — Criar mandate SCOPE LOCKING
**Arquivo:** `strategist/contracts/scope-locking.md` (criar)
**Ação:** criar arquivo com conteúdo conforme design.md §3.3

### T3.4 — Implementar `strategist install-global`
**Arquivos afetados:**
- `cmd/strategist/` — novo subcomando Cobra `install-global`
- `internal/install/installer.go` — reutilizar `Install()` com `target = os.UserHomeDir()`
- `~/.claude/skills/strategist/SKILL.md` — atualizado pelo comando com `source: ~/.strategist`

**Comportamento esperado:**
```
$ strategist install-global
→ extrai defaults em ~/.strategist/
→ atualiza ~/.claude/skills/strategist/SKILL.md
→ exit 0 com mensagem de confirmação
```

**Critério de aceite:**
- `~/.strategist/SKILL.md` existe após execução
- `~/.claude/skills/strategist/SKILL.md` tem `source: ~/.strategist`
- Invocar strategist skill globalmente carrega protocol.md e active.yaml corretamente
