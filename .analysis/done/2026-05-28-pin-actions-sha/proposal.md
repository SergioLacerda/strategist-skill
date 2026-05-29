# Proposal: Pin GitHub Actions to commit SHA

## O quê
Fixar as duas referências mutáveis em `.github/workflows/release.yml` para SHAs
de 40 caracteres, eliminando vetores de supply-chain attack.

## Por quê
Tags (`v4`, `v2`) podem ser re-apontadas pelo dono da action para commits maliciosos
sem aviso — como ocorreu nos incidentes trivy-action e kics-github-action.
CWE-1357 / CWE-353.

## Impacto
- Linha 17: `actions/checkout@v4` → SHA `34e114876b0b11c390a56381ad16ebd13914f8d5`
- Linha 39: `softprops/action-gh-release@v2` → SHA `3bb12739c298aeb8a4eeaf626c5b8d85266b0e65`
- Zero impacto funcional — mesmas versões, referências imutáveis
