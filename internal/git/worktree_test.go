package git

import "testing"

func TestParsePorcelain(t *testing.T) {
	input := `worktree /tmp/repo
HEAD abc1234567890abcdef
branch refs/heads/main

worktree /tmp/feat
HEAD abc1234567890abcdef
branch refs/heads/feature-x

worktree /tmp/detached
HEAD def4567890abcdef1234
detached

`
	got, err := parsePorcelain(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(got))
	}
	if got[0].Path != "/tmp/repo" || got[0].Branch != "main" {
		t.Errorf("first entry wrong: %+v", got[0])
	}
	if got[1].Branch != "feature-x" {
		t.Errorf("second branch wrong: %+v", got[1])
	}
	if !got[2].Detached || got[2].Branch != "" {
		t.Errorf("third entry should be detached: %+v", got[2])
	}
}

func TestBranchDiffersFromFolder(t *testing.T) {
	tests := []struct {
		name string
		w    Worktree
		want bool
	}{
		{"same", Worktree{Path: "/x/feature-x", Branch: "feature-x"}, false},
		{"different", Worktree{Path: "/x/hotfix", Branch: "release/1"}, true},
		{"detached", Worktree{Path: "/x/exp", Detached: true}, false},
		{"empty branch", Worktree{Path: "/x/bare"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.w.BranchDiffersFromFolder(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := map[string]string{
		"feature-x":        "feature-x",
		"my feature":       "my-feature",
		"-bad":             "bad",
		".hidden":          "hidden",
		"weird:name*stuff": "weird-name-stuff",
		"":                 "wt",
	}
	for in, want := range tests {
		if got := sanitizeBranchName(in); got != want {
			t.Errorf("sanitizeBranchName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDisplayBranchDetached(t *testing.T) {
	w := Worktree{Detached: true, HEAD: "abcdef1234567890"}
	if got := w.DisplayBranch(); got != "detached: abcdef1" {
		t.Errorf("got %q", got)
	}
}
