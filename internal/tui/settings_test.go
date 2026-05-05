package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rewdy/worktree-cli/internal/settings"
)

func sendSettingsKey(m tea.Model, key string) tea.Model {
	var msg tea.Msg
	switch key {
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyMsg{Type: tea.KeyShiftTab}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "space":
		msg = tea.KeyMsg{Type: tea.KeySpace}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, _ := m.Update(msg)
	return next
}

func TestSettingsSaveRoundtrip(t *testing.T) {
	start := settings.Settings{DefaultPathTemplate: "../", CollapsePaths: false}
	m := tea.Model(NewSettingsModel(start))
	// Tab to the collapse toggle, flip it on.
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "space")
	// Tab past Cancel onto Save, then Enter.
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "enter")
	sm := m.(SettingsModel)
	res := sm.Result()
	if !res.Saved {
		t.Fatalf("expected Saved=true, got %+v", res)
	}
	if !res.Settings.CollapsePaths {
		t.Errorf("expected CollapsePaths=true")
	}
	if res.Settings.DefaultPathTemplate != "../" {
		t.Errorf("template should be preserved, got %q", res.Settings.DefaultPathTemplate)
	}
}

func TestSettingsCancelViaEsc(t *testing.T) {
	m := tea.Model(NewSettingsModel(settings.Defaults()))
	m = sendSettingsKey(m, "esc")
	sm := m.(SettingsModel)
	if sm.Result().Saved {
		t.Errorf("esc should not save")
	}
}

func TestSettingsCancelViaButton(t *testing.T) {
	m := tea.Model(NewSettingsModel(settings.Defaults()))
	// Tab past collapse to Cancel button.
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "enter")
	sm := m.(SettingsModel)
	if sm.Result().Saved {
		t.Errorf("Cancel button should not save")
	}
}

func TestSettingsToggleWithArrowKeys(t *testing.T) {
	m := tea.Model(NewSettingsModel(settings.Settings{DefaultPathTemplate: "../", CollapsePaths: false}))
	m = sendSettingsKey(m, "tab")   // focus collapse
	m = sendSettingsKey(m, "right") // flip on
	// Jump to Save via two tabs.
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "tab")
	m = sendSettingsKey(m, "enter")
	sm := m.(SettingsModel)
	if !sm.Result().Settings.CollapsePaths {
		t.Errorf("right arrow should have toggled collapse on")
	}
}
