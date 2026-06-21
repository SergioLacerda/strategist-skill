package terminal

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const barWidth = 28

var (
	stylAmber = lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	stylEmber = lipgloss.NewStyle().Foreground(lipgloss.Color("173"))
	stylTrack = lipgloss.NewStyle().Foreground(lipgloss.Color("94"))
	stylGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	stylDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("94"))
)

// ProgressMsg advances the bar: Done steps out of Total.
type ProgressMsg struct{ Done, Total int }

// DoneMsg signals completion (Err may be nil).
type DoneMsg struct{ Err error }

type tickMsg time.Time

type spellChargeModel struct {
	label    string
	done     int
	total    int
	quitting bool
}

func newSpellCharge(label string, total int) spellChargeModel {
	return spellChargeModel{label: label, total: total}
}

func (m spellChargeModel) Init() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m spellChargeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tickMsg:
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
	case ProgressMsg:
		m.done, m.total = v.Done, v.Total
		return m, nil
	case DoneMsg:
		m.done = m.total
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m spellChargeModel) View() string {
	if m.quitting {
		return ""
	}
	p := 0
	if m.total > 0 {
		p = m.done * barWidth / m.total
	}

	bar := stylAmber.Render(strings.Repeat("█", p))
	if p < barWidth {
		bar += stylEmber.Render("▓") + stylTrack.Render(strings.Repeat("░", barWidth-p-1))
	}
	pct := p * 100 / barWidth

	tail := ""
	if m.done >= m.total && m.total > 0 {
		tail = stylGreen.Render(" ✓")
	}
	return stylDim.Render("$ strategist "+m.label) + "\n" +
		fmt.Sprintf("  %s  %s %3d%%%s\n",
			stylAmber.Render("✶ channeling mana"), bar, pct, tail)
}
