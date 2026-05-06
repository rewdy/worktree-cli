package trash

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

var (
	ErrUnsupported = errors.New("trash functionality not supported on this platform")
	ErrConflict    = errors.New("item already exists in trash")
)

// Move relocates the given path to the system trash/recycle bin.
// If a name conflict exists, appends a Unix timestamp to make it unique.
// Returns the final trash location or an error.
func Move(path string) (string, error) {
	return move(path)
}

// generateUniqueName creates a unique name for a trash item by appending
// a Unix timestamp when conflicts occur.
func generateUniqueName(base string) string {
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]
	timestamp := time.Now().Unix()
	if ext != "" {
		return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
	}
	return fmt.Sprintf("%s_%d", name, timestamp)
}
