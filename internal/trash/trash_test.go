//go:build !windows

package trash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateUniqueName(t *testing.T) {
	got := generateUniqueName("my-feature")
	if !strings.HasPrefix(got, "my-feature_") {
		t.Errorf("expected prefix 'my-feature_', got %q", got)
	}
	if parts := strings.Split(got, "_"); len(parts) != 2 {
		t.Errorf("expected format 'name_timestamp', got %q", got)
	}
}

func TestGenerateUniqueNameWithExtension(t *testing.T) {
	got := generateUniqueName("file.txt")
	if !strings.HasSuffix(got, ".txt") {
		t.Errorf("expected .txt extension preserved, got %q", got)
	}
	if !strings.HasPrefix(got, "file_") {
		t.Errorf("expected prefix 'file_', got %q", got)
	}
}

func TestGenerateUniqueNameNoExtension(t *testing.T) {
	got := generateUniqueName("worktree-branch")
	if !strings.HasPrefix(got, "worktree-branch_") {
		t.Errorf("expected prefix 'worktree-branch_', got %q", got)
	}
	if parts := strings.Split(got, "_"); len(parts) != 2 {
		t.Errorf("expected 2 parts, got %d in %q", len(parts), got)
	}
}

func TestMoveToGraveyardHappyPath(t *testing.T) {
	// Point $GRAVEYARD at a tempdir so the test doesn't pollute the real
	// graveyard and so rename stays on the same filesystem.
	graveyard := t.TempDir()
	t.Setenv("GRAVEYARD", graveyard)

	src := filepath.Join(t.TempDir(), "myworktree")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	dest, err := MoveToGraveyard(src)
	if err != nil {
		t.Fatalf("MoveToGraveyard: %v", err)
	}
	if !strings.HasPrefix(dest, graveyard) {
		t.Errorf("dest %q should be under %q", dest, graveyard)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be gone, stat err = %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("dest should exist, got %v", err)
	}
}

func TestMoveToGraveyardConflictAppendsTimestamp(t *testing.T) {
	graveyard := t.TempDir()
	t.Setenv("GRAVEYARD", graveyard)

	// Pre-create a clashing entry in the graveyard.
	if err := os.Mkdir(filepath.Join(graveyard, "wt"), 0o755); err != nil {
		t.Fatalf("seed graveyard: %v", err)
	}

	src := filepath.Join(t.TempDir(), "wt")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	dest, err := MoveToGraveyard(src)
	if err != nil {
		t.Fatalf("MoveToGraveyard: %v", err)
	}
	if filepath.Base(dest) == "wt" {
		t.Errorf("expected timestamp suffix, got %q", filepath.Base(dest))
	}
	if !strings.HasPrefix(filepath.Base(dest), "wt_") {
		t.Errorf("expected 'wt_<ts>' name, got %q", filepath.Base(dest))
	}
}

func TestGraveyardDirHonorsEnvOverride(t *testing.T) {
	t.Setenv("GRAVEYARD", "/some/explicit/path")
	got, err := graveyardDir()
	if err != nil {
		t.Fatalf("graveyardDir: %v", err)
	}
	if got != "/some/explicit/path" {
		t.Errorf("got %q, want explicit override", got)
	}
}

func TestGraveyardDirDefaultIncludesUser(t *testing.T) {
	t.Setenv("GRAVEYARD", "")
	got, err := graveyardDir()
	if err != nil {
		t.Fatalf("graveyardDir: %v", err)
	}
	if !strings.Contains(filepath.Base(got), "graveyard-") {
		t.Errorf("expected 'graveyard-<user>' basename, got %q", got)
	}
}
