package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// init re-profiles lipgloss's default renderer against stderr. Lipgloss
// probes stdout by default, but our stdout is reserved for the chosen path
// (and is a pipe when invoked through the shell wrapper). Probing stdout
// would make termenv decide "not a TTY" and downgrade to ASCII, killing
// colors. `WithUnsafe` tells termenv to trust the $COLORTERM / $TERM env
// vars even when the target isn't directly a TTY — necessary because the
// shell wrapper uses a command substitution.
func init() {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr, termenv.WithUnsafe()))
}

// ttyOptions returns the Bubble Tea program options that route rendering to
// stderr (so stdout stays clean for the shell wrapper to consume the chosen
// path) while keeping stdin as the real *os.File — Bubble Tea needs the
// concrete file so it can detect the TTY and put it into raw mode. Wrapping
// either in a plain io.Reader/io.Writer breaks input (arrow keys arrive as
// literal escape sequences like `^[[B`).
func ttyOptions() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithOutput(os.Stderr),
		tea.WithInput(os.Stdin),
	}
}
