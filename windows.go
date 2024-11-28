package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

// Helper function to get the full path of the active window's executable
func getActiveWindowExecutablePath() (string, error) {
	hWnd, _, _ := procGetForegroundWindow.Call()

	if hWnd == 0 {
		return "", fmt.Errorf("no active window found")
	}

	// Get the process ID from the window handle
	var processID uint32
	procGetWindowThreadProcessId.Call(hWnd, uintptr(unsafe.Pointer(&processID)))

	// Open the process with the process ID
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010
	hProcess, _, _ := procOpenProcess.Call(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, 0, uintptr(processID))

	if hProcess == 0 {
		return "", fmt.Errorf("unable to open process")
	}
	defer windows.CloseHandle(windows.Handle(hProcess))

	// Get the executable file path
	buffer := make([]uint16, syscall.MAX_PATH)
	procGetModuleFileNameExW.Call(hProcess, 0, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	// Convert the UTF-16 buffer to a Go string
	return syscall.UTF16ToString(buffer), nil
}

// Function to get the title of the foreground window
func getForegroundWindowText() (string, error) {
	hWnd, _, _ := procGetForegroundWindow.Call()
	buf := make([]uint16, 256)
	_, _, err := procGetWindowTextW.Call(hWnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil && err.Error() != "The operation completed successfully." {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}
