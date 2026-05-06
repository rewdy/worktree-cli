//go:build windows

package trash

func move(path string) (string, error) {
	return "", ErrUnsupported
}
