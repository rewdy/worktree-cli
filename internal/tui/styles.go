package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme — a soft, dreamy palette with rainbow accents for unicorn vibes.
var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "#8B5CF6", Dark: "#C4B5FD"} // violet
	colorAccent    = lipgloss.AdaptiveColor{Light: "#EC4899", Dark: "#F9A8D4"} // pink
	colorTeal      = lipgloss.AdaptiveColor{Light: "#0D9488", Dark: "#5EEAD4"} // teal
	colorMint      = lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#6EE7B7"} // mint
	colorSky       = lipgloss.AdaptiveColor{Light: "#0EA5E9", Dark: "#7DD3FC"} // sky
	colorGold      = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FCD34D"} // gold
	colorMuted     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorDim       = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"}
	colorFg        = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#F9FAFB"}
	colorDanger    = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#FCA5A5"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "#DDD6FE", Dark: "#5B21B6"} // violet selection bg
)

// Style bundles used across screens.
var (
	// StyleFrame is the outer purple border that wraps every screen.
	StyleFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	StyleTitleTeal = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTeal)

	StyleTitlePink = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	StyleRow = lipgloss.NewStyle().
			Padding(0, 1)

	StyleSelectedPath = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	StyleRowCursor = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	StyleRowCursorRemove = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)

	StyleCurrentDot = lipgloss.NewStyle().
			Foreground(colorMint).
			Bold(true)

	StyleBranchAnnotation = lipgloss.NewStyle().
				Foreground(colorSky).
				Italic(true)

	StyleDetached = lipgloss.NewStyle().
			Foreground(colorGold).
			Italic(true)

	StyleHelp = lipgloss.NewStyle().
			Foreground(colorDim).
			MarginTop(1)

	StyleError = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(colorMint).
			Bold(true)

	StyleLabel = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	StyleInput = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Padding(0, 1)

	StyleInputFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	StylePill = lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 1, 0, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDim).
			Foreground(colorMuted)

	StylePillActive = lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 1, 0, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Foreground(colorAccent).
			Bold(true)

	StylePillSelected = lipgloss.NewStyle().
				Padding(0, 2).
				Margin(0, 1, 0, 0).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Foreground(colorFg).
				Background(colorHighlight).
				Bold(true)

	StyleFilter = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	StyleAddNew = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)

// UnderlineWithColor wraps an already-styled string in an ANSI underline of
// the given hex color. Inner [0m SGR resets are re-terminated with the
// underline sequence so the underline persists across nested segments —
// something lipgloss can't do for us without clobbering the inner colors.
//
// We emit underline toggle (4) and underline color (58:2::R:G:B) as *two
// separate* SGR escapes. Combining them in one escape (e.g.
// `\x1b[4;58:2::R:G:Bm`) mixes semicolon and colon separators in a way that
// several terminals (notably some iTerm2 builds) tokenize incorrectly,
// causing leftover numeric tokens to be re-interpreted as foreground color
// codes and blowing out nested segment colors.
func UnderlineWithColor(s, hex string) string {
	r, g, b := parseHex(hex)
	open := fmt.Sprintf("\x1b[4m\x1b[58:2::%d:%d:%dm", r, g, b)
	close := "\x1b[59m\x1b[24m" // reset underline color, then turn underline off
	// After every reset inside the string, re-open the underline so the
	// following text keeps the underline even if the inner style reset it.
	replaced := strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+open)
	return open + replaced + close
}

// Rainbow stops. Two palettes — the lighter one is used when lipgloss
// detects a dark terminal background so the gradient pops against it.
var rainbowStopsLight = []string{"#8B5CF6", "#0EA5E9", "#10B981", "#F59E0B", "#EC4899"}
var rainbowStopsDark = []string{"#C4B5FD", "#7DD3FC", "#6EE7B7", "#FCD34D", "#F9A8D4"}

// RainbowLine returns a smooth gradient horizontal line of the given width.
// Colors interpolate in RGB between each adjacent pair of stops (one full
// sweep across the width, not a repeating cycle), rendered with U+2501
// (HEAVY HORIZONTAL) for a bit more visual weight than U+2500.
func RainbowLine(width int) string {
	if width <= 0 {
		return ""
	}
	stops := rainbowStopsLight
	if lipgloss.HasDarkBackground() {
		stops = rainbowStopsDark
	}
	var b strings.Builder
	for i := 0; i < width; i++ {
		// t in [0,1] across the full width.
		var t float64
		if width == 1 {
			t = 0
		} else {
			t = float64(i) / float64(width-1)
		}
		hex := gradientAt(stops, t)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("━"))
	}
	return b.String()
}

// gradientAt returns the interpolated hex color at position t∈[0,1] across
// the given ordered list of hex stops.
func gradientAt(stops []string, t float64) string {
	if len(stops) == 0 {
		return "#ffffff"
	}
	if len(stops) == 1 {
		return stops[0]
	}
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	// Find the segment [stops[i], stops[i+1]] that t falls into.
	segments := len(stops) - 1
	pos := t * float64(segments)
	i := int(pos)
	if i >= segments {
		i = segments - 1
	}
	local := pos - float64(i)
	return lerpHex(stops[i], stops[i+1], local)
}

// lerpHex linearly interpolates between two "#rrggbb" strings in RGB space.
func lerpHex(a, b string, t float64) string {
	ar, ag, ab := parseHex(a)
	br, bg, bb := parseHex(b)
	r := int(float64(ar) + (float64(br)-float64(ar))*t + 0.5)
	g := int(float64(ag) + (float64(bg)-float64(ag))*t + 0.5)
	bl := int(float64(ab) + (float64(bb)-float64(ab))*t + 0.5)
	return fmt.Sprintf("#%02x%02x%02x", clamp8(r), clamp8(g), clamp8(bl))
}

func parseHex(s string) (int, int, int) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 255, 255, 255
	}
	var r, g, b int
	fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func clamp8(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// Header renders a title line with the sparkle badge on the right and a
// rainbow underline beneath. titleStyle controls the title color (teal for
// list/add, pink for remove).
func Header(title string, titleStyle lipgloss.Style, width int) string {
	badge := StyleSubtitle.Render(Sparkle + " worktree " + Sparkle)
	left := titleStyle.Render(title)
	// Compose the top line: title on the left, badge on the right.
	gap := width - lipgloss.Width(left) - lipgloss.Width(badge)
	if gap < 2 {
		gap = 2
	}
	top := left + strings.Repeat(" ", gap) + badge
	return top + "\n" + RainbowLine(width)
}

// Sparkle is a small flourish used in headers and success messages.
const Sparkle = "✦"

// Unicorn emoji used on the add-new row and success screens.
const Unicorn = "🦄"
