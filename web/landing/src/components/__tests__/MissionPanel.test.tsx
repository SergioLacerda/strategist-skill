import { beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import MissionPanel from '../MissionPanel';

beforeEach(() => {
  localStorage.clear();
  document.body.innerHTML = '';
});

describe('MissionPanel', () => {
  it('starts with all five phases closed', () => {
    render(<MissionPanel />);
    expect(screen.getAllByRole('button', { name: /fase|phase/i })).toHaveLength(5);
    expect(document.querySelector('.gate-card strong')).toBeNull();
  });

  it('keeps only one phase open at a time', () => {
    render(<MissionPanel />);
    const phases = screen.getAllByRole('button', { name: /fase|phase/i });
    fireEvent.click(phases[0]);
    expect(screen.getByText('Prompt Intake')).toBeTruthy();
    fireEvent.click(phases[1]);
    expect(screen.getByText('Dossier Builder')).toBeTruthy();
    expect(screen.queryByText('Prompt Intake')).toBeNull();
  });

  it('renders Approval Gate details when its phase opens', () => {
    render(<MissionPanel />);
    fireEvent.click(screen.getAllByRole('button', { name: /fase|phase/i })[2]);
    expect(document.querySelector('.gate-panel-visual')).toBeTruthy();
    expect(screen.getByText(/REQUER ::/)).toBeTruthy();
    expect(screen.getByText(/aceitação humana/)).toBeTruthy();
  });
});
