import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import TabSwitcher from '../TabSwitcher';

beforeEach(() => {
  localStorage.clear();
  // Simulate 5 panel divs in the document
  document.body.innerHTML = `
    <div class="panel" data-panel="overview"></div>
    <div class="panel" data-panel="roles"></div>
    <div class="panel" data-panel="skills"></div>
    <div class="panel" data-panel="mission"></div>
    <div class="panel" data-panel="invoke"></div>
  `;
});

describe('TabSwitcher', () => {
  it('renders all 5 tabs in Portuguese by default', () => {
    render(<TabSwitcher />);
    expect(screen.getByText('Visão Geral')).toBeTruthy();
    expect(screen.getByText('Papéis')).toBeTruthy();
    expect(screen.getByText('Habilidades')).toBeTruthy();
    expect(screen.getByText('Arquitetura')).toBeTruthy();
    expect(screen.getByText('Instalação')).toBeTruthy();
  });

  it('carries data-pt/data-en attributes so LangToggle can translate the nav', () => {
    render(<TabSwitcher />);
    const roles = screen.getByText('Papéis');
    expect(roles.dataset.pt).toBe('Papéis');
    expect(roles.dataset.en).toBe('Roles');
    const mission = screen.getByText('Arquitetura');
    expect(mission.dataset.pt).toBe('Arquitetura');
    expect(mission.dataset.en).toBe('Architecture');
  });

  it('defaults to overview tab', () => {
    render(<TabSwitcher />);
    const btn = screen.getByText('Visão Geral');
    expect(btn.className).toContain('active');
  });

  it('switches active tab on click', () => {
    render(<TabSwitcher />);
    const rolesBtn = screen.getByText('Papéis');
    fireEvent.click(rolesBtn);
    expect(rolesBtn.className).toContain('active');
    expect(localStorage.getItem('strategist_console_tab')).toBe('roles');
  });

  it('marks matching panel as active on tab switch', () => {
    render(<TabSwitcher />);
    fireEvent.click(screen.getByText('Papéis'));
    const rolesPanel = document.querySelector('[data-panel="roles"]') as HTMLElement;
    expect(rolesPanel.classList.contains('active')).toBe(true);
  });

  it('responds to strategist:tab custom event', async () => {
    render(<TabSwitcher />);
    await act(async () => {
      window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'skills' }));
    });
    const skillsBtn = screen.getByText('Habilidades');
    expect(skillsBtn.className).toContain('active');
  });

  it('ignores unknown tab values from external events', () => {
    render(<TabSwitcher />);
    window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'unknown' }));
    // overview stays active (default)
    expect(screen.getByText('Visão Geral').className).toContain('active');
  });

  it('restores tab from localStorage', () => {
    localStorage.setItem('strategist_console_tab', 'invoke');
    render(<TabSwitcher />);
    expect(screen.getByText('Instalação').className).toContain('active');
  });
});
