package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

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
