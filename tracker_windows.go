//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
)

// getActiveWindow retrieves the title of the currently active window.
func getActiveWindow() (string, error) {
	// Get a handle to the foreground window
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", nil // No active window
	}

	// Get the window's title
	text := make([]uint16, 256)
	_, _, err := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)))
	// We check for a non-nil error, but it's often a zero value on success.
	if err != nil && err.Error() != "The operation completed successfully." {
		return "", err
	}

	return syscall.UTF16ToString(text), nil
}
