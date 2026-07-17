# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go test ./...                                  # run all tests
go test ./internal/tui -run TestListModel      # run a single test
go build -o worktree-bin ./cmd/worktree-bin    # build the binary
go vet ./...                                   # static checks
```

Go 1.24 (see `.tool-versions`). Module path is `github.com/rewdy/worktree-cli` — note that the on-disk directory name (`worktree-tool`) does not match the module path.

## Releases

Whenever you bump the version (any `git tag vX.Y.Z`), add a matching entry to `CHANGELOG.md` in the same commit. Follow the [Keep a Changelog](https://keepachangelog.com/) format already in use: a dated `## [X.Y.Z]` heading, then sections (`Added` / `Changed` / `Deprecated` / `Removed` / `Fixed` / `Security`) for whatever applies. Don't ship a tag without the corresponding changelog entry.

**Before pushing a feature (or any user-facing change) to `main`, remind me to cut a release.** Don't let feature work land untagged. The sequence is: move the `## [Unreleased]` notes into a dated `## [X.Y.Z]` entry, commit it, `git tag vX.Y.Z` that commit, then push both `main` and the tag (`git push origin main && git push origin vX.Y.Z`). If I'm about to push without a tag, flag it and propose the version bump (minor for new features/backward-compatible changes, patch for fixes).

## Architecture

### Two-binary pattern

The user-facing command `worktree` is a **shell function** that wraps the Go binary `worktree-bin`. This exists because a child process cannot change its parent shell's directory — the wrapper does the `cd` after the binary prints a path.

The wrapper source lives in `internal/shell/init.go` and is emitted by `worktree-bin shell-init`. Users install it with `eval "$(worktree-bin shell-init)"`.

### Path-emission protocol

The binary needs to return a chosen path to the wrapper **without** putting it on stdout, because the Bubble Tea TUI needs stdout connected to the real TTY for proper color rendering. The protocol:

- Wrapper opens fd 3 and sets `WORKTREE_PATH_FD=3`
- Binary writes the selected path to that fd (see `emitPath` / `writePathChannel` in `cmd/worktree-bin/main.go`)
- Stdout/stderr stay on the TTY so lipgloss/termenv render full color
- Wrapper also sets `WORKTREE_WRAPPED=1` so the binary knows not to print the first-run install tip

If the wrapper isn't installed, the binary falls back to stdout and prints a `cd <path>` hint on stderr. Dismissing the tip writes a marker file under `$XDG_CONFIG_HOME/worktree-cli/tip-dismissed`.

### TUI renders to stderr

`internal/tui/io.go` has an `init()` that re-profiles the default lipgloss renderer to `os.Stderr` and picks a termenv color profile from `$COLORTERM` / `$TERM` explicitly (rather than letting termenv probe a possibly-piped stdout). All Bubble Tea programs are constructed with `ttyOptions()` which routes output to stderr while keeping stdin as the real `*os.File` — wrapping stdin in a plain `io.Reader` breaks raw-mode input (arrow keys arrive as literal `^[[B`).

### Package layout

- `cmd/worktree-bin/main.go` — cobra CLI. Subcommands: bare (list+select), `add`, `remove` (alias `rm`), `home`, `shell-init`.
- `internal/git` — wraps `git worktree` plus helpers (`List`, `Add`, `Remove`, `MainWorktreePath`, `DefaultBranch`, `CurrentBranch`). `List` parses `--porcelain` output. `runCombined` sets `GIT_TERMINAL_PROMPT=0` and returns combined stdout+stderr so git errors surface verbatim.
- `internal/shell` — shell wrapper script generation (bash/zsh share a POSIX script; fish has its own) and the tip-dismissal marker.
- `internal/tui` — Bubble Tea models: `ListModel` (with `ModeSelect`/`ModeRemove`), `AddModel` (three-field form with segmented base picker), `ConfirmModel`. Styles in `styles.go`, spinner in `spinner.go`.

### Key behavioral invariants

- **`git worktree remove` never uses `--force`.** If the worktree is dirty, git's refusal is surfaced verbatim. Don't add a force path without explicit user request.
- **The remove picker excludes the current worktree** (`ModeRemove` in `NewListModel`). Removing-the-one-you're-in is a foot-gun we deliberately prevent.
- **`Add` works in dual mode**: If the branch exists, it checks it out; if the branch is new, it creates it with `-b` from the specified Base. Branch name defaults to the sanitized path basename when blank. The Base field is only used for new branches.
- **Passthrough modes**: `worktree add <path> [args…]` and `worktree remove <path>` skip the TUI entirely and forward args to `git worktree`. Cobra's `DisableFlagParsing` is set on these subcommands so user flags aren't consumed. **Exception — bare names**: `worktree add <name>` where `<name>` is a single arg with no path separator, no leading `.`, and no leading `-` (see `isBareName`) is treated as a worktree *name*, not a path: it's resolved against `default_path_template` (same as the interactive form's seed) so it lands in the configured location rather than cwd. Anything path-like or multi-arg still passes straight through.
- **Add-command preview**: The interactive add form renders the exact `git worktree add …` command it will run in a muted line at the bottom (`AddModel.commandPreview`). The command string comes from `git.AddArgs`, the single source of truth also used by `git.Add`, so the preview can never drift from what actually runs. `AddArgs` calls `BranchExists` (which shells out to `git show-ref`), so previews are memoized per `(path, branch, base)` in `AddModel.previewCache` to avoid a git call on every keystroke/blink.
- **Fast remove**: When `fast_remove: true` in settings, worktrees are moved to a per-user "graveyard" under the OS temp dir instead of being recursively deleted. The OS handles cleanup (macOS clears `$TMPDIR` aggressively; Linux tmpfs at `/tmp` wipes on reboot). Default location is `$TMPDIR/graveyard-$USER` on macOS and `/tmp/graveyard-$USER` on Linux; override with the `$GRAVEYARD` env var. Runs the dirty check before removal. On `EXDEV` (graveyard on a different filesystem), the package returns `trash.ErrCrossDevice` and main.go falls back to `git worktree remove` with a one-line hint to set `$GRAVEYARD`. Windows still falls back to standard git remove. Git metadata is cleaned up with `git worktree prune` after a successful move.
- **Legacy `remove_to_trash` alias**: The deprecated `remove_to_trash: true` key is still honored as a synonym for `fast_remove: true` (`Settings.FastRemoveEnabled`). The bare list view shows a one-line nag (`⚠ remove_to_trash is deprecated — press m to migrate`) when `Settings.IsLegacyConfig()` returns true; pressing `m` calls `Settings.MigrateLegacy()` and saves. Saving from the settings TUI also implicitly migrates by writing `remove_to_trash: false`. TODO: remove the alias in v2.0.
- **zsh gotcha**: the wrapper uses a local `wt_status` variable instead of `$status` because zsh reserves `status` as a read-only builtin.
