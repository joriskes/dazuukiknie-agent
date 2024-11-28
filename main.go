/**
Todo joris:
- Append to json instead of create new
- Login / connect to dazuukiknie.nl
- Send to dazuukiknie.nl
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/lxn/walk"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

// Windows API functions
var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
)

// Struct for JSON serialization
type AppUsageEntry struct {
	AppName string `json:"app_name"`
	Start   int    `json:"time_start"`
	End     int    `json:"time_end"`
}

// Global variable to store app usage times
var appUsageList []*AppUsageEntry

func getForegroundWindowText() (string, error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	buf := make([]uint16, 256)
	_, _, err := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if err != nil && err.Error() != "The operation completed successfully." {
		return "", err
	}
	return syscall.UTF16ToString(buf), nil
}

func trackForegroundWindow() {
	saveCountdown := 0
	lastApp := ""

	for {
		var now = int(time.Now().Unix())

		// Get the current foreground window title
		currentApp, err := getForegroundWindowText()
		if err != nil {
			fmt.Println("Error getting foreground window:", err)
		} else {
			if currentApp != "" {
				// If the foreground window has changed, update the time spent on the previous window
				if currentApp != lastApp {
					appUsageList = append(appUsageList, &AppUsageEntry{
						AppName: currentApp,
						Start:   now,
						End:     0,
					})
				}

				appUsageList[len(appUsageList)-1].End = now

				// Auto save to file every 10 minutes
				saveCountdown++
				if saveCountdown > 60 {
					saveCountdown = 0
					err := saveAppUsageToFile()
					if err != nil {
						//
					}
				}
			}

			lastApp = currentApp
		}
		// Sleep for 10 seconds before checking again
		time.Sleep(10 * time.Second)
	}
}

// Function to save app usage data to a file
func saveAppUsageToFile() error {
	// Get the executable path
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)

	// Create a flat array to store all app usage entries
	var flatAppUsageList []AppUsageEntry

	// Iterate over the map and append each entry (dereferenced) to the flat array
	for _, entry := range appUsageList {
		flatAppUsageList = append(flatAppUsageList, *entry) // Dereference the pointer
	}

	// Marshal the flat array to JSON
	jsonData, err := json.MarshalIndent(flatAppUsageList, "", "  ")
	if err != nil {
		return err
	}

	// Get the current time and format it for the filename
	currentTime := time.Now()
	fileName := exPath + "/" + currentTime.Format("20060102150405") + ".json"

	// Write the JSON data to a file
	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		// Saved? Clear usage map in memory
		clear(appUsageList)
	}
	return err
}

func main() {
	// Start tracking the foreground window in a separate goroutine
	go trackForegroundWindow()

	// We need either a walk.MainWindow or a walk.Dialog for their message loop.
	// We will not make it visible in this example, though.
	mw, err := walk.NewMainWindow()
	if err != nil {
		log.Fatal(err)
	}

	// We load our icon from a file.
	icon, err := walk.Resources.Icon("APP")
	if err != nil {
		log.Fatal(err)
	}

	// Create the notify icon and make sure we clean it up on exit.
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		log.Fatal(err)
	}
	defer ni.Dispose()

	// Set the icon and a tool tip text.
	if err := ni.SetIcon(icon); err != nil {
		log.Fatal(err)
	}
	if err := ni.SetToolTip("Dazuukiknie agent is running"); err != nil {
		log.Fatal(err)
	}

	// When the left mouse button is pressed, bring up our balloon.
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}

		if err := ni.ShowCustom(
			"Dazuukiknie agent",
			"Running properly",
			icon); err != nil {

			log.Fatal(err)
		}
	})

	// Add manual save function
	saveAction := walk.NewAction()
	if err := saveAction.SetText("S&ave"); err != nil {
		log.Fatal(err)
	}
	saveAction.Triggered().Attach(func() {
		err := saveAppUsageToFile()
		if err != nil {
			walk.MsgBox(mw, "Error", "Failed to save log: "+err.Error(), walk.MsgBoxIconError)
		} else {
			walk.MsgBox(mw, "Success", "Log saved", walk.MsgBoxIconInformation)
		}
	})
	if err := ni.ContextMenu().Actions().Add(saveAction); err != nil {
		log.Fatal(err)
	}

	// Exit action
	exitAction := walk.NewAction()
	if err := exitAction.SetText("E&xit"); err != nil {
		log.Fatal(err)
	}
	exitAction.Triggered().Attach(func() {
		err := saveAppUsageToFile()
		if err != nil {
			//
		}
		walk.App().Exit(0)
	})
	if err := ni.ContextMenu().Actions().Add(exitAction); err != nil {
		log.Fatal(err)
	}

	// The notify icon is hidden initially, so we have to make it visible.
	if err := ni.SetVisible(true); err != nil {
		log.Fatal(err)
	}

	// Now that the icon is visible, we can bring up an info balloon.
	if err := ni.ShowInfo("Dazuukiknie agent", "Click the icon to show again"); err != nil {
		log.Fatal(err)
	}

	// Run the message loop.
	mw.Run()
}
