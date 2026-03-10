//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

const (
	processQueryLimitedInformation = 0x1000
)

// getSteamRunningApp reads the currently active Steam game from the registry.
// Steam writes the running App ID to HKCU\SOFTWARE\Valve\Steam\ActiveProcess\ActiveGameId.
func getSteamRunningApp() (int, string, error) {
	var k syscall.Handle
	path, err := syscall.UTF16PtrFromString(`SOFTWARE\Valve\Steam\ActiveProcess`)
	if err != nil {
		return 0, "", err
	}

	err = syscall.RegOpenKeyEx(syscall.HKEY_CURRENT_USER, path, 0, syscall.KEY_READ, &k)
	if err != nil {
		return 0, "", fmt.Errorf("steam registry key not found: %w", err)
	}
	defer syscall.RegCloseKey(k)

	var valType uint32
	var buf [4]byte
	bufLen := uint32(len(buf))
	name, _ := syscall.UTF16PtrFromString("ActiveGameId")
	err = syscall.RegQueryValueEx(k, name, nil, &valType, &buf[0], &bufLen)
	if err != nil {
		return 0, "", fmt.Errorf("ActiveGameId not found: %w", err)
	}

	// REG_DWORD is little-endian
	appID := int(buf[0]) | int(buf[1])<<8 | int(buf[2])<<16 | int(buf[3])<<24
	if appID == 0 {
		return 0, "", fmt.Errorf("no active steam game")
	}

	return appID, "", nil
}

// getActiveWindowInfo returns the process name and title of the foreground window.
func getActiveWindowInfo() (string, string, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", "", nil
	}

	// Get window title
	titleBuf := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
	title := syscall.UTF16ToString(titleBuf)

	// Get PID
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", title, nil
	}

	// Get process image name
	handle, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return "", title, nil
	}
	defer procCloseHandle.Call(handle)

	nameBuf := make([]uint16, 260)
	nameLen := uint32(len(nameBuf))
	ret, _, _ := procQueryFullProcessImageName.Call(handle, 0, uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(unsafe.Pointer(&nameLen)))
	if ret == 0 {
		return "", title, nil
	}

	fullPath := syscall.UTF16ToString(nameBuf[:nameLen])
	// Extract just the filename without extension
	procName := baseNameNoExt(fullPath)

	return procName, title, nil
}

func baseNameNoExt(path string) string {
	// Find last backslash or forward slash
	last := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			last = i
			break
		}
	}
	name := path[last+1:]
	// Strip .exe
	if len(name) > 4 && name[len(name)-4:] == ".exe" {
		name = name[:len(name)-4]
	}
	return name
}
