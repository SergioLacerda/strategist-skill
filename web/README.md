# web/

Domínio autocontido para tudo relacionado ao site do Strategist.

## Estrutura

```
web/
├── design/     ← sistema de design (tokens, componentes, guidelines, ui_kits)
└── landing/    ← site de produção (Astro SSG)
```

## design/

Sistema de design completo da marca Strategist — amber CRT phosphor / grimoire.

- `design/README.md` — guia do sistema (tokens, componentes, uso)
- `design/SKILL.md` — wrapper agent-skills: invocável para gerar novos assets da marca
- `design/ui_kits/console/` — Strategist Console v2 (protótipo de referência, não produção)
- `design/ui_kits/landing/` — Landing v1 (referência)
- `design/guidelines/` — specimens de cores, tipo, glifos, efeitos

Abrir o protótipo localmente:

```bash
open web/design/ui_kits/console/index.html
# ou
open "web/design/Strategist Console.html"
```

## landing/

Site de produção em Astro SSG. Mesma identidade visual do protótipo, mas com:

- HTML real no initial load (SEO, no-JS, LCP rápido)
- React production build + islands mínimas (~3 ilhas: tabs, língua, copy)
- Fonts woff2 self-hosted + assets fingerprinted
- Sem React dev build, sem Babel no browser

### Dev

```bash
cd web/landing
npm install
npm run dev       # localhost:4321
```

### Build

```bash
cd web/landing
npm run build     # → web/landing/dist/
npm run preview   # preview do build estático
```

Ou via Makefile na raiz:

```bash
make build-site   # npm ci + build
make build-all    # Go build + site build
```

### Deploy (GitHub Pages)

O build em `dist/` é um site estático puro — pode ser servido por qualquer CDN.
Para GitHub Pages: aponte para `web/landing/dist/` ou configure o workflow para publicar a partir daí.

### Custom domain

Por padrão, `astro.config.mjs` está configurado para GitHub Pages com subpath:

```js
site: 'https://sergiolacerda.github.io',
base: '/strategist-skill',
```

E `src/styles/tokens/fonts.css` referencia as fontes com esse prefixo fixo:

```css
src: url('/strategist-skill/fonts/cinzel-latin.woff2') ...
```

**Se usar domínio próprio** (sem subpath), dois ajustes necessários:

1. Em `astro.config.mjs`: remover `base` ou definir `base: '/'`
2. Em `src/styles/tokens/fonts.css`: trocar `/strategist-skill/fonts/` por `/fonts/`

## Mapeamento design → produção

| Protótipo (design/) | Produção (landing/) |
|---------------------|---------------------|
| `ui_kits/console/index.html` (React+Babel) | `src/pages/index.astro` (SSG) |
| `components/*.jsx` inline-styled | `src/components/*.astro` + islands `.tsx` |
| `tokens/*.css` via `@import` | `src/styles/tokens/` copiados |
| Google Fonts `@import` | `public/fonts/*.woff2` self-hosted |
| `i18n.js` (`window.CONSOLE_I18N`) | inlined no build Astro |
