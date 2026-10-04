import { beforeEach, describe, expect, it } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import TabSwitcher from '../TabSwitcher';

beforeEach(() => {
  localStorage.clear();
  window.history.replaceState(null, '', '/');
  document.body.innerHTML = ['overview', 'features', 'structure', 'mission', 'invoke', 'allies'].map((panel) => `<div class="panel" data-panel="${panel}"></div>`).join('');
});

describe('TabSwitcher', () => {
  it('renders the six bilingual tabs', () => {
    render(<TabSwitcher />);
    expect(screen.getByText('Visão Geral')).toBeTruthy();
    expect(screen.getByText('Funcionalidades')).toBeTruthy();
    expect(screen.getByText('Componentes')).toBeTruthy();
    expect(screen.getByText('Missão')).toBeTruthy();
    expect(screen.getByText('Instalação')).toBeTruthy();
    expect(screen.getByText('Aliados')).toBeTruthy();
    expect(screen.getByText('Funcionalidades').dataset.en).toBe('Features');
  });

  it('normalizes legacy localStorage values to overview', () => {
    localStorage.setItem('strategist_console_tab', 'roles');
    render(<TabSwitcher />);
    expect(screen.getByText('Visão Geral').className).toContain('active');
  });

  it('switches panels and persists a supported tab', () => {
    render(<TabSwitcher />);
    fireEvent.click(screen.getByText('Aliados'));
    expect(localStorage.getItem('strategist_console_tab')).toBe('allies');
    expect(document.querySelector('[data-panel="allies"]')?.classList.contains('active')).toBe(true);
  });

  it('opens Allies from the hash', () => {
    window.history.replaceState(null, '', '/epic/#allies');
    render(<TabSwitcher />);
    expect(screen.getByText('Aliados').className).toContain('active');
  });

  it('responds to supported tab events and ignores unknown values', async () => {
    render(<TabSwitcher />);
    await act(async () => window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'features' })));
    expect(screen.getByText('Funcionalidades').className).toContain('active');
    await act(async () => window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'roles' })));
    expect(screen.getByText('Funcionalidades').className).toContain('active');
  });
});
