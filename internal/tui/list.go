package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/rewdy/worktree-cli/internal/git"
)

// ListMode controls which special rows are shown and what selection means.
type ListMode int

const (
	// ModeSelect renders an "Add new" row at the bottom and returns either a
	// selected worktree path or a signal that the user chose "add new".
	ModeSelect ListMode = iota
	// ModeRemove shows only existing worktrees and excludes the current one.
	ModeRemove
)

// ListResult is returned via the program model after Run() finishes.
type ListResult struct {
	// Selected is the absolute path of the chosen worktree. Empty if the user
	// cancelled or selected "add new".
	Selected string
	// SelectedWorktree is the full record for Selected — used by callers that
	// need more than just the path (e.g. the confirm dialog).
	SelectedWorktree git.Worktree
	// AddNew is true when the user picked the "＋ Add new worktree" row.
	AddNew bool
	// Remove is true when the user pressed the remove shortcut on a row.
	// Selected / SelectedWorktree point at the row they asked to remove.
	Remove bool
	// Unlock is true when the user pressed `u` on a locked worktree row.
	// Selected / SelectedWorktree point at that row.
	Unlock bool
	// Cancelled is true when the user quit without selecting.
	Cancelled bool
}

type listItem struct {
	wt      git.Worktree
	current bool
	addNew  bool
}

func (li listItem) searchString() string {
	if li.addNew {
		return "add new worktree"
	}
	return li.wt.Path + " " + li.wt.DisplayBranch()
}

// ListModel is the Bubble Tea model for the selector.
type ListModel struct {
	mode         ListMode
	items        []listItem
	filter       textinput.Model
	cursor       int
	filtering    bool
	filtered     []int // indexes into items; nil means no filter active
	result       ListResult
	done         bool
	emptyMessage string
}

// NewListModel constructs a ListModel. Pass the current worktree path so the
// model can mark which row you're on and (for ModeRemove) exclude it.
func NewListModel(worktrees []git.Worktree, currentPath string, mode ListMode) ListModel {
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.Prompt = ""
	ti.CharLimit = 200

	items := make([]listItem, 0, len(worktrees)+1)
	for _, w := range worktrees {
		isCurrent := samePath(w.Path, currentPath)
		if mode == ModeRemove && isCurrent {
			// Prevent foot-gun: don't let user remove the worktree they're in.
			continue
		}
		items = append(items, listItem{wt: w, current: isCurrent})
	}

	// Sort: current first (when present), then by path for stable UX.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].current != items[j].current {
			return items[i].current
		}
		return items[i].wt.Path < items[j].wt.Path
	})

	if mode == ModeSelect {
		items = append(items, listItem{addNew: true})
	}

	empty := "No worktrees found."
	if mode == ModeRemove {
		empty = "No removable worktrees (you can't remove the one you're in)."
	}

	return ListModel{
		mode:         mode,
		items:        items,
		filter:       ti,
		emptyMessage: empty,
	}
}

// Init implements tea.Model.
func (m ListModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When filtering, most keys go into the filter input; only Enter/Esc/arrows escape.
		if m.filtering {
			switch msg.String() {
			case "esc":
				m.filtering = false
				m.filter.Blur()
				m.filter.SetValue("")
				m.applyFilter()
				return m, nil
			case "enter":
				nm, cmd := m.selectCurrent()
				return nm, cmd
			case "up", "ctrl+p":
				return m.moveCursor(-1), nil
			case "down", "ctrl+n":
				return m.moveCursor(1), nil
			case "ctrl+c":
				m.result.Cancelled = true
				m.done = true
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.applyFilter()
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.result.Cancelled = true
			m.done = true
			return m, tea.Quit
		case "up", "k", "ctrl+p":
			return m.moveCursor(-1), nil
		case "down", "j", "ctrl+n":
			return m.moveCursor(1), nil
		case "enter":
			nm, cmd := m.selectCurrent()
			return nm, cmd
		case "x", "X":
			nm, cmd := m.requestRemove()
			return nm, cmd
		case "u", "U":
			nm, cmd := m.requestUnlock()
			return nm, cmd
		case "/":
			m.filtering = true
			m.filter.Focus()
			return m, textinput.Blink
		default:
			// Any printable rune flips into filter mode with that rune pre-typed.
			if len(msg.Runes) == 1 && msg.Runes[0] >= 32 && msg.Runes[0] != 127 {
				m.filtering = true
				m.filter.Focus()
				m.filter.SetValue(string(msg.Runes))
				m.filter.CursorEnd()
				m.applyFilter()
				return m, textinput.Blink
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ListModel) View() string {
	if m.done {
		return ""
	}

	const innerWidth = 76
	var b strings.Builder

	// Header: teal title (or pink for remove) + rainbow underline.
	titleStyle := StyleTitleTeal
	if m.mode == ModeRemove {
		titleStyle = StyleTitlePink
	}
	b.WriteString(Header(m.titleText(), titleStyle, innerWidth))
	b.WriteString("\n\n")

	visible := m.visibleIndexes()
	if len(visible) == 0 {
		if m.filtering && m.filter.Value() != "" {
			b.WriteString(StyleRow.Render(StyleSubtitle.Render("no matches")))
			b.WriteString("\n")
		} else {
			b.WriteString(StyleRow.Render(StyleSubtitle.Render(m.emptyMessage)))
			b.WriteString("\n")
		}
	} else {
		for rowIdx, itemIdx := range visible {
			item := m.items[itemIdx]
			// Blank spacer before the Add-new row to set it off visually.
			if item.addNew && rowIdx > 0 {
				b.WriteString("\n")
			}
			b.WriteString(m.renderRow(item, rowIdx == m.cursor, innerWidth))
			b.WriteString("\n")
		}
	}

	if m.filtering {
		b.WriteString("\n")
		b.WriteString(StyleRow.Render(StyleFilter.Render("/ ") + m.filter.View()))
		b.WriteString("\n")
	}

	b.WriteString(m.helpLine())
	return "\n" + StyleFrame.Render(b.String()) + "\n"
}

// --- result + helpers ---------------------------------------------------

// Result returns the user's selection; only meaningful after the program exits.
func (m ListModel) Result() ListResult { return m.result }

func (m ListModel) selectCurrent() (ListModel, tea.Cmd) {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		return m, nil
	}
	if m.cursor < 0 || m.cursor >= len(visible) {
		return m, nil
	}
	item := m.items[visible[m.cursor]]
	if item.addNew {
		m.result.AddNew = true
	} else {
		m.result.Selected = item.wt.Path
		m.result.SelectedWorktree = item.wt
	}
	m.done = true
	return m, tea.Quit
}

// requestRemove marks the cursor row for removal (ModeSelect only) and
// exits. Silently ignores the request on the "Add new" row or the current
// worktree (which the caller may not want to delete).
func (m ListModel) requestRemove() (ListModel, tea.Cmd) {
	if m.mode != ModeSelect {
		return m, nil
	}
	visible := m.visibleIndexes()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return m, nil
	}
	item := m.items[visible[m.cursor]]
	if item.addNew || item.current {
		return m, nil
	}
	m.result.Selected = item.wt.Path
	m.result.SelectedWorktree = item.wt
	m.result.Remove = true
	m.done = true
	return m, tea.Quit
}

// requestUnlock marks the cursor row for unlocking and exits. No-op unless
// the row is a locked worktree (not the "Add new" row).
func (m ListModel) requestUnlock() (ListModel, tea.Cmd) {
	visible := m.visibleIndexes()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return m, nil
	}
	item := m.items[visible[m.cursor]]
	if item.addNew || !item.wt.Locked {
		return m, nil
	}
	m.result.Selected = item.wt.Path
	m.result.SelectedWorktree = item.wt
	m.result.Unlock = true
	m.done = true
	return m, tea.Quit
}

func (m ListModel) moveCursor(delta int) ListModel {
	visible := m.visibleIndexes()
	if len(visible) == 0 {
		m.cursor = 0
		return m
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(visible) {
		m.cursor = len(visible) - 1
	}
	return m
}

func (m *ListModel) applyFilter() {
	q := strings.TrimSpace(m.filter.Value())
	if q == "" {
		m.filtered = nil
		if m.cursor >= len(m.items) {
			m.cursor = len(m.items) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return
	}
	// Build searchable strings, run fuzzy match, store matched indexes.
	strs := make([]string, len(m.items))
	for i, it := range m.items {
		strs[i] = it.searchString()
	}
	matches := fuzzy.Find(q, strs)
	idxs := make([]int, len(matches))
	for i, m := range matches {
		idxs[i] = m.Index
	}
	m.filtered = idxs
	if m.cursor >= len(idxs) {
		m.cursor = len(idxs) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m ListModel) visibleIndexes() []int {
	if m.filtered != nil {
		return m.filtered
	}
	out := make([]int, len(m.items))
	for i := range m.items {
		out[i] = i
	}
	return out
}

// titleText returns the header title without the old leading-spaces padding.
func (m ListModel) titleText() string {
	if m.mode == ModeRemove {
		return "Remove a worktree"
	}
	return "Select a worktree"
}

// cursorGlyph returns the two-character cursor (or its blank-space placeholder
// when the row isn't selected), colored appropriately for the mode.
func (m ListModel) cursorGlyph(selected bool) string {
	if !selected {
		return "  "
	}
	if m.mode == ModeRemove {
		return StyleRowCursorRemove.Render("✗ ")
	}
	return StyleRowCursor.Render("▸ ")
}

// underlineHex returns the hex color used for the selected-row underline.
// Violet for select mode, pink for remove mode.
func (m ListModel) underlineHex() string {
	if m.mode == ModeRemove {
		return "#EC4899" // pink
	}
	return "#8B5CF6" // violet
}

func (m ListModel) renderRow(item listItem, selected bool, innerWidth int) string {
	cursor := m.cursorGlyph(selected)

	if item.addNew {
		content := StyleAddNew.Render("＋ Add new worktree  " + Unicorn)
		line := cursor + content
		if selected {
			return StyleRow.Render(UnderlineWithColor(line, m.underlineHex()))
		}
		return StyleRow.Render(line)
	}

	// In ModeSelect we reserve a slot for the current-indicator dot so rows
	// align. In ModeRemove the current worktree is filtered out, so no slot.
	dot := ""
	if m.mode == ModeSelect {
		if item.current {
			dot = StyleCurrentDot.Render("● ")
		} else {
			dot = "  "
		}
	}

	annotation := ""
	if item.wt.Detached {
		annotation = "  " + StyleDetached.Render("("+item.wt.DisplayBranch()+")")
	} else if item.wt.BranchDiffersFromFolder() {
		annotation = "  " + StyleBranchAnnotation.Render("("+item.wt.Branch+")")
	}

	lock := ""
	if item.wt.Locked {
		lock = "  🔒"
	}

	path := item.wt.Path
	if selected {
		path = StyleSelectedPath.Render(path)
	}

	line := cursor + dot + path + lock + annotation
	if selected {
		return StyleRow.Render(UnderlineWithColor(line, m.underlineHex()))
	}
	return StyleRow.Render(line)
}

func (m ListModel) helpLine() string {
	parts := []string{}
	if m.filtering {
		parts = append(parts, "enter: select", "esc: clear filter", "↑↓: move", "ctrl+c: quit")
	} else {
		parts = append(parts, "↑↓: move", "enter: select", "/ or type: filter")
		if m.cursorOnLocked() {
			parts = append(parts, "u: unlock")
		}
		if m.mode == ModeSelect {
			parts = append(parts, "x: remove")
		}
		parts = append(parts, "q: quit")
	}
	return StyleHelp.Render(strings.Join(parts, "  •  "))
}

// cursorOnLocked reports whether the currently highlighted row is a locked
// worktree. Used to conditionally show the `u: unlock` hint.
func (m ListModel) cursorOnLocked() bool {
	visible := m.visibleIndexes()
	if m.cursor < 0 || m.cursor >= len(visible) {
		return false
	}
	item := m.items[visible[m.cursor]]
	return !item.addNew && item.wt.Locked
}

// --- utility ------------------------------------------------------------

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

// RunList runs the list model and returns the user's selection. TUI renders
// to stderr so stdout stays clean for the shell-wrapper path passing.
func RunList(model ListModel) (ListResult, error) {
	p := tea.NewProgram(model, ttyOptions()...)
	finalModel, err := p.Run()
	if err != nil {
		return ListResult{}, err
	}
	lm, ok := finalModel.(ListModel)
	if !ok {
		return ListResult{}, fmt.Errorf("unexpected model type")
	}
	return lm.Result(), nil
}
