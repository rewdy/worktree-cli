//go:build !windows

package trash

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
	"time"
)

// move relocates path to <graveyard>/<basename>, where the graveyard is:
//  1. $GRAVEYARD if set, else
//  2. $TMPDIR/graveyard-$USER on macOS (Apple cleans $TMPDIR aggressively),
//  3. /tmp/graveyard-$USER on Linux (tmpfs at /tmp wipes on reboot for
//     most distros).
//
// os.TempDir() resolves $TMPDIR with a /tmp fallback, which lines up with
// both macOS and Linux out of the box.
func move(path string) (string, error) {
	dir, err := graveyardDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create graveyard: %w", err)
	}

	basename := filepath.Base(path)
	target := filepath.Join(dir, basename)
	if _, err := os.Stat(target); err == nil {
		basename = generateUniqueName(basename)
		target = filepath.Join(dir, basename)
	}

	if err := os.Rename(path, target); err != nil {
		if errors.Is(err, syscall.EXDEV) {
			return "", ErrCrossDevice
		}
		return "", fmt.Errorf("failed to move to graveyard: %w", err)
	}
	return target, nil
}

// generateUniqueName creates a unique name for a graveyard item by
// appending a Unix timestamp when conflicts occur.
func generateUniqueName(base string) string {
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	timestamp := time.Now().Unix()
	if ext != "" {
		return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
	}
	return fmt.Sprintf("%s_%d", name, timestamp)
}

func graveyardDir() (string, error) {
	if g := os.Getenv("GRAVEYARD"); g != "" {
		return g, nil
	}
	uname := currentUsername()
	return filepath.Join(os.TempDir(), "graveyard-"+uname), nil
}

// currentUsername falls back to "user" if the OS lookup fails — the
// graveyard suffix is for per-user isolation on shared /tmp, not auth.
func currentUsername() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "user"
}
