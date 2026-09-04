import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// Structural regression coverage for the SDD badge added to the epic header
// (mission 20260902-landing-header-sdd-badge-refinement, SQ-001). This is a
// source-level snapshot, not a pixel screenshot: it guards the properties a
// real visual regression would catch cheaply (DOM position, token reuse,
// PT/EN parity, topbar spacing) without adding a browser-automation
// dependency. See design.md's Validation Plan for what a human visual check
// still covers (actual pixel alignment, overlap on narrow viewports).

const root = resolve(__dirname, '../../..');

function readSrc(path: string): string {
  return readFileSync(resolve(root, path), 'utf8');
}

const epicSource = readSrc('src/pages/epic.astro');
const cssSource = readSrc('src/styles/global.css');

describe('epic header SDD badge (structural regression)', () => {
  it('renders the badge as the last child of <header class="topbar">, after the rightcluster', () => {
    const headerMatch = epicSource.match(/<header class="topbar">([\s\S]*?)<\/header>/);
    expect(headerMatch, 'expected a <header class="topbar"> block').toBeTruthy();
    const headerBody = headerMatch![1];

    const langToggleIdx = headerBody.indexOf('<LangToggle client:load />');
    const badgeStart = headerBody.indexOf('class="sdd-float"');
    expect(langToggleIdx, 'expected the LangToggle island inside the rightcluster').toBeGreaterThan(-1);
    expect(badgeStart, 'expected the sdd-float badge markup').toBeGreaterThan(-1);
    expect(badgeStart, 'badge must come after the LangToggle island, not before it')
      .toBeGreaterThan(langToggleIdx);

    // Exactly one closing div (the rightcluster's own) sits between LangToggle
    // and the badge — the badge is a sibling after rightcluster closes, not
    // nested inside it.
    const between = headerBody.slice(langToggleIdx, badgeStart);
    expect(between.match(/<\/div>/g)?.length).toBe(1);

    // The badge is the last element child of <header> — only whitespace and
    // its own closing </div> separate it from </header>.
    const badgeCloseIdx = headerBody.indexOf('</div>', badgeStart);
    const afterBadge = headerBody.slice(badgeCloseIdx + '</div>'.length);
    expect(afterBadge.trim()).toBe('');
  });

  it('keeps the TabSwitcher and LangToggle islands present and untouched by the badge addition', () => {
    const headerMatch = epicSource.match(/<header class="topbar">([\s\S]*?)<\/header>/);
    const headerBody = headerMatch![1];
    expect(headerBody).toContain('<TabSwitcher client:load />');
    expect(headerBody).toContain('<LangToggle client:load />');
  });

  it('badge carries PT/EN parity via data-pt/data-en, matching the LangToggle [data-pt] contract', () => {
    const badgeMatch = epicSource.match(/<div\s+class="sdd-float"[\s\S]*?<\/div>/);
    expect(badgeMatch, 'expected the sdd-float badge block').toBeTruthy();
    const badge = badgeMatch![0];
    expect(badge).toContain('data-pt="Uma Experiência SDD"');
    expect(badge).toContain('data-en="An SDD Experience"');
    // The initial rendered text must match data-pt (site defaults to pt-BR).
    expect(badge).toMatch(/>\s*Uma Experiência SDD\s*</);
  });

  it('.sdd-float uses only pre-existing design tokens, no new hex/rgb color literals for text or border', () => {
    const ruleMatch = cssSource.match(/\.sdd-float\s*{([\s\S]*?)}/);
    expect(ruleMatch, 'expected a .sdd-float rule in global.css').toBeTruthy();
    const rule = ruleMatch![1];

    expect(rule).toContain('color: var(--amber-hi)');
    expect(rule).toContain('border: 1px solid var(--amber-dim)');
    expect(rule).toContain('border-radius: var(--radius-pill)');
    expect(rule).toContain('font-family: var(--font-mono)');

    // No bare hex color literal (e.g. #f8df90) — only var(--token) or rgba()
    // (background/shadow gradients are allowed to use rgba directly, per design.md).
    expect(rule).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
  });

  it('.sdd-float is positioned to hang below the topbar (absolute + top:100% + translateY(-50%))', () => {
    const ruleMatch = cssSource.match(/\.sdd-float\s*{([\s\S]*?)}/);
    const rule = ruleMatch![1];
    expect(rule).toContain('position: absolute');
    expect(rule).toContain('top: 100%');
    expect(rule).toContain('transform: translateY(-50%)');
  });

  it('.topbar bottom padding was widened to make room for the badge', () => {
    const ruleMatch = cssSource.match(/(?<!\.sdd-float\s*)\.topbar\s*{([\s\S]*?)}/);
    expect(ruleMatch, 'expected a .topbar rule in global.css').toBeTruthy();
    const rule = ruleMatch![1];
    const paddingMatch = rule.match(/padding:\s*([\d.]+)px\s+([\d.]+)px\s+([\d.]+)px/);
    expect(paddingMatch, 'expected a 3-value px padding shorthand on .topbar').toBeTruthy();
    const bottomPadding = Number(paddingMatch![3]);
    expect(bottomPadding, '.topbar bottom padding must be widened beyond the pre-badge 16px baseline')
      .toBeGreaterThan(16);
  });
});
