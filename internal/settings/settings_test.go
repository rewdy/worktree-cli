package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveVariables(t *testing.T) {
	got := Resolve("../{project-name}-worktrees/", "myrepo", "feat-x")
	want := "../myrepo-worktrees/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveBranchVariable(t *testing.T) {
	got := Resolve("../wt/{branch}", "myrepo", "feat-x")
	want := "../wt/feat-x/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveEmptyBranch(t *testing.T) {
	// Detached HEAD → empty branch → variable expands to "".
	got := Resolve("../{branch}", "myrepo", "")
	want := "../"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveEmptyTemplate(t *testing.T) {
	got := Resolve("", "myrepo", "feat")
	want := "../"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveAddsTrailingSlash(t *testing.T) {
	got := Resolve("../wt", "myrepo", "feat")
	want := "../wt/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")

	want := Settings{
		DefaultPathTemplate: "../{project-name}-worktrees/",
		CollapsePaths:       true,
	}
	if err := saveTo(path, want); err != nil {
		t.Fatalf("saveTo: %v", err)
	}
	got := loadFrom(path)
	if got != want {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	got := loadFrom(filepath.Join(dir, "does-not-exist.yaml"))
	if got != Defaults() {
		t.Errorf("missing file should yield defaults, got %+v", got)
	}
}

func TestLoadMalformedFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.yaml")
	writeFile(t, path, "default_path_template: [oops")
	got := loadFrom(path)
	if got != Defaults() {
		t.Errorf("malformed file should fall back to defaults, got %+v", got)
	}
}

func TestFastRemoveEnabled(t *testing.T) {
	cases := []struct {
		name string
		s    Settings
		want bool
	}{
		{"both off", Settings{}, false},
		{"new key on", Settings{FastRemove: true}, true},
		{"legacy on", Settings{RemoveToTrash: true}, true},
		{"both on", Settings{FastRemove: true, RemoveToTrash: true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.FastRemoveEnabled(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsLegacyConfig(t *testing.T) {
	cases := []struct {
		name string
		s    Settings
		want bool
	}{
		{"neither", Settings{}, false},
		{"new key only", Settings{FastRemove: true}, false},
		{"legacy only", Settings{RemoveToTrash: true}, true},
		{"both set", Settings{FastRemove: true, RemoveToTrash: true}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.IsLegacyConfig(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMigrateLegacy(t *testing.T) {
	in := Settings{
		DefaultPathTemplate: "../wt/",
		CollapsePaths:       true,
		RemoveToTrash:       true,
	}
	got := in.MigrateLegacy()
	if !got.FastRemove {
		t.Errorf("FastRemove should be true after migrate")
	}
	if got.RemoveToTrash {
		t.Errorf("RemoveToTrash should be cleared after migrate")
	}
	// Other fields preserved.
	if got.DefaultPathTemplate != in.DefaultPathTemplate || got.CollapsePaths != in.CollapsePaths {
		t.Errorf("unrelated fields changed: %+v", got)
	}
}

func TestLoadPartialFileFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")
	writeFile(t, path, "collapse_paths: true\n")
	got := loadFrom(path)
	if got.DefaultPathTemplate != Defaults().DefaultPathTemplate {
		t.Errorf("missing template should use default, got %q", got.DefaultPathTemplate)
	}
	if !got.CollapsePaths {
		t.Errorf("collapse_paths should be true")
	}
}
