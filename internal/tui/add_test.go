package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func typeString(m tea.Model, s string) tea.Model {
	for _, r := range s {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		next, _ := m.Update(msg)
		m = next
	}
	return m
}

func TestAddDefaultPathPrefilled(t *testing.T) {
	m := NewAddModel("main", "", "../")
	if m.pathInput.Value() != "../" {
		t.Errorf("path should pre-fill with '../', got %q", m.pathInput.Value())
	}
}

func TestAddSubmitWithDefaultBase(t *testing.T) {
	m := tea.Model(NewAddModel("main", "", "../"))
	m = typeString(m, "feature-xyz")
	// Tab → branch
	m = sendKey(m, "tab")
	m = typeString(m, "my-branch")
	// Tab → base selector (already on "main")
	m = sendKey(m, "tab")
	// Enter submits
	m = sendKey(m, "enter")
	am := m.(AddModel)
	if !am.Result().Submitted {
		t.Fatalf("expected submission, got cancelled")
	}
	opts := am.Result().Options
	if opts.Path != "../feature-xyz" {
		t.Errorf("path: got %q", opts.Path)
	}
	if opts.Branch != "my-branch" {
		t.Errorf("branch: got %q", opts.Branch)
	}
	if opts.Base != "main" {
		t.Errorf("base: got %q", opts.Base)
	}
}

func TestAddOtherBaseRequiresValue(t *testing.T) {
	m := tea.Model(NewAddModel("main", "", "../"))
	m = typeString(m, "work")
	m = sendKey(m, "tab")   // → branch
	m = sendKey(m, "tab")   // → base
	m = sendKey(m, "right") // default → other (no current branch shown)
	m = sendKey(m, "enter") // enter on base+other moves focus into other input
	m = sendKey(m, "enter") // enter again with empty other → validation error
	am := m.(AddModel)
	if am.Result().Submitted {
		t.Errorf("should not submit with empty 'other' base")
	}
	if am.errMsg == "" {
		t.Errorf("expected error message for empty other")
	}
}

func TestAddCurrentBranchHiddenWhenSameAsDefault(t *testing.T) {
	m := NewAddModel("main", "main", "../")
	if m.showCurrent {
		t.Errorf("current branch pill should be hidden when same as default")
	}
}

func TestAddCurrentBranchShownWhenDifferent(t *testing.T) {
	m := NewAddModel("main", "feature-x", "../")
	if !m.showCurrent {
		t.Errorf("current branch pill should show when different from default")
	}
}

func TestAddCancel(t *testing.T) {
	m := tea.Model(NewAddModel("main", "", "../"))
	m = sendKey(m, "esc")
	am := m.(AddModel)
	if !am.Result().Cancelled {
		t.Errorf("esc should cancel")
	}
}

func TestAddEmptyPathValidation(t *testing.T) {
	m := tea.Model(NewAddModel("main", "", "../"))
	// Path is "../" by default — submit without typing should fail validation.
	m = sendKey(m, "tab")   // path → branch
	m = sendKey(m, "tab")   // branch → base
	m = sendKey(m, "enter") // submit
	am := m.(AddModel)
	if am.Result().Submitted {
		t.Errorf("should not submit with just '../'")
	}
	if am.errMsg == "" {
		t.Errorf("expected validation error")
	}
}
