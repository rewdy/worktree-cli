package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rewdy/worktree-cli/internal/git"
	"github.com/rewdy/worktree-cli/internal/shell"
	"github.com/rewdy/worktree-cli/internal/tui"
)

func main() {
	root := &cobra.Command{
		Use:   "worktree",
		Short: "A dreamy little TUI for git worktrees 🦄",
		Long: "worktree — a TUI for listing, creating, and removing git worktrees.\n" +
			"Run with no args to browse and cd into a worktree.",
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

	root.AddCommand(addCmd, removeCmd, homeCmd, shellInitCmd)

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

	for {
		model := tui.NewListModel(worktrees, current, tui.ModeSelect)
		result, err := tui.RunList(model)
		if err != nil {
			return err
		}
		if result.Cancelled {
			return nil
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
	model := tui.NewAddModel(defaultBranch, currentBranch)
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
	// Interactive picker.
	worktrees, err := git.List()
	if err != nil {
		return err
	}
	current, _ := git.CurrentWorktreePath()
	model := tui.NewListModel(worktrees, current, tui.ModeRemove)
	result, err := tui.RunList(model)
	if err != nil {
		return err
	}
	if result.Cancelled || result.Selected == "" {
		return nil
	}
	var out string
	spinErr := tui.RunWithSpinner("removing "+result.Selected+"…", func() {
		out, err = git.Remove(result.Selected)
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
	fmt.Fprintln(os.Stderr, tui.StyleSuccess.Render("✦ removed "+result.Selected))
	return nil
}

// --- shared path emission ----------------------------------------------

// emitPath writes the chosen worktree path to stdout (for the shell wrapper
// to consume) and, if not wrapped, shows a helpful hint on stderr.
func emitPath(path string) {
	fmt.Println(path)
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
