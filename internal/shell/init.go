package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Script returns the shell function source for the given shell ("bash", "zsh",
// "fish"). If shell is empty, autodetects from $SHELL.
func Script(shell string) (string, error) {
	if shell == "" {
		shell = detect()
	}
	switch shell {
	case "bash", "zsh", "":
		return posixScript(), nil
	case "fish":
		return fishScript(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shell)
	}
}

func detect() string {
	base := filepath.Base(os.Getenv("SHELL"))
	switch base {
	case "bash", "zsh", "fish":
		return base
	}
	return "bash"
}

func posixScript() string {
	// The shell function wraps the binary and cds into whatever path the
	// binary emits on fd 3.
	//
	// A straight $(...) command substitution would redirect stdout to a
	// pipe, which confuses terminal rendering — some terminals degrade
	// color fidelity when the foreground process doesn't own the TTY's
	// stdout. Instead we redirect so fd 3 captures the path, stdout goes
	// to stderr (and thus the TTY), and the TUI sees both stdout and
	// stderr connected to the real terminal and renders with full color.
	//
	// WORKTREE_WRAPPED=1 tells the binary it's being called through the
	// wrapper (suppresses the first-run tip). `wt_status` avoids zsh's
	// read-only `status` builtin.
	return strings.TrimSpace(`
# worktree-cli shell integration
worktree() {
  local target
  target=$(WORKTREE_WRAPPED=1 WORKTREE_PATH_FD=3 command worktree-bin "$@" 3>&1 1>&2)
  local wt_status=$?
  if [ $wt_status -ne 0 ]; then
    return $wt_status
  fi
  if [ -n "$target" ] && [ -d "$target" ]; then
    cd "$target" || return $?
  fi
}
`) + "\n"
}

func fishScript() string {
	// Uses fd 3 to pass the selected path — same rationale as the posix
	// script: keeps stdout/stderr connected to the real TTY so the TUI can
	// render with full color.
	return strings.TrimSpace(`
# worktree-cli shell integration
function worktree
  set -l target (WORKTREE_WRAPPED=1 WORKTREE_PATH_FD=3 command worktree-bin $argv 3>&1 1>&2)
  set -l wt_status $status
  if test $wt_status -ne 0
    return $wt_status
  end
  if test -n "$target" -a -d "$target"
    cd $target
  end
end
`) + "\n"
}

// TipDismissedPath returns the marker file path used to suppress the first-run
// install tip. Follows XDG conventions with a fallback to ~/.config.
func TipDismissedPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "worktree-cli", "tip-dismissed")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "worktree-cli", "tip-dismissed")
}

// IsWrapped returns true when the binary is being invoked through the shell
// function (which sets WORKTREE_WRAPPED=1).
func IsWrapped() bool {
	return os.Getenv("WORKTREE_WRAPPED") == "1"
}

// TipDismissed returns true if the user has dismissed the install tip.
func TipDismissed() bool {
	p := TipDismissedPath()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// DismissTip writes the marker file to stop showing the install tip.
func DismissTip() error {
	p := TipDismissedPath()
	if p == "" {
		return fmt.Errorf("could not determine config directory")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte("dismissed\n"), 0o644)
}
