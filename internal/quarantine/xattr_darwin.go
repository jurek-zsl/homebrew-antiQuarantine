//go:build darwin

package quarantine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	// AttributeName is the extended attribute key used by macOS Gatekeeper
	AttributeName = "com.apple.quarantine"

	// XattrNoFollow prevents following symbolic links on Darwin
	XattrNoFollow = 0x0001
)

var (
	// ErrAttributeNotFound is returned when the extended attribute does not exist
	ErrAttributeNotFound = errors.New("attribute not found")
)

// HasQuarantine checks whether the target path contains the com.apple.quarantine attribute without following symlinks.
func HasQuarantine(path string) (bool, error) {
	sz, err := unix.Lgetxattr(path, AttributeName, nil)
	if err == nil && sz >= 0 {
		return true, nil
	}
	if isAttrNotFoundError(err) {
		return false, nil
	}
	if os.IsNotExist(err) {
		return false, os.ErrNotExist
	}
	return false, fmt.Errorf("lgetxattr failed for %s: %w", path, err)
}

// GetRawQuarantine retrieves the raw quarantine string without following symlinks.
func GetRawQuarantine(path string) (string, error) {
	sz, err := unix.Lgetxattr(path, AttributeName, nil)
	if err != nil {
		if isAttrNotFoundError(err) {
			return "", ErrAttributeNotFound
		}
		return "", fmt.Errorf("lgetxattr size check failed for %s: %w", path, err)
	}

	if sz <= 0 {
		return "", nil
	}

	buf := make([]byte, sz)
	n, err := unix.Lgetxattr(path, AttributeName, buf)
	if err != nil {
		if isAttrNotFoundError(err) {
			return "", ErrAttributeNotFound
		}
		return "", fmt.Errorf("lgetxattr read failed for %s: %w", path, err)
	}

	return string(buf[:n]), nil
}

// RemoveQuarantine strips the com.apple.quarantine attribute without following symlinks.
func RemoveQuarantine(path string) error {
	err := unix.Lremovexattr(path, AttributeName)
	if err == nil || isAttrNotFoundError(err) {
		return nil
	}
	if os.IsNotExist(err) {
		return os.ErrNotExist
	}
	return fmt.Errorf("lremovexattr failed for %s: %w", path, err)
}

// SetRawQuarantine sets the com.apple.quarantine attribute without following symlinks.
func SetRawQuarantine(path string, val string) error {
	err := unix.Lsetxattr(path, AttributeName, []byte(val), 0)
	if err != nil {
		return fmt.Errorf("lsetxattr failed for %s: %w", path, err)
	}
	return nil
}

// ListAttributes returns all extended attribute names on the file without following symlinks.
func ListAttributes(path string) ([]string, error) {
	sz, err := unix.Llistxattr(path, nil)
	if err != nil {
		if isAttrNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("llistxattr size check failed for %s: %w", path, err)
	}

	if sz <= 0 {
		return nil, nil
	}

	buf := make([]byte, sz)
	n, err := unix.Llistxattr(path, buf)
	if err != nil {
		return nil, fmt.Errorf("llistxattr read failed for %s: %w", path, err)
	}

	rawNames := bytes.Split(buf[:n], []byte{0})
	var names []string
	for _, name := range rawNames {
		if len(name) > 0 {
			names = append(names, string(name))
		}
	}
	return names, nil
}

func isAttrNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, unix.ENOATTR) ||
		errors.Is(err, unix.ENODATA) ||
		strings.Contains(err.Error(), "attribute not found")
}
