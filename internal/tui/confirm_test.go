package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rewdy/worktree-cli/internal/git"
)

func sampleWT() git.Worktree {
	return git.Worktree{Path: "/work/feat-x", Branch: "feature-x"}
}

func TestConfirmDefaultsToRemove(t *testing.T) {
	m := NewConfirmModel(sampleWT())
	if m.choice != 1 {
		t.Errorf("default focus should be on Remove (choice=1), got %d", m.choice)
	}
}

func TestConfirmEnterOnDefaultConfirms(t *testing.T) {
	m := tea.Model(NewConfirmModel(sampleWT()))
	m = sendKey(m, "enter")
	cm := m.(ConfirmModel)
	if !cm.Result().Confirmed {
		t.Errorf("enter on default focus should confirm")
	}
}

func TestConfirmTabThenEnterCancels(t *testing.T) {
	m := tea.Model(NewConfirmModel(sampleWT()))
	m = sendKey(m, "tab")   // → focus Cancel
	m = sendKey(m, "enter") // cancel
	cm := m.(ConfirmModel)
	if cm.Result().Confirmed {
		t.Errorf("tab + enter from Remove should cancel")
	}
	if !cm.Result().Cancelled {
		t.Errorf("expected cancelled")
	}
}

func TestConfirmYShortcut(t *testing.T) {
	m := tea.Model(NewConfirmModel(sampleWT()))
	// Use raw rune so msg.String() maps to "y"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	cm := m.(ConfirmModel)
	if !cm.Result().Confirmed {
		t.Errorf("y should confirm immediately")
	}
}

func TestConfirmNShortcut(t *testing.T) {
	m := tea.Model(NewConfirmModel(sampleWT()))
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	cm := m.(ConfirmModel)
	if !cm.Result().Cancelled {
		t.Errorf("n should cancel")
	}
}

func TestConfirmEsc(t *testing.T) {
	m := tea.Model(NewConfirmModel(sampleWT()))
	m = sendKey(m, "esc")
	cm := m.(ConfirmModel)
	if !cm.Result().Cancelled {
		t.Errorf("esc should cancel")
	}
}

func TestConfirmViewContainsPath(t *testing.T) {
	m := NewConfirmModel(sampleWT())
	view := stripANSI(m.View())
	if !contains(view, "/work/feat-x") {
		t.Errorf("confirm view should show the worktree path, got:\n%s", view)
	}
	if !contains(view, "Cancel") || !contains(view, "Remove") {
		t.Errorf("confirm view should show both buttons")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
