# Strategist — Design System

> _“Your experience with your demands will never be the same.”_

The brand language of **Strategist**, an autonomous developer skill that turns a technical request into a governed mission (discovery → refinement → **approval gate** → execution) through pluggable RPG-styled roles. The aesthetic is an **arcane terminal / grimoire** — amber CRT phosphor glowing off deep resin-brown panels, Cinzel display type over IBM Plex Mono, and a Dungeons-&-Dragons "party" metaphor for the pipeline (rendered **"grupo"** in the Portuguese voice).

The primary surface is the **Strategist Console (v2)** — a tabbed, bilingual console that frames the product around four promises: *customize the roles*, *specialize roles with multiple skills*, *integrate with governance models*, and *generate documentation to your standards*.

This system was reverse-engineered from the project's own canonical landing page, which is the single source of truth for every token, glyph, and treatment below.

---

## Sources

- **GitHub:** <https://github.com/SergioLacerda/strategist-skill> — the skill itself. Explore it further to design more faithfully against the product:
  - `pages/index.html` — **the canonical landing page**; every color, font, glow, and glyph in this system is lifted from it.
  - `readme.md` / `readme_en.md` — product overview, the canonical pipeline (`Ranger → Archivist → approval gate → Sniper`), install flow, slot config, credits (PT-BR is the primary language).
  - `readme_detailed.md` / `readme_detailed_en.md` — the deep technical spec: full mission pipeline, internal flow, personas, knowledge system, slot contracts, SDD integration, stop conditions, forbidden behaviors, drift self-correction. **The authority for every product fact in this guide.**
  - `strategist/SKILL.md`, `strategist/protocol.md`, `strategist/skill.yaml`, `strategist/personas/`, `strategist/roles/` — the slot definitions (Ranger, Archivist, Sniper), persona voices (`epic`, `pragmatic`), and the routing contract.
  - `docs/architecture.md`, `docs/c4-diagrams.md`, `docs/fluxo-*.png` — flow diagrams.
- **Hosted docs / landing:** <https://sergiolacerda.github.io/strategist-skill/>
- **Governance integration (optional):** Strategist is **standalone by default** and can plug into a governance harness — the **SDD Harness** — documented in its own `readme_detailed*` *SDD Integration* section: SDD injects the execution (Sniper) slot, `base_path`, extra `knowledge_paths`, and a read-only `governance_context`. This is what the console's *Integration with governance models* promise refers to.
- **Authors:** Sergio Lacerda & Raphael Vernil. **License:** CC BY-NC 4.0 (commercial use requires prior authorization).

> No raster brand assets (logos, illustrations, photography) exist in the repo — the mark is a typographic crest, and all iconography is Unicode glyphs. Nothing was imported as an image.

---

## Product model (ground truth)

Use these facts when writing copy or building screens so the brand stays accurate to the repo:

- **One orchestrator + three slots.** Strategist orchestrates; it validates contracts, emits progress, and enforces the gate — it **never executes**. The pluggable slots are **Ranger** (`discovery`, contract `write_pending`), **Archivist** (`refinement`, contract `write_analysis`), **Sniper** (`execution`, contract `controlled`).
- **Canonical pipeline:** `Bootstrap → Preflight → Intake → Context Enrichment → Ranger → Opportunist Attack (internal) → Archivist → Approval Gate → Sniper → Learning Loop`. The console's *Mission Flow* shows the four human-facing phases (Discovery · Refinement · Approval Gate · Execution).
- **Approval Gate is mandatory** whenever `tasks.md` is non-empty; an empty or absent `tasks.md` resolves as `analysis_delivered` with no gate. Invoking Sniper without explicit approval is a *forbidden behavior*.
- **Two personas, one pipeline:** `epic` (ranger / archivist / sniper labels, strategic tone) and `pragmatic` (analysis / refinement / execution labels, analytical tone). The `mode: pragmatic · epic` badge reflects this.
- **Artifacts:** `pending/<id>-discovery.md` → `refined/<id>/{proposal,design,tasks}.md` → `done/<id>-report.md`. Config lives in the skill root (`active.yaml`, `roles/`, `personas/`, `knowledge.index.yaml`) — never written into the target repo.
- **Install:** bootstrap the binary (`curl … | bash`), then `strategist install --wizard` (interactive TUI to pick template, base_path, slot providers, knowledge source).
- **Default slot bindings:** `discovery: brainstorm` · `refinement: openspec-explore` · `execution: sdd-ask`. External base skills credited: `brainstorming` (Ranger) and `openspec` (Archivist).

---

## Content fundamentals

How Strategist writes:

- **Bilingual, PT-BR first.** The primary product voice is Brazilian Portuguese; English is a parallel translation. Ship copy in both when possible (the UI kit carries a PT/EN toggle).
- **RPG / dungeon metaphor, taken seriously.** A mission is run by an **orchestrator** (the **Strategist** — it coordinates, it never executes) plus **three pluggable slots**: **Ranger** (discovery), **Archivist** (refinement), **Sniper** (execution). The canonical pipeline is `Ranger → Archivist → approval gate → Sniper`, with an internal **Opportunist Attack** scan (side quests) between Ranger and Archivist and a **non-blocking Learning Loop** at the end. **Wizard** is not a runtime slot — it is the install persona (`strategist install --wizard`); the brand's landing dramatizes it as a party member. In the PT voice the party is **"grupo"**. A ranked slot binds a provider skill and can arm its sub-skills as **weapons** (e.g. Ranger→`brainstorming`, Archivist→`openspec-explore`). In-mission capabilities are "Abilities" (distinct from Weapons — external provider skills): _Opportunist Attack_ / _Side Quest_, _Search_, _Critical Hit_ (label only — mechanically a Route resolved by Scout). _Treasure Chest_ (offline knowledge by scope) is the resource Search consults, not an Ability. The approval checkpoint is the **Approval Gate** — a mandatory human stop before any execution. Lean into this vocabulary; it is the brand's personality.
- **Tone:** confident, terse, a little incantatory. Headlines are evocative ("the invocation spell", "approval gate · inviolable rule"); body copy is precise and technical. The famous self-aware nag link — _"Too fancy? I want the normal version!"_ — shows the brand winks at its own theatrics.
- **Casing:** section labels and klass tags are **UPPERCASE** with wide tracking. Role/title names use Title Case in the grimoire face. Code identifiers (`task_type`, `mission_mode`, `active.yaml`, `tasks.md`, `.analysis/`) stay lowercase mono and are often italicized inline. Progress is reported as single-line events: `[Strategist] phase=<label> status=done`.
- **Voice:** speaks _about_ the system in third person ("The Strategist orchestrates…"), addresses the user implicitly. Imperative for CTAs ("Invoke the skill").
- **Punctuation signatures:** the middot `·` separates label pairs ("Master · Orchestrator", "pragmatic · epic"); `::` introduces requirements ("REQUIRES :: human confirmation"); `❯` prefixes commands; `⟡` and `✦` ornament headings.
- **Emoji:** essentially none in product chrome — only national flags 🇧🇷 🇺🇸 on the language toggle. Do **not** add decorative emoji; use the Unicode sigil set instead.

---

## Visual foundations

- **Mood:** a glowing amber CRT terminal bound in a leather grimoire. Dark, warm, arcane, slightly haunted.
- **Color:** deep resin-brown surfaces (`#0c0805`→`#2a2117`) under an **amber phosphor** family (`#e8c25a` / highlight `#f8df90` / dim `#b8933f`) that carries the brand. **Ember** orange (`#cf7a2c`) and rust (`#8a4128`) are the warm secondary (section labels, the gate, portcullis walls). **CRT green** (`#74cf8e`) is used _sparingly_ for "OK / live / passing" and the learning-loop line only. Text is parchment (`#d9c49a` → muted `#9b865d` → faint `#6c5c3e`). See `tokens/colors.css`.
- **Type:** three voices — **Cinzel Decorative** (grimoire titles, the wordmark), **Cinzel** (section headings, role names), **IBM Plex Mono** (all body, labels, terminal output). Letter-spacing is wide and deliberate: `.22em` on klass labels, `.4em` on section labels. See `tokens/typography.css`.
- **Backgrounds:** radial-gradient resin washes (lighter at top, near-black at bottom), never flat. No photography, no illustration. The **CRT overlay** (`.crt`) lays repeating scanlines (multiply) plus a vignette and a soft top phosphor bloom over the whole page.
- **Glow is the signature.** Every amber element leaks light — text via `text-shadow` (`--glow-amber`), panels via inset shadows. Titles get a 30px amber halo; the gate lock gets an orange one.
- **Borders & hairlines:** 1px amber-at-low-alpha lines (`--line` .16, `--line-2` .32). The **Approval Gate** is the only element drawn with a **dashed orange** border — it reads as sealed/forbidden, and is rendered as an explicit **arched portal** (stone jambs + iron portcullis + keystone + glowing central lock); see `components/mission/GatePanel`.
- **Corner radii:** small and sharp — 3px (badges), 4px (buttons, cards, install blocks), 6px (panels, portals). Pills (999px) only on klass tags. This is a terminal, not a soft consumer app.
- **Cards:** dark gradient fill (`--grad-panel`), 1px `--line-2` border, **inset** shadow (`inset 0 0 34px rgba(0,0,0,.4)`) so they read as recessed wells rather than raised cards. Role cards carry a circular sigil medallion on the left.
- **Shadows:** primarily **inset** (sunken panels/terminals). Outer shadows appear only on floating chrome (fixed bars, the amber button glow). See `tokens/effects.css`.
- **Motion:** restrained. `~.15s` ease transitions on hover (background lightens, border brightens to `--amber`, box-shadow blooms); buttons translate `1px` down on `:active`. No bounces, no parallax, no infinite loops.
- **Hover / press:** hover = brighter border + stronger amber glow + slightly lighter fill. Press = 1px downward nudge. Disabled = ~45% opacity.
- **Layout:** a fixed **1120px** page centered in the viewport (it scrolls horizontally below that), 80px hero/section gutters, ornamental `⟡ ━━━ ⟡` rules opening every section.
- **Transparency & blur:** sparse — `backdrop-filter: blur(6px)` only on the fixed floating CTA/lang chrome. Everywhere else, depth comes from gradients and inset shadows.

---

## Iconography

- **System:** **Unicode glyphs only.** No icon font, no SVG sprite, no PNG icons, no emoji (flags excepted). Each glyph sits inside a circular hairline **medallion** with a radial amber wash and a phosphor glow.
- **Sigil set (meaning is fixed):** ☉ Strategist · ⌖ Ranger · ❒ Archivist · ✜ Sniper · ✶ Wizard · ⛨ Approval Gate / portal (orange) · ⚲ Opportunist Attack · ⚑ Side Quest · ❖ Treasure Chest · ⧗ Initiative · ⛩ Dojo · ‡ / † Weapons (arsenal · individual) · ⌕ Discovery · ✎ Refinement · ◎ Execution (scope) · ✦ crest · ⟡ ornament.
- **Drawing rule:** never hand-roll SVG icons or substitute emoji. Reuse glyphs from this set; if a new concept needs a mark, pick a fitting Unicode symbol and give it the medallion treatment (see `components/party/SkillPanel` and the `guidelines/brand-glyphs.html` card).

---

## Index / manifest

**Root**
- `styles.css` — global entry point (consumers link this). `@import` manifest only.
- `tokens/` — `fonts.css`, `colors.css`, `typography.css`, `spacing.css`, `effects.css`.
- `README.md` (this file) · `SKILL.md` (Agent-Skills-compatible wrapper).
- `Strategist Console.html` / `site/index.html` — single self-contained, **offline-ready** export of the console (also downloadable as `site.zip`).

**Components** (`window.StrategistDesignSystem_88a8a0.*`)
- `components/brand/` — **Logo** (diamond compass-seal emblem + grimoire wordmark; `lockup` / `mark` / `wordmark`)
- `components/core/` — **Button**, **Badge**, **SectionLabel**
- `components/party/` — **RoleCard**, **SkillPanel**
- `components/mission/` — **GatePanel**, **InstallBlock**

**Foundation cards** (`guidelines/`) — Colors (amber, ember/green, surfaces, text/lines), Type (grimoire, mono, scale), Spacing (scale, radii), Effects (glow/CRT), Brand (mark, sigils).

**UI kits**
- `ui_kits/console/` — **Strategist Console (v2)**, the current primary surface. A professional tabbed, bilingual (PT/EN, state-persisted) console — `index.html` + `i18n.js`:
  - **Overview** — seal logo, tagline, CI/version/license badges, and four value-prop tiles (*Customize roles* · *Specialize with multiple skills* · *Governance-model integration* · *Documentation to your standards*). Floating GitHub mark, top-right.
  - **Roles** — the orchestrator's three pluggable slots (Ranger / Archivist / Sniper; Wizard shown as the installer) plus the **Rank & Weapons** mechanic (a ranked slot binds a provider and arms its sub-skills as "weapons"), with real examples (Ranger → `brainstorming` → `writing-skills`; Archivist → `openspec-explore` → `openspec-propose` / `-apply-change` / `-archive-change`).
  - **Abilities** — special actions, distinct from Weapons. From the repo: **Opportunist Attack**, **Search**, **Critical Hit**, **Side Quest**. **Treasure Chest** is a resource Search consults, not an Ability. Plus two **product additions** (not yet in the repo): **Initiative** (assess the team before missions) and **Dojo** (controlled environment to test skills).
  - **Mission Flow** — four phases with medallion icons (⌕ Discovery · ✎ Refinement · ⛨ Approval Gate · ◎ Execution), the ASCII pipeline diagram, and the arched approval-gate portal.
  - **Invoke** — two-step setup: bootstrap via `curl` / `irm`, then the CLI `strategist install --wizard`.
- `ui_kits/landing/` — **v1** single-scroll landing recreation, kept for reference (`index.html`, `Hero.jsx`, `PipelineMap.jsx`, `i18n.js`).

---

## Notes & substitutions

- **Fonts** are loaded from **Google Fonts** (Cinzel, Cinzel Decorative, IBM Plex Mono) via `@import` in `tokens/fonts.css` rather than self-hosted `.woff2` files — these are the exact families used by the source, so this is a faithful match, not a guess. For fully offline/self-hosted delivery, drop the binaries in `assets/fonts/` and swap the `@import` for local `@font-face` rules.
