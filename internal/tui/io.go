package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// init re-profiles lipgloss's default renderer so colors render correctly
// when the TUI goes to stderr. By default lipgloss profiles stdout, which
// could be anything depending on how we're invoked; we explicitly point
// the renderer at stderr and set the color profile from $TERM /
// $COLORTERM rather than letting termenv probe. This matches how most
// interactive programs decide and avoids edge cases where termenv's TTY
// check fails and everything downgrades to ASCII.
func init() {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stderr, termenv.WithUnsafe()))
	lipgloss.SetColorProfile(detectProfile())
}

// detectProfile picks a termenv color profile from the environment.
func detectProfile() termenv.Profile {
	if strings.EqualFold(os.Getenv("COLORTERM"), "truecolor") ||
		strings.EqualFold(os.Getenv("COLORTERM"), "24bit") {
		return termenv.TrueColor
	}
	term := os.Getenv("TERM")
	switch {
	case strings.Contains(term, "truecolor"), strings.Contains(term, "direct"):
		return termenv.TrueColor
	case strings.Contains(term, "256"):
		return termenv.ANSI256
	case term == "", term == "dumb":
		return termenv.Ascii
	default:
		return termenv.ANSI
	}
}

// ttyOptions returns the Bubble Tea program options that route rendering
// to stderr, keeping stdin as the real *os.File so Bubble Tea can detect
// the TTY and put it into raw mode. Wrapping either in a plain
// io.Reader/io.Writer breaks input (arrow keys arrive as literal escape
// sequences like `^[[B`).
func ttyOptions() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithOutput(os.Stderr),
		tea.WithInput(os.Stdin),
	}
}

// frameChrome is the total horizontal space consumed by StyleFrame's
// rounded border (1 col each side) plus its padding (2 cols each side).
// Inner content width is terminal width minus this.
const frameChrome = 6

// defaultInnerWidth is used before the first WindowSizeMsg arrives and as
// a floor so narrow terminals don't collapse the layout entirely.
const defaultInnerWidth = 76

// innerWidthFor clamps the inner width for a given terminal width. Falls
// back to defaultInnerWidth when termWidth is 0 (pre-size message).
func innerWidthFor(termWidth int) int {
	if termWidth <= 0 {
		return defaultInnerWidth
	}
	w := termWidth - frameChrome
	if w < 20 {
		return 20
	}
	return w
}
