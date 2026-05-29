# Design: Pin Actions to SHA

## Mudanças

| Linha | Antes | Depois |
|-------|-------|--------|
| 17 | `actions/checkout@v4` | `actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5` |
| 39 | `softprops/action-gh-release@v2` | `softprops/action-gh-release@3bb12739c298aeb8a4eeaf626c5b8d85266b0e65` |

## SHA verification

- `actions/checkout`: SHA `34e114876b0b11c390a56381ad16ebd13914f8d5` resolvido de `refs/tags/v4` via GitHub API
- `softprops/action-gh-release`: SHA `3bb12739c298aeb8a4eeaf626c5b8d85266b0e65` resolvido de `refs/tags/v2` via GitHub API

## Manutenção futura

Adicionar comentário inline `# vX` após cada SHA para facilitar updates:
```yaml
uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4
uses: softprops/action-gh-release@3bb12739c298aeb8a4eeaf626c5b8d85266b0e65 # v2
```
