//go:build linux

package main

import (
	"errors"
)

// getActiveWindow retrieves the title of the currently active window using X11.
func getActiveWindow() (string, error) {
	return "", errors.New("not implemented")
}
