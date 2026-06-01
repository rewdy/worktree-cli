// Package trash moves directories to a "graveyard" under the OS temp dir
// instead of recursively deleting them. The OS clears the graveyard on its
// own schedule (macOS aggressively cleans $TMPDIR; Linux tmpfs at /tmp is
// reboot-wiped on most distros).
//
// The strategy mirrors nivekuil/rip (rm-improved) where it doesn't conflict
// with platform reality: same-disk rename for speed, no cross-FS copy
// fallback, $GRAVEYARD env var honored as an override.
package trash

import "errors"

var (
	// ErrUnsupported is returned on platforms that don't support graveyard
	// moves (currently Windows). Caller should fall back to git remove.
	ErrUnsupported = errors.New("graveyard not supported on this platform")
	// ErrCrossDevice is returned when the graveyard is on a different
	// filesystem than the source path. Caller should fall back to git
	// remove and hint the user to set $GRAVEYARD.
	ErrCrossDevice = errors.New("graveyard on different filesystem")
)

// MoveToGraveyard relocates path to the graveyard directory.
// Returns the final graveyard location or an error.
func MoveToGraveyard(path string) (string, error) {
	return move(path)
}
