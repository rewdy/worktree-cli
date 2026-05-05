package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rewdy/worktree-cli/internal/git"
)

// ConfirmResult is the outcome of a confirm dialog.
type ConfirmResult struct {
	Confirmed bool
	Cancelled bool
}

// ConfirmModel is a simple two-button confirm dialog for destructive actions.
// Defaults focus to Remove; `y` confirms, `n`/esc cancel.
type ConfirmModel struct {
	worktree  git.Worktree
	choice    int // 0 = Cancel, 1 = Remove
	done      bool
	result    ConfirmResult
	termWidth int
}

// NewConfirmModel builds a confirm dialog for removing the given worktree.
// Focus defaults to the Remove button — users mostly arrive here having
// already signalled intent to remove, so Enter should do the obvious thing.
func NewConfirmModel(wt git.Worktree) ConfirmModel {
	return ConfirmModel{worktree: wt, choice: 1}
}

// Init implements tea.Model.
func (m ConfirmModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.termWidth = ws.Width
		return m, nil
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "ctrl+c", "esc", "q", "n", "N":
		m.result.Cancelled = true
		m.done = true
		return m, tea.Quit
	case "tab", "shift+tab", "left", "right", "h", "l":
		m.choice = 1 - m.choice
		return m, nil
	case "y", "Y":
		m.result.Confirmed = true
		m.done = true
		return m, tea.Quit
	case "enter":
		if m.choice == 1 {
			m.result.Confirmed = true
		} else {
			m.result.Cancelled = true
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// View implements tea.Model.
func (m ConfirmModel) View() string {
	if m.done {
		return ""
	}

	innerWidth := innerWidthFor(m.termWidth)
	var b strings.Builder

	b.WriteString(Header("Remove worktree?", StyleTitlePink, innerWidth))
	b.WriteString("\n\n")

	b.WriteString("This will remove the worktree at:\n\n")

	path := m.worktree.Path
	annotation := ""
	if m.worktree.Detached {
		annotation = "  " + StyleDetached.Render("("+m.worktree.DisplayBranch()+")")
	} else if m.worktree.BranchDiffersFromFolder() {
		annotation = "  " + StyleBranchAnnotation.Render("("+m.worktree.Branch+")")
	}
	b.WriteString("  " + StyleSelectedPath.Render(path) + annotation + "\n\n")

	b.WriteString(StyleSubtitle.Render("Git will refuse if the worktree has uncommitted changes.") + "\n\n")

	// Buttons.
	cancel := m.renderButton("Cancel", m.choice == 0, false)
	remove := m.renderButton("Remove", m.choice == 1, true)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, cancel, "   ", remove)
	b.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(buttons) + "\n")

	b.WriteString(m.helpLine())
	return "\n" + StyleFrame.Render(b.String()) + "\n"
}

// Result returns the user's choice; only meaningful after the program exits.
func (m ConfirmModel) Result() ConfirmResult { return m.result }

// renderButton styles one of the two buttons. `destructive=true` uses the
// pink palette; the Cancel button uses the muted palette. When focused, the
// button gets a bright border and bold text.
func (m ConfirmModel) renderButton(label string, focused, destructive bool) string {
	base := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Foreground(colorMuted)

	if focused {
		if destructive {
			return base.
				BorderForeground(colorAccent).
				Foreground(colorAccent).
				Bold(true).
				Render(label)
		}
		return base.
			BorderForeground(colorPrimary).
			Foreground(colorPrimary).
			Bold(true).
			Render(label)
	}
	return base.Render(label)
}

func (m ConfirmModel) helpLine() string {
	parts := []string{"tab/←→: switch", "enter: confirm", "y: remove", "n/esc: cancel"}
	return "\n" + StyleHelp.Render(strings.Join(parts, "  •  "))
}

// RunConfirm shows the confirm dialog and returns the user's choice.
func RunConfirm(model ConfirmModel) (ConfirmResult, error) {
	p := tea.NewProgram(model, ttyOptions()...)
	finalModel, err := p.Run()
	if err != nil {
		return ConfirmResult{}, err
	}
	cm, ok := finalModel.(ConfirmModel)
	if !ok {
		return ConfirmResult{}, fmt.Errorf("unexpected model type")
	}
	return cm.Result(), nil
}
