package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// spinnerDoneMsg signals the background worker finished.
type spinnerDoneMsg struct{}

type spinnerModel struct {
	spinner spinner.Model
	label   string
}

func (m spinnerModel) Init() tea.Cmd { return m.spinner.Tick }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinnerDoneMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		// Swallow keys during the operation — don't let Ctrl+C corrupt a
		// half-finished `git worktree remove`.
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	label := lipgloss.NewStyle().Foreground(colorMuted).Render(m.label)
	return "  " + m.spinner.View() + " " + label + "\n"
}

// RunWithSpinner shows an animated spinner with `label` while `work` runs in a
// goroutine. The spinner renders to stderr so stdout stays clean. Callers
// capture any outputs from `work` via closure variables.
func RunWithSpinner(label string, work func()) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorAccent)

	p := tea.NewProgram(spinnerModel{spinner: sp, label: label}, ttyOptions()...)

	go func() {
		work()
		p.Send(spinnerDoneMsg{})
	}()

	_, err := p.Run()
	return err
}
