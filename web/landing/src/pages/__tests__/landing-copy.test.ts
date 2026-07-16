import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const root = resolve(__dirname, '../../..');

function readPage(path: string): string {
  return readFileSync(resolve(root, path), 'utf8');
}

const pages = [
  'src/pages/index.astro',
  'src/pages/pragmatic.astro',
  'src/pages/epic.astro',
] as const;

describe('landing documentation-only copy', () => {
  it.each(pages)('%s does not promise code implementation by Strategist', (page) => {
    const source = readPage(page).toLowerCase();

    const forbidden = [
      'implements the approved spec',
      'implementa a spec',
      'implementa o escopo',
      'writes code',
      'escrita de código',
      'mudança de código',
      'code execution',
      'developer executing',
      'desenvolvedor executando',
      'execução controlada através de papéis',
      'controlled execution through pluggable roles',
    ];

    for (const token of forbidden) {
      expect(source, `${page} must not contain ${token}`).not.toContain(token);
    }
  });

  it.each(pages)('%s keeps Approval Gate terminology', (page) => {
    const source = readPage(page);
    expect(source).toContain('Approval Gate');
    expect(source).not.toContain('Review Gate');
  });

  it('epic page defines Sniper as documentation executor', () => {
    const source = readPage('src/pages/epic.astro').toLowerCase();
    expect(source).toContain('sniper');
    expect(source).toContain('documentation');
    expect(source).toContain('executor handoff');
  });

  it.each(pages)('%s does not present scan/polish/pack as required public commands', (page) => {
    const source = readPage(page).toLowerCase();
    // scan/polish/pack must never appear as user-facing CLI commands on landing copy
    expect(source).not.toContain('treasure-chest scan');
    expect(source).not.toContain('treasure-chest polish');
    expect(source).not.toContain('treasure-chest pack');
  });

  it.each(pages)('%s does not use "active" as the current jewel status', (page) => {
    const source = readPage(page).toLowerCase();
    expect(source).not.toContain('status: active');
    expect(source).not.toContain('jewel status: active');
  });

  it.each(pages)('%s does not overclaim jewel runtime, telemetry, or Scout runtime routing as already shipped', (page) => {
    const source = readPage(page).toLowerCase();
    const overclaims = [
      'jewel runtime lookup is live',
      'telemetry is live',
      'scout runtime routing is live',
      'already shipped',
    ];
    for (const token of overclaims) {
      expect(source, `${page} must not contain overclaim: ${token}`).not.toContain(token);
    }
  });

  it('pragmatic page mentions Scout, index, and mine without a many-command mining flow', () => {
    const source = readPage('src/pages/pragmatic.astro');
    expect(source.toLowerCase()).toContain('scout');
    expect(source).toContain('<code>index</code>');
    expect(source).toContain('<code>mine</code>');
  });

  it('epic page presents Scout as internal, not a public Mission Flow phase', () => {
    const source = readPage('src/pages/epic.astro');
    expect(source).toContain('Scout');
    // Scout's role card carries the "interno"/"internal" badge
    expect(source).toContain('badgePt="interno"');
    // The Mission Flow phase list (fase 0X) must not include Scout as a numbered phase
    const missionFlowSection = source.split('data-panel="mission"')[1]?.split('</section>')[0] ?? '';
    expect(missionFlowSection).not.toContain('>Scout<');
  });

  it('epic page includes a Joias/Jewels disclosure with source/evidence/status chips', () => {
    const source = readPage('src/pages/epic.astro');
    expect(source).toContain('jewel-disclosure');
    for (const chip of ['documentos', 'source cards', 'evidence packs', 'proposta', 'aceita', 'verificada', 'joias/jewels']) {
      expect(source.toLowerCase()).toContain(chip);
    }
  });

  it('PT/EN parity: Scout badge, Index/Mine wording, and Jewels disclosure have both data-pt and data-en', () => {
    const source = readPage('src/pages/epic.astro');
    // crude parity check: every data-pt on a line introduced by this mission has a sibling data-en
    const scoutBadgeLine = source.split('\n').find((l) => l.includes('badgePt="interno"'));
    expect(scoutBadgeLine).toBeTruthy();
    expect(scoutBadgeLine).toContain('badgeEn="internal"');

    const disclosureLines = source.split('\n').filter((l) => l.includes('data-pt=') && l.includes('jewel') === false && (l.includes('Joias') || l.includes('index') || l.includes('proposta') || l.includes('documentos')));
    for (const line of disclosureLines) {
      expect(line, `line missing data-en: ${line}`).toContain('data-en=');
    }
  });
});
