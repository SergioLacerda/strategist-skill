# Bilingual Web Pages (pt-BR / EN) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add English versions of `pages/index.html` and `pages/banner.html`, wire language switchers into all four page files, and fix two small issues in `readme_en.md` (wrong banner image + wrong docs badge URL).

**Architecture:** Static HTML only — no build step, no JS i18n library. EN pages are full copies of the pt-BR originals with translated text and updated `lang` attribute. Language switchers are plain HTML links inserted at the `runefoot` div (index pages) and as an absolutely-positioned overlay (banner pages). GitHub Pages serves the `pages/` directory as site root.

**Tech Stack:** HTML, CSS variables from `dungeon.css`, GitHub Pages static hosting.

---

## Task 1: Fix `readme_en.md` (banner image + docs badge URL)

Two one-line edits. Fast, isolated, commit separately.

**Files:**
- Modify: `readme_en.md` lines 2 and 6

**Step 1: Fix banner image (line 2)**

Change:
```html
  <img src="pages/docs/banner.png" alt="Strategist — Your experience with your demands will never be the same." width="100%" />
```
To:
```html
  <img src="pages/docs/banner_en.png" alt="Strategist — Your experience with your demands will never be the same." width="100%" />
```

**Step 2: Fix docs badge URL (line 6)**

Change:
```html
  <a href="https://sergiolacerda.github.io/strategist-skill/">
```
To:
```html
  <a href="https://sergiolacerda.github.io/strategist-skill/index_en.html">
```

**Step 3: Verify**

```bash
grep -n "banner\|sergiolacerda" readme_en.md | head -5
```

Expected:
```
2:  <img src="pages/docs/banner_en.png" ...
6:  <a href="https://sergiolacerda.github.io/strategist-skill/index_en.html">
```

**Step 4: Commit**

```bash
git add readme_en.md
git commit -m "fix: readme_en.md use banner_en.png and point docs badge to EN site"
```

---

## Task 2: Add language switcher to `pages/index.html`

**Files:**
- Modify: `pages/index.html` line 572

**Step 1: Locate the runefoot div**

```bash
grep -n "runefoot" pages/index.html
```

Expected: one match at the closing `</div>` near end of file (currently line 572).

**Step 2: Edit the runefoot**

Find:
```html
    <div class="runefoot"><a href="https://github.com/SergioLacerda/strategist-skill" target="_blank" rel="noopener" style="color:var(--amber-dim);text-decoration:none">strategist-skill</a></div>
```

Replace with:
```html
    <div class="runefoot"><a href="https://github.com/SergioLacerda/strategist-skill" target="_blank" rel="noopener" style="color:var(--amber-dim);text-decoration:none">strategist-skill</a>&nbsp;·&nbsp; 🇧🇷 Português | <a href="index_en.html" style="color:var(--amber-dim);text-decoration:none">🇺🇸 English</a></div>
```

**Step 3: Verify**

```bash
grep "runefoot" pages/index.html
```

Expected: contains `index_en.html` and both flag emojis.

**Step 4: Commit**

```bash
git add pages/index.html
git commit -m "feat: add language switcher to pages/index.html"
```

---

## Task 3: Add language switcher to `pages/banner.html`

`banner.html` has no runefoot. The switcher goes as an absolutely-positioned overlay inside `.banner`.

**Files:**
- Modify: `pages/banner.html`

**Step 1: Locate the closing `.banner` div**

```bash
grep -n "banner\|</div>" pages/banner.html | tail -10
```

Find the last `</div>` that closes the `.banner` div.

**Step 2: Insert the switcher before the closing `</div>`**

Add this line immediately before the closing `</div>` of `.banner`:

```html
  <div style="position:absolute;bottom:12px;right:18px;z-index:10;font-family:var(--mono);font-size:11px;color:var(--faint)">🇧🇷 Português | <a href="banner_en.html" style="color:var(--amber-dim);text-decoration:none">🇺🇸 English</a></div>
```

**Step 3: Verify**

```bash
grep "banner_en" pages/banner.html
```

Expected: one match with the switcher div.

**Step 4: Commit**

```bash
git add pages/banner.html
git commit -m "feat: add language switcher to pages/banner.html"
```

---

## Task 4: Create `pages/index_en.html`

Full EN translation of `pages/index.html` (577 lines). This is the largest task.

**Files:**
- Create: `pages/index_en.html`

**Step 1: Copy source and apply all changes**

Start from `pages/index.html` and apply:

1. `<html lang="pt-BR">` → `<html lang="en">`
2. Translate `<meta name="description" content="...">` to English
3. Translate `<meta property="og:description" content="...">` to English
4. Rename section ID: `id="invocacao"` → `id="invocation"`, update all `href="#invocacao"` → `href="#invocation"`
5. Translate all visible text (headings, paragraphs, button labels, table cells, tooltip text, footer)
6. Keep unchanged: `Ranger`, `Archivist`, `Sniper`, `slots`, `approval gate`, `knowledge index`, `side quest`, `housekeeping scan`, `skill`, `provider`, all CSS class names, all `href` to external URLs, all code/YAML/bash blocks
7. Language switcher in runefoot — reversed:
   ```html
   <a href="index.html" style="color:var(--amber-dim);text-decoration:none">🇧🇷 Português</a> | 🇺🇸 English
   ```

**Step 2: Verify file exists and has correct lang attribute**

```bash
head -3 pages/index_en.html
grep -c "invocation" pages/index_en.html
grep "runefoot" pages/index_en.html
```

Expected:
- Line 2: `<html lang="en">`
- At least 2 matches for `invocation` (the `id` and the `href`)
- runefoot contains `index.html` link and `🇺🇸 English` as plain text

**Step 3: Commit**

```bash
git add pages/index_en.html
git commit -m "feat: create pages/index_en.html — EN landing page"
```

---

## Task 5: Create `pages/banner_en.html`

Full EN translation of `pages/banner.html` (57 lines).

**Files:**
- Create: `pages/banner_en.html`

**Step 1: Copy source and apply all changes**

Start from `pages/banner.html` and apply:

1. `<html lang="pt-BR">` → `<html lang="en">`
2. Translate all visible banner text to English
3. CSS: identical (same `dungeon.css` reference, same inline styles)
4. Language switcher — reversed (links back to `banner.html`):
   ```html
   <div style="position:absolute;bottom:12px;right:18px;z-index:10;font-family:var(--mono);font-size:11px;color:var(--faint)"><a href="banner.html" style="color:var(--amber-dim);text-decoration:none">🇧🇷 Português</a> | 🇺🇸 English</div>
   ```

**Step 2: Verify**

```bash
head -3 pages/banner_en.html
grep "banner.html" pages/banner_en.html
```

Expected:
- Line 2: `<html lang="en">`
- One match for `banner.html` in the switcher link

**Step 3: Commit**

```bash
git add pages/banner_en.html
git commit -m "feat: create pages/banner_en.html — EN banner"
```

---

## Task 6: Final verification

**Step 1: Check all 5 changed files are in git**

```bash
git log --oneline -6
```

Expected: 5 commits from this feature (tasks 1–5).

**Step 2: Verify cross-links are consistent**

```bash
grep "index_en\|banner_en\|index\.html\|banner\.html" pages/index.html pages/index_en.html pages/banner.html pages/banner_en.html
```

Expected:
- `index.html` references `index_en.html` in switcher
- `index_en.html` references `index.html` in switcher
- `banner.html` references `banner_en.html` in switcher
- `banner_en.html` references `banner.html` in switcher

**Step 3: Verify readme_en.md fixes**

```bash
grep -n "banner_en\|index_en" readme_en.md
```

Expected: 2 matches (banner image on line ~2, docs badge href on line ~6).
