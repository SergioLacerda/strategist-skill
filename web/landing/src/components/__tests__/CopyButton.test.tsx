import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, act, fireEvent } from '@testing-library/react';
import CopyButtons from '../CopyButton';

beforeEach(() => {
  vi.useFakeTimers();
  // clipboard is a read-only getter in jsdom — define it manually
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    writable: true,
    configurable: true,
  });
  document.body.innerHTML = `
    <button class="copy-btn" data-command="npm install strategist">copy</button>
  `;
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('CopyButtons', () => {
  it('renders nothing initially', () => {
    const { container } = render(<CopyButtons />);
    expect(container.firstChild).toBeNull();
  });

  it('shows toast after clicking a .copy-btn', async () => {
    render(<CopyButtons />);
    const copyBtn = document.querySelector('.copy-btn') as HTMLElement;
    await act(async () => {
      fireEvent.click(copyBtn);
      await Promise.resolve(); // flush clipboard promise
    });
    expect(screen.getByText('✓ copied')).toBeTruthy();
  });

  it('calls clipboard.writeText with data-command value', async () => {
    render(<CopyButtons />);
    const copyBtn = document.querySelector('.copy-btn') as HTMLElement;
    await act(async () => {
      fireEvent.click(copyBtn);
      await Promise.resolve();
    });
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('npm install strategist');
  });

  it('hides toast after 1800ms', async () => {
    render(<CopyButtons />);
    const copyBtn = document.querySelector('.copy-btn') as HTMLElement;
    await act(async () => {
      fireEvent.click(copyBtn);
      await Promise.resolve();
    });
    expect(screen.getByText('✓ copied')).toBeTruthy();
    await act(async () => { vi.advanceTimersByTime(1800); });
    expect(screen.queryByText('✓ copied')).toBeNull();
  });

  it('does not show toast when clicking non-.copy-btn elements', async () => {
    render(<CopyButtons />);
    await act(async () => { fireEvent.click(document.body); });
    expect(screen.queryByText('✓ copied')).toBeNull();
  });

  it('toast has aria-live="polite"', async () => {
    render(<CopyButtons />);
    const copyBtn = document.querySelector('.copy-btn') as HTMLElement;
    await act(async () => {
      fireEvent.click(copyBtn);
      await Promise.resolve();
    });
    const toast = screen.getByText('✓ copied');
    expect(toast.getAttribute('aria-live')).toBe('polite');
  });
});
