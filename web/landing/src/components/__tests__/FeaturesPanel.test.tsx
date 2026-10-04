import { beforeEach, describe, expect, it } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import FeaturesPanel from '../FeaturesPanel';

beforeEach(() => {
  localStorage.clear();
  document.body.innerHTML = '';
});

describe('FeaturesPanel', () => {
  it('renders eight supported features and starts with the first one', () => {
    render(<FeaturesPanel />);
    expect(document.querySelectorAll('.feature-tile')).toHaveLength(8);
    expect(screen.getByRole('heading', { name: /Baú do Tesouro/ })).toBeTruthy();
  });

  it('persists the selected feature and falls back from an unknown id', () => {
    localStorage.setItem('strategist_console_feature', 'unknown');
    render(<FeaturesPanel />);
    fireEvent.click(screen.getByRole('button', { name: /Iniciativa/ }));
    expect(localStorage.getItem('strategist_console_feature')).toBe('initiative');
    expect(screen.getByRole('heading', { name: /Iniciativa/ })).toBeTruthy();
  });

  it('updates dynamic content when the language event is dispatched', async () => {
    render(<FeaturesPanel />);
    await act(async () => window.dispatchEvent(new CustomEvent('strategist:lang', { detail: 'en' })));
    expect(screen.getByRole('heading', { name: /Treasure Chest/ })).toBeTruthy();
  });
});
