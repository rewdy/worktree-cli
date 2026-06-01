<p align="center">
  <img src=".assets/icon-sm.png" alt="worktree" width="160">
</p>

# worktree 🦄

A dreamy little TUI for managing git worktrees. Browse, create, and remove
worktrees with arrow keys, fuzzy filter, and a sprinkle of rainbow garnish.

## Demo

[![asciicast](https://asciinema.org/a/LpY6YrBFm0ba4pei.svg)](https://asciinema.org/a/LpY6YrBFm0ba4pei)

📹 [View on asciinema](https://asciinema.org/a/LpY6YrBFm0ba4pei)

## Install

Requires Go 1.22+.

```sh
# 1. Install the binary (goes into $(go env GOBIN), usually ~/go/bin — make sure it's on $PATH)
go install github.com/rewdy/worktree-cli/cmd/worktree-bin@latest

# 2. Install the shell wrapper so `worktree` cd's into the selected path.
#    Pick the line for your shell:
echo 'eval "$(worktree-bin shell-init)"' >> ~/.zshrc       # zsh
echo 'eval "$(worktree-bin shell-init)"' >> ~/.bashrc      # bash
echo 'worktree-bin shell-init fish | source' >> ~/.config/fish/config.fish   # fish

# 3. Reload your shell (or open a new terminal).
```

After that, run `worktree` from inside any git repo.

### Why the shell wrapper?

A child process can't change its parent shell's directory. The wrapper is a
small shell function: the binary prints the chosen worktree path on stdout,
and the function `cd`s there. Without it, `worktree` still works — it just
prints `cd <path>` for you to copy. Dismiss the install hint with:

```sh
worktree-bin shell-init --dismiss-tip
```

### Build from source

```sh
git clone https://github.com/rewdy/worktree-cli
cd worktree-cli
go build -o worktree-bin ./cmd/worktree-bin
mv worktree-bin ~/.local/bin/   # or anywhere on your $PATH
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
| `worktree settings` | Edit preferences (default path template, fast remove, etc.). |
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

### Settings

Run `worktree settings` to open the preferences modal. Settings are stored
at `$XDG_CONFIG_HOME/worktree-cli/settings.yaml` (or `~/.config/...` if XDG
isn't set).

| Key | Default | Behavior |
|---|---|---|
| `default_path_template` | `../` | Pre-fills the Path field in the Add form. Supports `{project-name}` and `{branch}` variables. |
| `collapse_paths` | `false` | In list view, elide the longest shared directory prefix as `…/`. Display only — actual paths are unchanged. |
| `fast_remove` | `false` | Move removed worktrees to a per-user "graveyard" under the OS temp dir instead of recursively deleting them. Much faster, and the OS clears the graveyard on its own schedule. |

#### `fast_remove` details

When enabled, the destination is:
- macOS: `$TMPDIR/graveyard-$USER` (Apple cleans `$TMPDIR` aggressively)
- Linux: `/tmp/graveyard-$USER` (tmpfs at `/tmp` is reboot-wiped on most distros)
- Override either with the `$GRAVEYARD` env var

The removal uses an atomic `rename(2)`, which only works when the source
and the graveyard are on the same filesystem. On a cross-filesystem
graveyard, `worktree` falls back to `git worktree remove` and prints a
hint to set `$GRAVEYARD` to a same-disk path. Windows always falls back
to standard `git worktree remove`.

The dirty check still runs first — uncommitted changes block fast removal
the same way they block `git worktree remove`.

> **Migrating from `remove_to_trash`.** Earlier versions used a
> `remove_to_trash` setting that moved worktrees to the system Trash. That
> key is still honored as an alias for `fast_remove`, but it's deprecated.
> Bare `worktree` shows a one-line nag offering migration; press `m` to
> rewrite the YAML in place.

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
