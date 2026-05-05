package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/rewdy/worktree-cli/internal/git"
	"github.com/rewdy/worktree-cli/internal/settings"
	"github.com/rewdy/worktree-cli/internal/shell"
	"github.com/rewdy/worktree-cli/internal/tui"
)

// version is stamped at build time via `-ldflags "-X main.version=..."`.
// When unset (typical for `go install` or `go build`), it's populated from
// the embedded module info so `worktree --version` still reports something
// useful.
var version = ""

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	root := &cobra.Command{
		Use:   "worktree",
		Short: "A dreamy little TUI for git worktrees 🦄",
		Long: "worktree — a TUI for listing, creating, and removing git worktrees.\n" +
			"Run with no args to browse and cd into a worktree.",
		Version:       resolveVersion(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBare()
		},
	}

	addCmd := &cobra.Command{
		Use:                "add [path] [git worktree add args...]",
		Short:              "Create a new worktree (interactive form if no path given)",
		DisableFlagParsing: true, // we want raw passthrough when a path is given
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args)
		},
	}

	removeCmd := &cobra.Command{
		Use:                "remove [path]",
		Short:              "Remove a worktree (interactive picker if no path given)",
		Aliases:            []string{"rm"},
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args)
		},
	}

	homeCmd := &cobra.Command{
		Use:   "home",
		Short: "Jump to the main worktree (the original clone)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHome()
		},
	}

	shellInitCmd := &cobra.Command{
		Use:   "shell-init [bash|zsh|fish]",
		Short: "Print the shell wrapper function for eval",
		Long: "Emit a shell function that wraps the binary so `worktree` can change " +
			"your current directory. Install with:\n\n  eval \"$(worktree shell-init)\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			dismiss, _ := cmd.Flags().GetBool("dismiss-tip")
			if dismiss {
				if err := shell.DismissTip(); err != nil {
					return err
				}
				fmt.Fprintln(os.Stderr, "Got it — tip dismissed.")
				return nil
			}
			shellName := ""
			if len(args) > 0 {
				shellName = args[0]
			}
			script, err := shell.Script(shellName)
			if err != nil {
				return err
			}
			fmt.Print(script)
			return nil
		},
	}
	shellInitCmd.Flags().Bool("dismiss-tip", false, "stop showing the first-run install tip")

	settingsCmd := &cobra.Command{
		Use:   "settings",
		Short: "Edit worktree preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := openSettings(settings.Load())
			return err
		},
	}

	root.AddCommand(addCmd, removeCmd, homeCmd, shellInitCmd, settingsCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, tui.StyleError.Render("✗ "+err.Error()))
		os.Exit(1)
	}
}

// --- bare command -------------------------------------------------------

func runBare() error {
	if err := git.InsideRepo(); err != nil {
		return err
	}
	worktrees, err := git.List()
	if err != nil {
		return err
	}
	current, _ := git.CurrentWorktreePath()
	s := settings.Load()

	for {
		model := tui.NewListModel(worktrees, current, tui.ModeSelect, s.CollapsePaths)
		result, err := tui.RunList(model)
		if err != nil {
			return err
		}
		if result.Cancelled {
			return nil
		}
		if result.OpenSettings {
			if updated, err := openSettings(s); err != nil {
				return err
			} else if updated != nil {
				s = *updated
			}
			worktrees, _ = git.List()
			continue
		}
		if result.AddNew {
			// Show add form; on success, cd into the new worktree.
			path, err := runAddInteractive()
			if err != nil {
				return err
			}
			if path == "" {
				// User cancelled the add form — return to the list.
				// Refresh list in case anything changed.
				worktrees, _ = git.List()
				continue
			}
			emitPath(path)
			return nil
		}
		if result.Remove {
			// User pressed `x` on a row — confirm, remove, then return to
			// the list (without emitting a path, since we're not cd-ing).
			if err := confirmAndRemove(result.SelectedWorktree); err != nil {
				return err
			}
			worktrees, _ = git.List()
			continue
		}
		if result.Unlock {
			if err := unlockWorktree(result.SelectedWorktree); err != nil {
				return err
			}
			worktrees, _ = git.List()
			continue
		}
		if result.Selected != "" {
			emitPath(result.Selected)
			return nil
		}
		return nil
	}
}

// --- add command --------------------------------------------------------

func runAdd(args []string) error {
	if err := git.InsideRepo(); err != nil {
		return err
	}
	// No args → interactive form.
	if len(args) == 0 {
		path, err := runAddInteractive()
		if err != nil {
			return err
		}
		if path == "" {
			return nil
		}
		emitPath(path)
		return nil
	}
	// With args → pure passthrough to `git worktree add`.
	var (
		path string
		out  string
		err  error
	)
	spinErr := tui.RunWithSpinner("creating worktree…", func() {
		path, out, err = git.AddPassthrough(args)
	})
	if spinErr != nil {
		return spinErr
	}
	if out != "" {
		fmt.Fprintln(os.Stderr, out)
	}
	if err != nil {
		return err
	}
	if path != "" {
		fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ created "+path+" "+tui.Unicorn))
		emitPath(path)
	}
	return nil
}

// runAddInteractive runs the add form and performs the git operation.
// Returns the absolute path of the new worktree on success, "" if cancelled.
func runAddInteractive() (string, error) {
	defaultBranch := git.DefaultBranch()
	currentBranch := git.CurrentBranch()
	s := settings.Load()
	projectName := resolveProjectName()
	seededPath := settings.Resolve(s.DefaultPathTemplate, projectName, currentBranch)
	model := tui.NewAddModel(defaultBranch, currentBranch, seededPath)
	result, err := tui.RunAdd(model)
	if err != nil {
		return "", err
	}
	if result.Cancelled || !result.Submitted {
		return "", nil
	}
	var (
		path string
		out  string
	)
	spinErr := tui.RunWithSpinner("creating worktree…", func() {
		path, out, err = git.Add(result.Options)
	})
	if spinErr != nil {
		return "", spinErr
	}
	if out != "" {
		fmt.Fprintln(os.Stderr, out)
	}
	if err != nil {
		return "", err
	}
	fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ created "+path+" "+tui.Unicorn))
	return path, nil
}

// --- home command -------------------------------------------------------

func runHome() error {
	if err := git.InsideRepo(); err != nil {
		return err
	}
	path, err := git.MainWorktreePath()
	if err != nil {
		return err
	}
	emitPath(path)
	return nil
}

// --- remove command -----------------------------------------------------

func runRemove(args []string) error {
	if err := git.InsideRepo(); err != nil {
		return err
	}
	// Passthrough mode when a path is given.
	if len(args) > 0 {
		target := args[0]
		var (
			out string
			err error
		)
		spinErr := tui.RunWithSpinner("removing "+target+"…", func() {
			out, err = git.Remove(target)
		})
		if spinErr != nil {
			return spinErr
		}
		if out != "" {
			fmt.Fprintln(os.Stderr, out)
		}
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ removed "+target))
		return nil
	}
	// Interactive picker. Loops so `u` can unlock and return to the list.
	s := settings.Load()
	for {
		worktrees, err := git.List()
		if err != nil {
			return err
		}
		current, _ := git.CurrentWorktreePath()
		model := tui.NewListModel(worktrees, current, tui.ModeRemove, s.CollapsePaths)
		result, err := tui.RunList(model)
		if err != nil {
			return err
		}
		if result.Cancelled {
			return nil
		}
		if result.Unlock {
			if err := unlockWorktree(result.SelectedWorktree); err != nil {
				return err
			}
			continue
		}
		if result.Selected == "" {
			return nil
		}
		return confirmAndRemove(result.SelectedWorktree)
	}
}

// unlockWorktree runs `git worktree unlock` on the given worktree and
// surfaces git's output. No confirm dialog — unlock is benign.
func unlockWorktree(wt git.Worktree) error {
	var (
		out string
		err error
	)
	spinErr := tui.RunWithSpinner("unlocking "+wt.Path+"…", func() {
		out, err = git.Unlock(wt.Path)
	})
	if spinErr != nil {
		return spinErr
	}
	if out != "" {
		fmt.Fprintln(os.Stderr, out)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ unlocked "+wt.Path))
	return nil
}

// resolveProjectName returns the basename of the main worktree. Falls back
// to the current worktree's basename for bare repos where there's no main,
// and finally to "project" if nothing resolves cleanly.
func resolveProjectName() string {
	if main, err := git.MainWorktreePath(); err == nil {
		if name := filepath.Base(main); name != "" && name != "." && name != "/" {
			return name
		}
	}
	if cur, err := git.CurrentWorktreePath(); err == nil {
		if name := filepath.Base(cur); name != "" && name != "." && name != "/" {
			return name
		}
	}
	return "project"
}

// openSettings runs the settings modal with `current` pre-filled, persists
// the result on save, and returns the new settings (or nil if cancelled).
func openSettings(current settings.Settings) (*settings.Settings, error) {
	result, err := tui.RunSettings(tui.NewSettingsModel(current))
	if err != nil {
		return nil, err
	}
	if !result.Saved {
		return nil, nil
	}
	if err := settings.Save(result.Settings); err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ settings saved"))
	return &result.Settings, nil
}

// confirmAndRemove shows the confirm dialog for the given worktree and,
// if the user confirms, runs `git worktree remove`. Surfaces git's error
// verbatim on failure (e.g. uncommitted changes). A no-op if the user
// cancels.
func confirmAndRemove(wt git.Worktree) error {
	confirmResult, err := tui.RunConfirm(tui.NewConfirmModel(wt))
	if err != nil {
		return err
	}
	if !confirmResult.Confirmed {
		return nil
	}
	var out string
	spinErr := tui.RunWithSpinner("removing "+wt.Path+"…", func() {
		out, err = git.Remove(wt.Path)
	})
	if spinErr != nil {
		return spinErr
	}
	if out != "" {
		fmt.Fprintln(os.Stderr, out)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ removed "+wt.Path))
	return nil
}

// --- shared path emission ----------------------------------------------

// emitPath writes the chosen worktree path to wherever the shell wrapper
// wants it (file descriptor specified by $WORKTREE_PATH_FD, or stdout if
// unset), and shows a helpful hint on stderr if the binary is running
// outside the shell wrapper.
//
// Using a dedicated file descriptor (instead of stdout) lets the wrapper
// keep stdout connected to the real terminal, which is what bubbletea /
// lipgloss need to render full color without degradation.
func emitPath(path string) {
	writePathChannel(path + "\n")
	if !shell.IsWrapped() && !shell.TipDismissed() {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, tui.StyleSubtitle.Render("  See the worktree:"))
		fmt.Fprintln(os.Stderr, "    cd "+path)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, tui.StyleSubtitle.Render(
			"  💡 Tip: to cd automatically, add this to your shell config:"))
		fmt.Fprintln(os.Stderr, "       eval \"$(worktree shell-init)\"")
		fmt.Fprintln(os.Stderr, tui.StyleSubtitle.Render(
			"  (dismiss this tip with: worktree shell-init --dismiss-tip)"))
	}
}

// writePathChannel writes to the caller-designated fd (via $WORKTREE_PATH_FD)
// or stdout as a fallback. Silently falls back to stdout if the fd isn't open.
func writePathChannel(s string) {
	if fdStr := os.Getenv("WORKTREE_PATH_FD"); fdStr != "" {
		if fd, err := strconv.Atoi(fdStr); err == nil && fd > 0 {
			f := os.NewFile(uintptr(fd), "path-fd")
			if f != nil {
				if _, err := f.WriteString(s); err == nil {
					return
				}
			}
		}
	}
	fmt.Print(s)
}
