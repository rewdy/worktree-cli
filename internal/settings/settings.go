package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Settings holds user-configurable preferences persisted to
// ~/.config/worktree-cli/settings.yaml.
type Settings struct {
	// DefaultPathTemplate is used to pre-populate the Path field in the
	// add form. Supports {project-name} and {branch} variables. Resolved
	// value always ends with "/" so users can append a folder name.
	DefaultPathTemplate string `yaml:"default_path_template"`
	// CollapsePaths, when true, replaces the longest common prefix across
	// the visible worktrees with "…/" in the list view. Display-only —
	// actual paths are unchanged.
	CollapsePaths bool `yaml:"collapse_paths"`
	// FastRemove, when true, moves removed worktrees to a graveyard under
	// the OS temp dir instead of recursively deleting. The OS clears the
	// graveyard on its own schedule. macOS and Linux only.
	FastRemove bool `yaml:"fast_remove"`
	// RemoveToTrash is the legacy alias for FastRemove. Treated as
	// FastRemove=true when set. The bare list view shows a one-line nag
	// offering to migrate.
	//
	// Deprecated: use FastRemove. TODO: remove in v2.0.
	RemoveToTrash bool `yaml:"remove_to_trash"`
}

// Defaults returns the out-of-the-box settings, used when no settings
// file exists or parsing fails.
func Defaults() Settings {
	return Settings{
		DefaultPathTemplate: "../",
		CollapsePaths:       false,
		FastRemove:          false,
		RemoveToTrash:       false,
	}
}

// FastRemoveEnabled returns true when fast-remove behavior should apply,
// honoring both the new key and the legacy alias.
func (s Settings) FastRemoveEnabled() bool {
	return s.FastRemove || s.RemoveToTrash
}

// IsLegacyConfig returns true when the user is on the deprecated
// remove_to_trash key without the new fast_remove key set. Used to
// trigger the migration nag.
func (s Settings) IsLegacyConfig() bool {
	return s.RemoveToTrash && !s.FastRemove
}

// MigrateLegacy returns a copy with the legacy alias cleared and the
// new key set, so the saved YAML no longer contains remove_to_trash.
func (s Settings) MigrateLegacy() Settings {
	out := s
	out.FastRemove = true
	out.RemoveToTrash = false
	return out
}

// Path returns the absolute path to the settings file, honoring
// $XDG_CONFIG_HOME with a ~/.config fallback. Returns "" if it can't
// resolve a home directory.
func Path() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "worktree-cli", "settings.yaml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "worktree-cli", "settings.yaml")
}

// Load reads the settings file. Missing file → defaults silently.
// Malformed file → warn on stderr and return defaults. Never returns
// an error; settings are best-effort and shouldn't block the TUI.
func Load() Settings {
	return loadFrom(Path())
}

// loadFrom is the testable inner: same behavior as Load but takes a path.
func loadFrom(path string) Settings {
	defaults := Defaults()
	if path == "" {
		return defaults
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaults
	}
	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not parse %s (%v); using defaults\n", path, err)
		return defaults
	}
	// Fill missing fields with defaults so partial files don't break.
	if s.DefaultPathTemplate == "" {
		s.DefaultPathTemplate = defaults.DefaultPathTemplate
	}
	return s
}

// Save writes the settings to disk, creating the config directory if
// needed.
func Save(s Settings) error {
	return saveTo(Path(), s)
}

func saveTo(path string, s Settings) error {
	if path == "" {
		return fmt.Errorf("could not determine settings path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Resolve expands {project-name} and {branch} in the template and
// guarantees a trailing "/". Empty template falls back to "../".
func Resolve(tmpl, projectName, branch string) string {
	if strings.TrimSpace(tmpl) == "" {
		tmpl = "../"
	}
	out := strings.ReplaceAll(tmpl, "{project-name}", projectName)
	out = strings.ReplaceAll(out, "{branch}", branch)
	if !strings.HasSuffix(out, "/") {
		out += "/"
	}
	return out
}
