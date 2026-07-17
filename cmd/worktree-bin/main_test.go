package main

import "testing"

func TestIsBareName(t *testing.T) {
	tests := map[string]bool{
		"feature-x":     true,
		"my_branch":     true,
		"release":       true,
		"../feature-x":  false, // relative path
		"./thing":       false, // relative path
		".hidden":       false, // leading dot
		"nested/branch": false, // contains separator
		"/abs/path":     false, // absolute path
		"-b":            false, // flag
		"--force":       false, // flag
		"":              false, // empty
	}
	for in, want := range tests {
		if got := isBareName(in); got != want {
			t.Errorf("isBareName(%q) = %v, want %v", in, got, want)
		}
	}
}
