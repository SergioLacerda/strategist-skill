import { describe, it, expect, beforeEach, vi } from 'vitest';
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
  it('renders all 5 tabs in PT', () => {
    render(<TabSwitcher lang="pt" />);
    expect(screen.getByText('Visão Geral')).toBeTruthy();
    expect(screen.getByText('Papéis')).toBeTruthy();
    expect(screen.getByText('Habilidades')).toBeTruthy();
    expect(screen.getByText('Fluxo da Missão')).toBeTruthy();
    expect(screen.getByText('Invocação')).toBeTruthy();
  });

  it('renders all 5 tabs in EN', () => {
    render(<TabSwitcher lang="en" />);
    expect(screen.getByText('Overview')).toBeTruthy();
    expect(screen.getByText('Roles')).toBeTruthy();
    expect(screen.getByText('Skills')).toBeTruthy();
    expect(screen.getByText('Mission Flow')).toBeTruthy();
    expect(screen.getByText('Invoke')).toBeTruthy();
  });

  it('defaults to overview tab', () => {
    render(<TabSwitcher lang="pt" />);
    const btn = screen.getByText('Visão Geral');
    expect(btn.className).toContain('active');
  });

  it('switches active tab on click', () => {
    render(<TabSwitcher lang="pt" />);
    const rolesBtn = screen.getByText('Papéis');
    fireEvent.click(rolesBtn);
    expect(rolesBtn.className).toContain('active');
    expect(localStorage.getItem('strategist_console_tab')).toBe('roles');
  });

  it('marks matching panel as active on tab switch', () => {
    render(<TabSwitcher lang="pt" />);
    fireEvent.click(screen.getByText('Papéis'));
    const rolesPanel = document.querySelector('[data-panel="roles"]') as HTMLElement;
    expect(rolesPanel.classList.contains('active')).toBe(true);
  });

  it('responds to strategist:tab custom event', async () => {
    render(<TabSwitcher lang="pt" />);
    await act(async () => {
      window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'skills' }));
    });
    const skillsBtn = screen.getByText('Habilidades');
    expect(skillsBtn.className).toContain('active');
  });

  it('ignores unknown tab values from external events', () => {
    render(<TabSwitcher lang="pt" />);
    window.dispatchEvent(new CustomEvent('strategist:tab', { detail: 'unknown' }));
    // overview stays active (default)
    expect(screen.getByText('Visão Geral').className).toContain('active');
  });

  it('restores tab from localStorage', () => {
    localStorage.setItem('strategist_console_tab', 'invoke');
    render(<TabSwitcher lang="pt" />);
    expect(screen.getByText('Invocação').className).toContain('active');
  });
});
