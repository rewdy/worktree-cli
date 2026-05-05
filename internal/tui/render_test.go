package tui

import (
	"strings"
	"testing"
)

// stripANSI removes ANSI escape sequences so we can assert on visible text.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			// Terminate on a letter (typical SGR end).
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestListViewRenders(t *testing.T) {
	m := NewListModel(sampleWorktrees(), "/repo", ModeSelect, false)
	view := stripANSI(m.View())

	// Paths should appear
	for _, want := range []string{"/repo", "/work/feat-x", "/work/hotfix-123", "/work/experiment"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
	// Branch annotation should appear for hotfix-123 (folder != branch)
	if !strings.Contains(view, "release/2026-04") {
		t.Errorf("view should show differing branch annotation")
	}
	// No annotation for feat-x whose branch is feature-x (differs → shown) — actually differs, expect it.
	if !strings.Contains(view, "feature-x") {
		t.Errorf("view should show feature-x annotation")
	}
	// Detached annotation
	if !strings.Contains(view, "detached") {
		t.Errorf("view should show detached annotation")
	}
	// Add-new row present in select mode
	if !strings.Contains(view, "Add new worktree") {
		t.Errorf("view should contain add-new row")
	}
}

func TestAddViewRenders(t *testing.T) {
	m := NewAddModel("main", "feature-x", "../")
	view := stripANSI(m.View())

	for _, want := range []string{"Path", "Branch", "Base", "main", "feature-x", "Other"} {
		if !strings.Contains(view, want) {
			t.Errorf("add view missing %q:\n%s", want, view)
		}
	}
}
