//go:build darwin

package trash

import (
	"fmt"
	"os"
	"path/filepath"
)

func move(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}

	trashDir := filepath.Join(homeDir, ".Trash")

	// Verify trash directory exists
	if stat, err := os.Stat(trashDir); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("trash directory not accessible: %s", trashDir)
	}

	basename := filepath.Base(path)
	target := filepath.Join(trashDir, basename)

	// Handle name conflicts by appending timestamp
	if _, err := os.Stat(target); err == nil {
		basename = generateUniqueName(basename)
		target = filepath.Join(trashDir, basename)
	}

	// Move to trash using os.Rename (atomic within same filesystem)
	if err := os.Rename(path, target); err != nil {
		return "", fmt.Errorf("failed to move to trash: %w", err)
	}

	return target, nil
}
