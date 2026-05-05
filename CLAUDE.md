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
- **`Add` always creates a new branch** (`-b <branch>`). The `Base` field is treated as a *starting point* (committish), never as the checked-out branch. Branch name defaults to the sanitized path basename when blank.
- **Passthrough modes**: `worktree add <path> [args…]` and `worktree remove <path>` skip the TUI entirely and forward args to `git worktree`. Cobra's `DisableFlagParsing` is set on these subcommands so user flags aren't consumed.
- **zsh gotcha**: the wrapper uses a local `wt_status` variable instead of `$status` because zsh reserves `status` as a read-only builtin.
