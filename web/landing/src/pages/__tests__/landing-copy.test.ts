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
});
