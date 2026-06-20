import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import LangToggle from '../LangToggle';

beforeEach(() => {
  localStorage.clear();
  document.documentElement.lang = '';
  document.body.innerHTML = `
    <span data-pt="Papéis" data-en="Roles">Papéis</span>
    <span data-pt="Invocação" data-en="Invoke">Invocação</span>
  `;
});

describe('LangToggle', () => {
  it('renders PT and EN buttons', () => {
    render(<LangToggle />);
    expect(screen.getByText(/PT/)).toBeTruthy();
    expect(screen.getByText(/EN/)).toBeTruthy();
  });

  it('PT button is active by default', () => {
    render(<LangToggle />);
    const ptBtn = screen.getByText(/PT/);
    expect(ptBtn.className).toContain('active');
  });

  it('switches to EN on click', () => {
    render(<LangToggle />);
    fireEvent.click(screen.getByText(/EN/));
    expect(localStorage.getItem('strategist_console_lang')).toBe('en');
    expect(document.documentElement.lang).toBe('en');
  });

  it('swaps data-pt / data-en element content on toggle', () => {
    render(<LangToggle />);
    fireEvent.click(screen.getByText(/EN/));
    const el = document.querySelector('[data-pt="Papéis"]') as HTMLElement;
    expect(el.innerHTML).toBe('Roles');
  });

  it('restores to EN from localStorage', () => {
    localStorage.setItem('strategist_console_lang', 'en');
    render(<LangToggle />);
    expect(screen.getByText(/EN/).className).toContain('active');
  });

  it('switches back to PT', () => {
    render(<LangToggle />);
    fireEvent.click(screen.getByText(/EN/));
    fireEvent.click(screen.getByText(/PT/));
    expect(localStorage.getItem('strategist_console_lang')).toBe('pt');
    const el = document.querySelector('[data-pt="Papéis"]') as HTMLElement;
    expect(el.innerHTML).toBe('Papéis');
  });

  it('dispatches strategist:lang event on change', () => {
    const events: string[] = [];
    window.addEventListener('strategist:lang', (e) => {
      events.push((e as CustomEvent).detail);
    });
    render(<LangToggle />);
    fireEvent.click(screen.getByText(/EN/));
    expect(events).toContain('en');
  });
});
