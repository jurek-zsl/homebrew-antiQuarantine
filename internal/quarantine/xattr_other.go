//go:build !darwin

package quarantine

import (
	"errors"
	"os"
)

const (
	AttributeName = "com.apple.quarantine"
)

var (
	ErrUnsupportedPlatform = errors.New("extended attributes for macOS Gatekeeper are only supported on Darwin (macOS)")
	ErrAttributeNotFound   = errors.New("attribute not found")
)

func HasQuarantine(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return false, err
	}
	return false, ErrUnsupportedPlatform
}

func GetRawQuarantine(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return "", ErrUnsupportedPlatform
}

func RemoveQuarantine(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return ErrUnsupportedPlatform
}

func SetRawQuarantine(path string, val string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return ErrUnsupportedPlatform
}

func ListAttributes(path string) ([]string, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return nil, ErrUnsupportedPlatform
}
