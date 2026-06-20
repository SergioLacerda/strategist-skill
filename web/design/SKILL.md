---
name: strategist-design
description: Use this skill to generate well-branded interfaces and assets for Strategist (the arcane-terminal / grimoire developer skill — amber CRT phosphor on resin-brown, Cinzel + IBM Plex Mono, RPG "party" metaphor), either for production or throwaway prototypes/mocks/etc. Contains essential design guidelines, colors, type, fonts, assets, and UI kit components for prototyping.
user-invocable: true
---

Read the README.md file within this skill, and explore the other available files.

If creating visual artifacts (slides, mocks, throwaway prototypes, etc), copy assets out and create static HTML files for the user to view. If working on production code, you can copy assets and read the rules here to become an expert in designing with this brand.

If the user invokes this skill without any other guidance, ask them what they want to build or design, ask some questions, and act as an expert designer who outputs HTML artifacts _or_ production code, depending on the need.

## Where things live
- `styles.css` — link this one file to pick up every token and webfont.
- `tokens/` — colors, typography, spacing, effects (CRT overlay, phosphor glows, gradients).
- `components/` — React primitives: `core/` (Button, Badge, SectionLabel), `party/` (RoleCard, SkillPanel), `mission/` (GatePanel, InstallBlock). Each has a `.prompt.md` with usage.
- `guidelines/` — foundation specimen cards (colors, type, spacing, brand glyphs).
- `ui_kits/landing/` — full bilingual recreation of the Strategist landing page.

## Non-negotiables of the brand
- Amber phosphor on deep resin-brown; **everything amber glows**. CRT scanline + vignette overlay on full pages (`.crt`).
- Three type voices: Cinzel Decorative (titles), Cinzel (headings), IBM Plex Mono (everything else). Wide tracking on labels.
- Iconography is **Unicode glyphs in circular medallions** — never SVG icons or emoji (flags excepted).
- The **Approval Gate** is the only dashed-orange element. RPG "party" vocabulary (Wizard/Strategist/Ranger/Archivist/Sniper, treasure chests, quick draw) is the brand's voice — use it.
- Small sharp radii (3–6px), inset shadows (recessed wells), restrained ~.15s motion.
- Bilingual PT-BR / EN; PT-BR is primary.
