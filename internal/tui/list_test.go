package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rewdy/worktree-cli/internal/git"
)

func sampleWorktrees() []git.Worktree {
	return []git.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: "/work/feat-x", Branch: "feature-x"},
		{Path: "/work/hotfix-123", Branch: "release/2026-04"},
		{Path: "/work/experiment", Detached: true, HEAD: "a1b2c3d4e5f6"},
	}
}

func sendKey(m tea.Model, key string) tea.Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		msg = tea.KeyMsg{Type: tea.KeyTab}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	}
	next, _ := m.Update(msg)
	return next
}

func TestListSelectsWorktree(t *testing.T) {
	m := tea.Model(NewListModel(sampleWorktrees(), "/repo", ModeSelect))
	// Cursor starts at 0 (which should be /repo since current is sorted first).
	// Move down once → /work/experiment (alphabetical after /repo).
	m = sendKey(m, "down")
	m = sendKey(m, "enter")
	lm := m.(ListModel)
	if lm.Result().Selected == "" {
		t.Fatalf("expected a selection, got empty")
	}
	if lm.Result().AddNew {
		t.Errorf("did not expect AddNew")
	}
}

func TestListCurrentIsSortedFirst(t *testing.T) {
	m := NewListModel(sampleWorktrees(), "/work/feat-x", ModeSelect)
	if len(m.items) == 0 || m.items[0].wt.Path != "/work/feat-x" {
		t.Errorf("current worktree should be first, got %+v", m.items[0])
	}
	if !m.items[0].current {
		t.Errorf("first item should be marked current")
	}
}

func TestListRemoveExcludesCurrent(t *testing.T) {
	m := NewListModel(sampleWorktrees(), "/work/feat-x", ModeRemove)
	for _, it := range m.items {
		if it.wt.Path == "/work/feat-x" {
			t.Errorf("ModeRemove should exclude the current worktree")
		}
		if it.addNew {
			t.Errorf("ModeRemove should not include an Add-new row")
		}
	}
}

func TestListAddNewRowPresentInSelect(t *testing.T) {
	m := NewListModel(sampleWorktrees(), "/repo", ModeSelect)
	last := m.items[len(m.items)-1]
	if !last.addNew {
		t.Errorf("last item should be Add-new in ModeSelect")
	}
}

func TestListFuzzyFilter(t *testing.T) {
	m := tea.Model(NewListModel(sampleWorktrees(), "/repo", ModeSelect))
	// Type "hot" — should filter to hotfix-123 only.
	m = sendKey(m, "h")
	m = sendKey(m, "o")
	m = sendKey(m, "t")
	lm := m.(ListModel)
	visible := lm.visibleIndexes()
	if len(visible) == 0 {
		t.Fatalf("expected at least one match for 'hot'")
	}
	// Top match should contain "hotfix".
	if !strings.Contains(lm.items[visible[0]].wt.Path, "hotfix") {
		t.Errorf("top match should be hotfix, got %q", lm.items[visible[0]].wt.Path)
	}
}

func TestListSelectAddNew(t *testing.T) {
	m := tea.Model(NewListModel([]git.Worktree{{Path: "/repo", Branch: "main"}}, "/repo", ModeSelect))
	// Move down once → Add-new row
	m = sendKey(m, "down")
	m = sendKey(m, "enter")
	lm := m.(ListModel)
	if !lm.Result().AddNew {
		t.Errorf("expected AddNew to be true")
	}
}

func TestListCancel(t *testing.T) {
	m := tea.Model(NewListModel(sampleWorktrees(), "/repo", ModeSelect))
	m = sendKey(m, "esc")
	lm := m.(ListModel)
	if !lm.Result().Cancelled {
		t.Errorf("esc should cancel")
	}
}
