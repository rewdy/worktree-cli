package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rewdy/worktree-cli/internal/settings"
)

// SettingsResult is the outcome of the settings modal.
type SettingsResult struct {
	Saved    bool
	Settings settings.Settings
}

// settingsFocus identifies which element of the settings modal has focus.
type settingsFocus int

const (
	settingsFocusPath settingsFocus = iota
	settingsFocusCollapse
	settingsFocusCancel
	settingsFocusSave
)

// SettingsModel is a modal form for editing user preferences.
type SettingsModel struct {
	pathInput     textinput.Model
	collapsePaths bool

	focus     settingsFocus
	done      bool
	result    SettingsResult
	termWidth int
}

// NewSettingsModel builds a settings modal pre-populated with the given
// current values.
func NewSettingsModel(current settings.Settings) SettingsModel {
	ti := textinput.New()
	ti.Placeholder = "../{project-name}-worktrees/"
	ti.Prompt = ""
	ti.CharLimit = 500
	ti.SetValue(current.DefaultPathTemplate)
	ti.CursorEnd()
	ti.Focus()

	return SettingsModel{
		pathInput:     ti,
		collapsePaths: current.CollapsePaths,
		focus:         settingsFocusPath,
	}
}

// Init implements tea.Model.
func (m SettingsModel) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit

		case "tab":
			m = m.moveFocus(true)
			return m, textinput.Blink
		case "shift+tab":
			m = m.moveFocus(false)
			return m, textinput.Blink

		case " ":
			if m.focus == settingsFocusCollapse {
				m.collapsePaths = !m.collapsePaths
				return m, nil
			}
		case "left", "right", "h", "l":
			if m.focus == settingsFocusCollapse {
				m.collapsePaths = !m.collapsePaths
				return m, nil
			}

		case "enter":
			switch m.focus {
			case settingsFocusPath, settingsFocusCollapse:
				m = m.moveFocus(true)
				return m, textinput.Blink
			case settingsFocusCancel:
				m.done = true
				return m, tea.Quit
			case settingsFocusSave:
				m.commit()
				return m, tea.Quit
			}
		}
	}

	if m.focus == settingsFocusPath {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View implements tea.Model.
func (m SettingsModel) View() string {
	if m.done {
		return ""
	}

	innerWidth := innerWidthFor(m.termWidth)
	var b strings.Builder

	b.WriteString(Header("Settings", StyleTitleTeal, innerWidth))
	b.WriteString("\n\n")

	// Default path template.
	b.WriteString(m.labelLine("Default path", "used to pre-fill the Add form — vars: {project-name}, {branch}"))
	b.WriteString(m.inputBox(m.pathInput, m.focus == settingsFocusPath))
	b.WriteString("\n\n")

	// Collapse common path prefix toggle.
	b.WriteString(m.labelLine("Collapse common path", "shorten list view by eliding the shared path prefix"))
	b.WriteString(m.togglePills())
	b.WriteString("\n\n")

	// Separator between form fields and the action buttons.
	b.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("─", innerWidth)))
	b.WriteString("\n\n")

	// Buttons — flush-left with the form fields above.
	cancel := m.renderButton("Cancel", m.focus == settingsFocusCancel, false)
	save := m.renderButton("Save", m.focus == settingsFocusSave, true)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, cancel, "   ", save)
	b.WriteString(buttons + "\n")

	b.WriteString(m.helpLine())
	return "\n" + StyleFrame.Render(b.String()) + "\n"
}

// Result returns the modal's outcome after the program exits.
func (m SettingsModel) Result() SettingsResult { return m.result }

// --- focus + navigation -------------------------------------------------

func (m SettingsModel) moveFocus(forward bool) SettingsModel {
	order := []settingsFocus{
		settingsFocusPath,
		settingsFocusCollapse,
		settingsFocusCancel,
		settingsFocusSave,
	}
	idx := 0
	for i, f := range order {
		if f == m.focus {
			idx = i
			break
		}
	}
	if forward {
		idx = (idx + 1) % len(order)
	} else {
		idx = (idx - 1 + len(order)) % len(order)
	}
	m.focus = order[idx]

	m.pathInput.Blur()
	if m.focus == settingsFocusPath {
		m.pathInput.Focus()
	}
	return m
}

// --- rendering helpers --------------------------------------------------

func (m SettingsModel) labelLine(label, hint string) string {
	return StyleLabel.Render(label) + "   " + StyleSubtitle.Render(hint) + "\n"
}

func (m SettingsModel) inputBox(ti textinput.Model, focused bool) string {
	style := StyleInput
	if focused {
		style = StyleInputFocused
	}
	const innerWidth = 60
	ti.Width = innerWidth
	return style.Width(innerWidth + 2).Render(ti.View())
}

func (m SettingsModel) togglePills() string {
	renderPill := func(label string, on bool) string {
		focused := m.focus == settingsFocusCollapse
		switch {
		case on && focused:
			return StylePillActive.Render(label)
		case on:
			return StylePillSelected.Render(label)
		default:
			return StylePill.Render(label)
		}
	}
	off := renderPill("Off", !m.collapsePaths)
	on := renderPill("On", m.collapsePaths)
	return lipgloss.JoinHorizontal(lipgloss.Top, off, on)
}

// renderButton mirrors ConfirmModel.renderButton — compact rounded buttons
// with a focused bright state.
func (m SettingsModel) renderButton(label string, focused, primary bool) string {
	base := lipgloss.NewStyle().
		Padding(0, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Foreground(colorMuted)

	if focused {
		if primary {
			return base.
				BorderForeground(colorPrimary).
				Foreground(colorPrimary).
				Bold(true).
				Render(label)
		}
		return base.
			BorderForeground(colorMuted).
			Foreground(colorMuted).
			Bold(true).
			Render(label)
	}
	return base.Render(label)
}

func (m SettingsModel) helpLine() string {
	parts := []string{"tab: next", "space/←→: toggle", "enter: confirm", "esc: cancel"}
	return "\n" + StyleHelp.Render(strings.Join(parts, "  •  "))
}

// --- submission ---------------------------------------------------------

func (m *SettingsModel) commit() {
	m.result.Saved = true
	m.result.Settings = settings.Settings{
		DefaultPathTemplate: strings.TrimSpace(m.pathInput.Value()),
		CollapsePaths:       m.collapsePaths,
	}
	m.done = true
}

// RunSettings renders the settings modal and returns the user's choice.
func RunSettings(model SettingsModel) (SettingsResult, error) {
	p := tea.NewProgram(model, ttyOptions()...)
	finalModel, err := p.Run()
	if err != nil {
		return SettingsResult{}, err
	}
	sm, ok := finalModel.(SettingsModel)
	if !ok {
		return SettingsResult{}, fmt.Errorf("unexpected model type")
	}
	return sm.Result(), nil
}
