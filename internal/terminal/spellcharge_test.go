package terminal

import (
	"testing"
)

func TestNewSpellCharge_initialView(t *testing.T) {
	m := newSpellCharge("install", 4)
	view := m.View()
	if view == "" {
		t.Fatal("View() should not be empty before quitting")
	}
}

func TestSpellCharge_progressUpdates(t *testing.T) {
	m := newSpellCharge("install", 4)

	updated, _ := m.Update(ProgressMsg{Done: 2, Total: 4})
	m2 := updated.(spellChargeModel)
	if m2.done != 2 || m2.total != 4 {
		t.Errorf("expected done=2 total=4, got done=%d total=%d", m2.done, m2.total)
	}
}

func TestSpellCharge_doneMsg(t *testing.T) {
	m := newSpellCharge("install", 4)
	updated, cmd := m.Update(DoneMsg{})
	m2 := updated.(spellChargeModel)

	if !m2.quitting {
		t.Error("expected quitting=true after DoneMsg")
	}
	if m2.View() != "" {
		t.Error("View() should be empty string when quitting")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd after DoneMsg")
	}
}

func TestSpellCharge_zeroTotal(t *testing.T) {
	m := newSpellCharge("install", 0)
	view := m.View()
	if view == "" {
		t.Fatal("View() should not panic with total=0")
	}
}
