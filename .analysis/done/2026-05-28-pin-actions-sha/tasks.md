# Tasks: Pin Actions to SHA

## Escopo
Editar `.github/workflows/release.yml` — 2 substituições de string.

## Tarefas

1. Substituir linha 17:
   - De: `uses: actions/checkout@v4`
   - Para: `uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4`

2. Substituir linha 39:
   - De: `uses: softprops/action-gh-release@v2`
   - Para: `uses: softprops/action-gh-release@3bb12739c298aeb8a4eeaf626c5b8d85266b0e65 # v2`

3. Commit: `fix: pin GitHub Actions to commit SHA to prevent supply-chain attacks`
