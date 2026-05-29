# Discovery: Adicionar seção de testes ao readme.md
**Mission ID:** readme-tests-20260529  
**Date:** 2026-05-29

## Estado atual

`readme.md` não tem seção de testes. O projeto tem dois runners distintos:

**Runner 1 — fixture-based** (`strategist/tests/run-tests.sh`)
- Requer: `python3` + `pyyaml`
- Testa: invariantes de segurança via 5 fixtures YAML (approval-bypass, slot-risk-mismatch, discovery-failed, yaml-null-field, side-quest-bypass)

**Runner 2 — unit/integration** (`strategist/tests/harness/run-tests.sh`)
- Requer: `jq`, `yq` (mikefarah v4+), `gzip`
- Testa: shell scripts (check-stale, compile-config, compile-domain, compile-all) + install.sh
- Makefile em `strategist/tests/harness/Makefile`

**BDD specs** (`strategist/tests/specs/*.feature`)
- Documentação formal dos invariantes — sem runner necessário

## Onde inserir

Entre a seção "Instalação local (sem curl / clone repo)" (linha ~136) e "## 📄 Licença" (linha ~139).

## Conteúdo a documentar

- Pré-requisitos por runner (python3+pyyaml vs jq+yq+gzip)
- Como instalar pré-requisitos (macOS/Debian/Arch)
- Como rodar cada runner
- Como rodar componentes individualmente
