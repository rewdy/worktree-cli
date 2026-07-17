# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- The add form now previews the exact `git worktree add …` command it will
  run, in a muted line at the bottom of the dialog, so you can see what
  filling out the form will do before submitting.

### Changed
- `worktree add <name>` with a bare name (no slash, dot, or flags) now
  resolves the new worktree's location against your `default_path_template`
  setting instead of creating it in the current working directory. Path-like
  args (`../foo`, `/abs`, `./x`) and any multi-arg invocations still pass
  through to `git worktree add` verbatim.

## [0.6.0] — 2026-06-01

### Added
- `fast_remove` setting. When enabled, removed worktrees move to a
  per-user "graveyard" under the OS temp dir
  (`$TMPDIR/graveyard-$USER` on macOS, `/tmp/graveyard-$USER` on Linux)
  instead of being recursively deleted. The OS clears the graveyard on
  its own schedule.
- `$GRAVEYARD` environment variable to override the graveyard location
  (e.g. when the default is on a different filesystem than your
  worktrees).
- One-line migration nag in the bare list view for users still on the
  legacy `remove_to_trash` key. Pressing `m` rewrites the YAML in place.
- `worktree settings` entry in the README command table.

### Changed
- Replaced the "move to system Trash" behavior with the OS graveyard
  described above. Atomic `rename(2)` keeps removal fast; the system
  Trash icon no longer fills up.
- On a cross-filesystem graveyard (`EXDEV`), removal falls back to
  `git worktree remove` and prints a hint to set `$GRAVEYARD`.

### Deprecated
- `remove_to_trash` setting key. Still honored as an alias for
  `fast_remove`. To be removed in v2.0.
