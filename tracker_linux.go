//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// getSteamRunningApp scans /proc for a process owned by the current user
// that has SteamAppId set in its environment.
func getSteamRunningApp() (int, string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, "", err
	}

	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}

		envPath := fmt.Sprintf("/proc/%d/environ", pid)
		appID, err := readSteamAppID(envPath)
		if err != nil || appID == 0 {
			continue
		}

		commPath := fmt.Sprintf("/proc/%d/comm", pid)
		comm, _ := os.ReadFile(commPath)
		process := strings.TrimSpace(string(comm))

		return appID, process, nil
	}

	return 0, "", fmt.Errorf("no steam game running")
}

// readSteamAppID reads the SteamAppId value from a /proc/[pid]/environ file.
// Returns 0 if not found. Returns an error if the file can't be read (e.g. wrong user).
func readSteamAppID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// Permission denied = another user's process, skip silently
		return 0, err
	}
	// environ is NUL-separated key=value pairs
	for _, entry := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(entry, "SteamAppId=") {
			val := strings.TrimPrefix(entry, "SteamAppId=")
			id, err := strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return 0, err
			}
			if id > 0 {
				return id, nil
			}
		}
	}
	return 0, nil
}

// getActiveWindowInfo returns the process name and window title of the focused window.
// Requires xdotool. Does not work on Wayland.
func getActiveWindowInfo() (string, string, error) {
	if os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("DISPLAY") == "" {
		return "", "", fmt.Errorf("wayland-only session: active window detection not supported")
	}

	out, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return "", "", fmt.Errorf("xdotool not available: %w", err)
	}
	winID := strings.TrimSpace(string(out))

	out, err = exec.Command("xdotool", "getwindowpid", winID).Output()
	if err != nil {
		return "", "", fmt.Errorf("xdotool getwindowpid: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return "", "", err
	}

	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	comm, err := os.ReadFile(commPath)
	if err != nil {
		return "", "", err
	}
	procName := strings.TrimSpace(string(comm))

	out, err = exec.Command("xdotool", "getwindowname", winID).Output()
	if err != nil {
		return procName, "", nil
	}
	title := strings.TrimSpace(string(out))

	return procName, title, nil
}
