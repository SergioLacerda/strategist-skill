import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// Structural regression coverage for the SDD badge added to the pragmatic
// header (re-skinned to match the epic topbar while keeping pragmatic's own
// nav links and PT/EN toggle). Same source-level snapshot approach as
// epic-header-badge.test.ts — see that file for the rationale.

const source = readFileSync(
  resolve(__dirname, '../pragmatic.astro'),
  'utf8',
);

describe('pragmatic header SDD badge (structural regression)', () => {
  it('renders the badge inside .wrap (not .topbar directly), so it aligns with the logo instead of the full-bleed topbar edge', () => {
    const topbarMatch = source.match(/<div class="topbar">([\s\S]*?)\n<\/div>/);
    expect(topbarMatch, 'expected a <div class="topbar"> block').toBeTruthy();
    const topbarBody = topbarMatch![1];

    const langswIdx = topbarBody.indexOf('<div class="langsw">');
    const badgeStart = topbarBody.indexOf('class="sdd-float"');
    expect(langswIdx, 'expected the langsw toggle inside .wrap').toBeGreaterThan(-1);
    expect(badgeStart, 'expected the sdd-float badge markup').toBeGreaterThan(-1);
    expect(badgeStart, 'badge must come after the langsw toggle')
      .toBeGreaterThan(langswIdx);

    // Exactly 2 closing divs (langsw, .right) sit between the langsw toggle
    // and the badge — the badge is a sibling of .right *inside* .wrap, not a
    // sibling of .wrap itself. .wrap must still be open (position:relative)
    // when the badge's left/top offsets are resolved, or the badge aligns
    // to the full-width .topbar edge instead of the centered logo column.
    const between = topbarBody.slice(langswIdx, badgeStart);
    expect(between.match(/<\/div>/g)?.length).toBe(2);

    expect(source).toMatch(/\.topbar \.wrap\{position:relative/);
    expect(source).toContain('data-i18n="sddBadge"');
    expect(source).toMatch(/"sddBadge":"Uma Experiência SDD"/);
    expect(source).toMatch(/"sddBadge":"An SDD Experience"/);
  });

  it('keeps the existing nav links and PT/EN toggle untouched by the badge/logo re-skin', () => {
    expect(source).toContain('data-i18n="ctaGh"');
    expect(source).toContain('<button data-lang="pt">🇧🇷 PT</button><button data-lang="en">🇺🇸 EN</button>');
  });

  it('brand link uses the shared Logo component instead of a plain-text i18n binding', () => {
    expect(source).toContain('import Logo from \'../components/Logo.astro\';');
    expect(source).toContain('<Logo size={30} />');
    // The old data-i18n="brand" binding must be gone: applyLang() sets
    // el.textContent, which would silently wipe out the Logo's rendered
    // markup (SVG + wordmark) on every language switch if left in place.
    expect(source).not.toMatch(/<a class="brand"[^>]*data-i18n="brand"/);
  });
});
