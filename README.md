<p align="center">
  <img src=".assets/icon-sm.png" alt="worktree" width="160">
</p>

# worktree 🦄

A dreamy little TUI for managing git worktrees. Browse, create, and remove
worktrees with arrow keys, fuzzy filter, and a sprinkle of rainbow garnish.

## Install

Requires Go 1.22+.

```sh
go install github.com/rewdy/worktree-cli/cmd/worktree-bin@latest
```

This drops a binary called `worktree-bin` into `$(go env GOBIN)` (usually
`~/go/bin`). Make sure that directory is in your `$PATH`.

To use the ergonomic name `worktree` (and get `cd`-into-worktree behavior),
install the shell wrapper — see below.

Alternatively, build from source:

```sh
git clone https://github.com/rewdy/worktree-cli
cd worktree-cli
go build -o worktree-bin ./cmd/worktree-bin
mv worktree-bin ~/.local/bin/   # or anywhere on your $PATH
```

## Shell wrapper (for `cd`-into-worktree)

A child process can't change its parent shell's directory. The wrapper solves
this: the binary prints the chosen worktree path on stdout, and the wrapper
`cd`s there.

Add this to your `~/.zshrc` or `~/.bashrc`:

```sh
eval "$(worktree-bin shell-init)"
```

For fish (`~/.config/fish/config.fish`):

```fish
worktree-bin shell-init fish | source
```

After that, `worktree` is a shell function that invokes the binary and `cd`s
into whatever you select.

If you haven't installed the wrapper, `worktree` still works — it just prints
a helpful `cd <path>` hint instead of changing your directory. Dismiss that
tip with:

```sh
worktree-bin shell-init --dismiss-tip
```

## Usage

| Command | Behavior |
|---|---|
| `worktree` | Pick a worktree from a list, or `＋ Add new worktree`. Enter to select. `/` or just start typing to fuzzy-filter. |
| `worktree add` | Open the add-worktree form (path, branch, base picker). |
| `worktree add <path> [args…]` | Pure passthrough to `git worktree add <path> [args…]`. |
| `worktree remove` | Pick a worktree to remove. The one you're currently in is excluded. |
| `worktree remove <path>` | Passthrough to `git worktree remove <path>`. |
| `worktree home` | Jump to the main worktree (the original clone). |
| `worktree shell-init [bash\|zsh\|fish]` | Print the shell wrapper function. |

### List features

- **● indicator** marks the worktree you're currently in
- Branch name shown in `(parens)` when it differs from the folder name
- Detached HEADs show `(detached: <short-sha>)`
- Fuzzy filter — type anywhere to narrow, `esc` to clear
- `q` or `esc` to quit without selecting

### Add form

Three fields, `tab` to navigate:

1. **Path** — pre-populated with `../` since new worktrees usually go next to
   the current one
2. **Branch** — new branch name. Leave blank to use the folder name from the
   path (e.g. `../my-feature` → branch `my-feature`).
3. **Base** — segmented picker:
   - **main** (or **master** — whichever your repo has)
   - **&lt;current-branch&gt;** — only shown if it's not the same as main
   - **Other…** — free-form text input for any committish

### Safety

`worktree remove` never passes `--force`. If the worktree has uncommitted
changes, git's error is surfaced verbatim so you can decide what to do.

## Keys

### List
- `↑`/`↓` or `k`/`j` — move
- `enter` — select
- `/` or any letter — start fuzzy filter
- `esc` — clear filter (or quit if none active)
- `q` — quit

### Add form
- `tab` / `shift+tab` — next / previous field
- `enter` — advance to next field; on the last field, submit
- `←` / `→` — pick base option
- `esc` — cancel

## Development

```sh
go test ./...
go build -o worktree-bin ./cmd/worktree-bin
```
