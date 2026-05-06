//go:build linux

package trash

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func move(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}

	// Follow XDG spec: check XDG_DATA_HOME first, fall back to ~/.local/share
	trashDir := filepath.Join(homeDir, ".local", "share", "Trash")
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		trashDir = filepath.Join(xdg, "Trash")
	}

	filesDir := filepath.Join(trashDir, "files")
	infoDir := filepath.Join(trashDir, "info")

	// Create trash directories if they don't exist (Freedesktop spec requires 0700)
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create trash files directory: %w", err)
	}
	if err := os.MkdirAll(infoDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create trash info directory: %w", err)
	}

	// Get absolute path for .trashinfo metadata
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	basename := filepath.Base(path)
	targetFile := filepath.Join(filesDir, basename)
	infoFile := filepath.Join(infoDir, basename+".trashinfo")

	// Handle name conflicts
	if _, err := os.Stat(targetFile); err == nil {
		basename = generateUniqueName(basename)
		targetFile = filepath.Join(filesDir, basename)
		infoFile = filepath.Join(infoDir, basename+".trashinfo")
	}

	// Create .trashinfo file first (Freedesktop spec)
	trashInfo := createTrashInfo(absPath)
	if err := os.WriteFile(infoFile, []byte(trashInfo), 0o600); err != nil {
		return "", fmt.Errorf("failed to create .trashinfo file: %w", err)
	}

	// Move file to trash
	if err := os.Rename(path, targetFile); err != nil {
		// Clean up .trashinfo on failure
		os.Remove(infoFile)
		return "", fmt.Errorf("failed to move to trash: %w", err)
	}

	return targetFile, nil
}

// createTrashInfo generates a .trashinfo file per the Freedesktop spec.
// Format:
//
//	[Trash Info]
//	Path=/absolute/path/to/original
//	DeletionDate=2024-05-06T15:30:00
func createTrashInfo(originalPath string) string {
	// URL-encode the path (spaces → %20, etc.)
	encoded := urlEncodePath(originalPath)

	// ISO 8601 format
	deletionDate := time.Now().Format("2006-01-02T15:04:05")

	return fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", encoded, deletionDate)
}

// urlEncodePath performs minimal URL encoding for .trashinfo Path field.
// Encodes space, tab, newline, and percent per Freedesktop trash spec.
func urlEncodePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		switch r {
		case ' ':
			b.WriteString("%20")
		case '\t':
			b.WriteString("%09")
		case '\n':
			b.WriteString("%0A")
		case '%':
			b.WriteString("%25")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
