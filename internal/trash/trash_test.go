package trash

import (
	"strings"
	"testing"
)

func TestGenerateUniqueName(t *testing.T) {
	base := "my-feature"
	got := generateUniqueName(base)

	if !strings.HasPrefix(got, "my-feature_") {
		t.Errorf("expected prefix 'my-feature_', got %q", got)
	}

	// Should contain a Unix timestamp
	parts := strings.Split(got, "_")
	if len(parts) != 2 {
		t.Errorf("expected format 'name_timestamp', got %q", got)
	}
}

func TestGenerateUniqueNameWithExtension(t *testing.T) {
	base := "file.txt"
	got := generateUniqueName(base)

	if !strings.HasSuffix(got, ".txt") {
		t.Errorf("expected .txt extension preserved, got %q", got)
	}

	if !strings.Contains(got, "_") {
		t.Errorf("expected timestamp separator, got %q", got)
	}

	// Name part should be before timestamp
	if !strings.HasPrefix(got, "file_") {
		t.Errorf("expected prefix 'file_', got %q", got)
	}
}

func TestGenerateUniqueNameNoExtension(t *testing.T) {
	base := "worktree-branch"
	got := generateUniqueName(base)

	if !strings.HasPrefix(got, "worktree-branch_") {
		t.Errorf("expected prefix 'worktree-branch_', got %q", got)
	}

	// No extension, should end with timestamp
	parts := strings.Split(got, "_")
	if len(parts) != 2 { // worktree-branch_<timestamp>
		t.Errorf("expected 2 parts separated by '_', got %q with %d parts", got, len(parts))
	}
}
