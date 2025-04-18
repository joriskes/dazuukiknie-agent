// windows.go
package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows" // Use this for CloseHandle and constants
)

// Helper function to get the full path of the active window's executable
func getActiveWindowExecutablePath() (string, error) {
	var hWnd syscall.Handle
	r1, _, err := procGetForegroundWindow.Call()
	// Check error first. If GetLastError is ERROR_SUCCESS, err might be nil or syscall.Errno(0).
	// If err is non-nil and not ERROR_SUCCESS, that's a real error.
	// If r1 is 0, it means no foreground window (e.g., desktop focused).
	if err != nil && err.(syscall.Errno) != 0 { // Check if error is a non-zero errno
		return "", fmt.Errorf("GetForegroundWindow failed: %w", err)
	}
	hWnd = syscall.Handle(r1)
	if hWnd == 0 {
		// This is not necessarily an error, might just be desktop or screen saver
		return "", nil // Return empty string, no error
	}

	// Get the process ID from the window handle
	var processID uint32
	_, _, err = procGetWindowThreadProcessId.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&processID)))
	if err != nil && err.(syscall.Errno) != 0 {
		return "", fmt.Errorf("GetWindowThreadProcessId failed for handle %v: %w", hWnd, err)
	}
	if processID == 0 {
		return "", fmt.Errorf("could not get process ID for handle %v", hWnd)
	}

	// Open the process with necessary permissions
	// PROCESS_QUERY_LIMITED_INFORMATION is safer and often sufficient
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	hProcess, err := windows.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_VM_READ, false, processID)
	if err != nil {
		return "", fmt.Errorf("OpenProcess failed for PID %d: %w", processID, err)
	}
	defer windows.CloseHandle(hProcess) // Ensure handle is closed

	// Get the executable file path using QueryFullProcessImageNameW (more reliable)
	buffer := make([]uint16, windows.MAX_PATH)
	var bufferSize uint32 = uint32(len(buffer))

	// Use QueryFullProcessImageName from kernel32
	procQueryFullProcessImageNameW := kernel32.NewProc("QueryFullProcessImageNameW")
	ret, _, err := procQueryFullProcessImageNameW.Call(
		uintptr(hProcess),
		0, // Use 0 for win32 path format
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferSize)),
	)
	if ret == 0 {
		// Check the specific error
		if err != nil && err.(syscall.Errno) != 0 {
			// Attempt fallback with K32GetModuleFileNameExW if QueryFullProcessImageNameW fails
			// This might happen for certain system processes or due to permissions
			log.Printf("QueryFullProcessImageNameW failed for PID %d (Error: %v), attempting fallback", processID, err)
			buffer2 := make([]uint16, syscall.MAX_PATH)
			ret2, _, err2 := procGetModuleFileNameExW.Call(uintptr(hProcess), 0, uintptr(unsafe.Pointer(&buffer2[0])), uintptr(len(buffer2)))
			if ret2 == 0 {
				if err2 != nil && err2.(syscall.Errno) != 0 {
					return "", fmt.Errorf("GetModuleFileNameExW fallback failed for PID %d: %w", processID, err2)
				}
				return "", fmt.Errorf("GetModuleFileNameExW fallback failed for PID %d with zero return", processID)
			}
			return syscall.UTF16ToString(buffer2), nil // Return result from fallback
		}
		return "", fmt.Errorf("QueryFullProcessImageNameW failed for PID %d with zero return", processID)
	}

	// Convert the UTF-16 buffer to a Go string
	return syscall.UTF16ToString(buffer[:bufferSize]), nil // Use actual size returned
}

// Function to get the title of the foreground window
func getForegroundWindowText() (string, error) {
	var hWnd syscall.Handle
	r1, _, err := procGetForegroundWindow.Call()
	// Same error checking as above
	if err != nil && err.(syscall.Errno) != 0 {
		return "", fmt.Errorf("GetForegroundWindow failed: %w", err)
	}
	hWnd = syscall.Handle(r1)
	if hWnd == 0 {
		return "", nil // No foreground window
	}

	textLen, _, err := procGetWindowTextW.Call(uintptr(hWnd), 0, 0)
	if textLen == 0 {
		if err != nil && err.(syscall.Errno) != 0 {
			// It might fail for some windows, return empty string gracefully
			// log.Printf("GetWindowTextW (length check) failed for handle %v: %v", hWnd, err)
			return "", nil // Treat as empty title if error getting length
		}
		return "", nil // Window has no title
	}

	// Allocate buffer of correct size + 1 for null terminator
	buf := make([]uint16, textLen+1)
	ret, _, err := procGetWindowTextW.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		if err != nil && err.(syscall.Errno) != 0 {
			// Log error but potentially return empty string?
			// log.Printf("GetWindowTextW failed for handle %v: %v", hWnd, err)
			return "", nil // Treat as empty title on error
		}
		// No error, but zero length returned? Return empty.
		return "", nil
	}

	return syscall.UTF16ToString(buf), nil
}
