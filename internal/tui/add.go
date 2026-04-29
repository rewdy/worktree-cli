package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rewdy/worktree-tool/internal/git"
)

// AddResult is what the add form returns after the user submits or cancels.
type AddResult struct {
	Submitted bool
	Cancelled bool
	Options   git.AddOptions
}

// focusField identifies which part of the add form currently has focus.
type focusField int

const (
	focusPath focusField = iota
	focusBranch
	focusBase   // the segmented selector
	focusOther  // the "other base" text input (only when the Other pill is active)
	focusCount_ // sentinel; keep last
)

// baseChoice represents one of the base-branch pills.
type baseChoice int

const (
	baseDefault baseChoice = iota // main/master
	baseCurrent                   // current branch (hidden if same as default)
	baseOther                     // free-form
)

// AddModel is the Bubble Tea model for the add-worktree form.
type AddModel struct {
	pathInput   textinput.Model
	branchInput textinput.Model
	otherInput  textinput.Model

	defaultBase string // "main" or "master"
	currentBase string // current branch, empty if == defaultBase or detached
	showCurrent bool

	focus      focusField
	baseChoice baseChoice

	result AddResult
	done   bool

	errMsg string
}

// NewAddModel constructs the add form. Pass in the detected default branch
// ("main"/"master") and the current branch name (empty if detached).
func NewAddModel(defaultBranch, currentBranch string) AddModel {
	path := textinput.New()
	path.Placeholder = "../new-worktree"
	path.SetValue("../")
	path.CursorEnd()
	path.Prompt = ""
	path.CharLimit = 500
	path.Focus()

	branch := textinput.New()
	branch.Placeholder = "(blank = use folder name)"
	branch.Prompt = ""
	branch.CharLimit = 200

	other := textinput.New()
	other.Placeholder = "branch, tag, or SHA"
	other.Prompt = ""
	other.CharLimit = 200

	showCurrent := currentBranch != "" && currentBranch != defaultBranch

	return AddModel{
		pathInput:   path,
		branchInput: branch,
		otherInput:  other,
		defaultBase: defaultBranch,
		currentBase: currentBranch,
		showCurrent: showCurrent,
		focus:       focusPath,
		baseChoice:  baseDefault,
	}
}

// Init implements tea.Model.
func (m AddModel) Init() tea.Cmd { return textinput.Blink }

// Update implements tea.Model.
func (m AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.result.Cancelled = true
			m.done = true
			return m, tea.Quit

		case "tab", "shift+tab":
			forward := msg.String() == "tab"
			m = m.moveFocus(forward)
			return m, textinput.Blink

		case "enter":
			// On the base row, Enter acts as "next field" unless we're on the
			// final field, in which case it submits.
			if m.focus == focusPath || m.focus == focusBranch {
				m = m.moveFocus(true)
				return m, textinput.Blink
			}
			if m.focus == focusBase && m.baseChoice == baseOther {
				// Move into the Other text input first.
				m.focus = focusOther
				m.otherInput.Focus()
				m.pathInput.Blur()
				m.branchInput.Blur()
				return m, textinput.Blink
			}
			if ok, err := m.validate(); !ok {
				m.errMsg = err
				return m, nil
			}
			m.submit()
			return m, tea.Quit

		case "left", "h":
			if m.focus == focusBase {
				m.baseChoice = m.prevBase()
				return m, nil
			}
		case "right", "l":
			if m.focus == focusBase {
				m.baseChoice = m.nextBase()
				return m, nil
			}
		}
	}

	// Delegate to the focused text input.
	var cmd tea.Cmd
	switch m.focus {
	case focusPath:
		m.pathInput, cmd = m.pathInput.Update(msg)
	case focusBranch:
		m.branchInput, cmd = m.branchInput.Update(msg)
	case focusOther:
		m.otherInput, cmd = m.otherInput.Update(msg)
	}
	return m, cmd
}

// View implements tea.Model.
func (m AddModel) View() string {
	if m.done {
		return ""
	}

	const innerWidth = 76
	var b strings.Builder

	b.WriteString(Header("Add a new worktree", StyleTitleTeal, innerWidth))
	b.WriteString("\n\n")

	// Path
	b.WriteString(m.labelLine("Path", "where the new worktree will live"))
	b.WriteString(m.inputBox(m.pathInput, m.focus == focusPath))
	b.WriteString("\n\n")

	// Branch
	b.WriteString(m.labelLine("Branch", "new branch name — leave blank to use the folder name"))
	b.WriteString(m.inputBox(m.branchInput, m.focus == focusBranch))
	b.WriteString("\n\n")

	// Base
	b.WriteString(m.labelLine("Base", "what to branch off of"))
	b.WriteString(m.basePills())
	b.WriteString("\n")
	if m.baseChoice == baseOther {
		b.WriteString("\n")
		b.WriteString(m.inputBox(m.otherInput, m.focus == focusOther))
		b.WriteString("\n")
	}

	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(StyleError.Render("✗ " + m.errMsg))
		b.WriteString("\n")
	}

	b.WriteString(m.helpLine())
	return "\n" + StyleFrame.Render(b.String()) + "\n"
}

// Result returns the form's outcome after tea exits.
func (m AddModel) Result() AddResult { return m.result }

// --- focus + navigation -------------------------------------------------

func (m AddModel) moveFocus(forward bool) AddModel {
	// Build ordered list of reachable fields based on current state.
	order := []focusField{focusPath, focusBranch, focusBase}
	if m.baseChoice == baseOther {
		order = append(order, focusOther)
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
	m.branchInput.Blur()
	m.otherInput.Blur()
	switch m.focus {
	case focusPath:
		m.pathInput.Focus()
	case focusBranch:
		m.branchInput.Focus()
	case focusOther:
		m.otherInput.Focus()
	}
	return m
}

func (m AddModel) prevBase() baseChoice {
	choices := m.baseChoicesInOrder()
	for i, c := range choices {
		if c == m.baseChoice {
			return choices[(i-1+len(choices))%len(choices)]
		}
	}
	return m.baseChoice
}

func (m AddModel) nextBase() baseChoice {
	choices := m.baseChoicesInOrder()
	for i, c := range choices {
		if c == m.baseChoice {
			return choices[(i+1)%len(choices)]
		}
	}
	return m.baseChoice
}

func (m AddModel) baseChoicesInOrder() []baseChoice {
	if m.showCurrent {
		return []baseChoice{baseDefault, baseCurrent, baseOther}
	}
	return []baseChoice{baseDefault, baseOther}
}

// --- rendering helpers --------------------------------------------------

func (m AddModel) labelLine(label, hint string) string {
	return StyleLabel.Render(label) + "   " + StyleSubtitle.Render(hint) + "\n"
}

func (m AddModel) inputBox(ti textinput.Model, focused bool) string {
	style := StyleInput
	if focused {
		style = StyleInputFocused
	}
	// Target a consistent inner width.
	const innerWidth = 60
	ti.Width = innerWidth
	return style.Width(innerWidth + 2).Render(ti.View())
}

func (m AddModel) basePills() string {
	renderPill := func(label string, choice baseChoice) string {
		active := m.baseChoice == choice
		focused := m.focus == focusBase
		switch {
		case active && focused:
			return StylePillActive.Render(label)
		case active:
			return StylePillSelected.Render(label)
		default:
			return StylePill.Render(label)
		}
	}

	pills := []string{renderPill(m.defaultBase, baseDefault)}
	if m.showCurrent {
		pills = append(pills, renderPill(m.currentBase, baseCurrent))
	}
	pills = append(pills, renderPill("Other…", baseOther))

	return lipgloss.JoinHorizontal(lipgloss.Top, pills...)
}

func (m AddModel) helpLine() string {
	parts := []string{"tab: next field", "enter: next / submit", "←→: pick base", "esc: cancel"}
	return "\n" + StyleHelp.Render(strings.Join(parts, "  •  "))
}

// --- validation + submission --------------------------------------------

func (m AddModel) validate() (bool, string) {
	path := strings.TrimSpace(m.pathInput.Value())
	if path == "" || path == "../" {
		return false, "please enter a path"
	}
	if m.baseChoice == baseOther && strings.TrimSpace(m.otherInput.Value()) == "" {
		return false, "please enter a base branch/committish (or pick a different base)"
	}
	return true, ""
}

func (m *AddModel) submit() {
	opts := git.AddOptions{
		Path:   strings.TrimSpace(m.pathInput.Value()),
		Branch: strings.TrimSpace(m.branchInput.Value()),
	}
	switch m.baseChoice {
	case baseDefault:
		opts.Base = m.defaultBase
	case baseCurrent:
		opts.Base = m.currentBase
	case baseOther:
		opts.Base = strings.TrimSpace(m.otherInput.Value())
	}
	m.result.Submitted = true
	m.result.Options = opts
	m.done = true
}

// RunAdd renders the add form and returns the user's choices.
func RunAdd(model AddModel) (AddResult, error) {
	p := tea.NewProgram(model, ttyOptions()...)
	finalModel, err := p.Run()
	if err != nil {
		return AddResult{}, err
	}
	am, ok := finalModel.(AddModel)
	if !ok {
		return AddResult{}, fmt.Errorf("unexpected model type")
	}
	return am.Result(), nil
}
