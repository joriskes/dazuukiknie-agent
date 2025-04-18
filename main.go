// main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/lxn/walk"
)

// Windows API functions
var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procGetModuleFileNameExW     = kernel32.NewProc("K32GetModuleFileNameExW")
)

// Struct for JSON serialization
type AppUsageEntry struct {
	AppName        string `json:"app_name"`
	ExecutablePath string `json:"executable_path"`
	Start          int64  `json:"time_start"` // Use int64 for Unix timestamp
	End            int64  `json:"time_end"`   // Use int64 for Unix timestamp
}

// Global variables with mutex
var (
	appUsageList  []*AppUsageEntry
	appUsageMutex sync.Mutex // Mutex to protect appUsageList
)

// Function to track the foreground window and capture usage data
func trackForegroundWindow() {
	saveCountdown := 0
	lastExePath := ""
	ticker := time.NewTicker(10 * time.Second) // Use a ticker for regular intervals
	defer ticker.Stop()

	for range ticker.C { // Loop based on ticker
		now := time.Now().Unix()

		// Get the current foreground window title and executable path
		currentAppTitle, errTitle := getForegroundWindowText()
		currentExePath, errPath := getActiveWindowExecutablePath()

		// Log errors if any occurred
		if errTitle != nil {
			log.Printf("Error getting foreground window title: %v\n", errTitle)
			// Decide if you want to continue or skip this cycle
			// continue
		}
		if errPath != nil {
			log.Printf("Error getting foreground window executable path: %v\n", errPath)
			// If we can't get the path, we probably can't track accurately
			// We might want to update the end time of the last known app here
			appUsageMutex.Lock()
			if len(appUsageList) > 0 {
				appUsageList[len(appUsageList)-1].End = now
			}
			appUsageMutex.Unlock()
			lastExePath = "" // Reset last path as we lost track
			continue         // Skip to next tick
		}

		// Lock the mutex for accessing appUsageList
		appUsageMutex.Lock()

		// Check if the foreground window executable path has changed
		if currentExePath != "" && currentExePath != lastExePath {
			// Update the end time of the *previous* app's entry if there was one
			if len(appUsageList) > 0 {
				appUsageList[len(appUsageList)-1].End = now
			}

			// Add a new entry for the current app
			appUsageList = append(appUsageList, &AppUsageEntry{
				AppName:        currentAppTitle, // Use title fetched earlier
				ExecutablePath: currentExePath,
				Start:          now,
				End:            now, // Initial end time is the same as start
			})
			log.Printf("App changed: %s (%s)\n", currentAppTitle, currentExePath)
			lastExePath = currentExePath // Update last known path

		} else if currentExePath != "" && len(appUsageList) > 0 {
			// If the app hasn't changed, update the end time of the current (last) entry
			appUsageList[len(appUsageList)-1].End = now
		} else if currentExePath == "" {
			// Handle case where no valid foreground app path is found (e.g., desktop)
			if len(appUsageList) > 0 {
				appUsageList[len(appUsageList)-1].End = now // Update end time of the last app
			}
			lastExePath = "" // Reset last path
		}

		// Check for auto-save (every 60 * 10 seconds = 10 minutes)
		saveCountdown++
		if saveCountdown >= 60 {
			saveCountdown = 0
			// We need to potentially unlock before calling saveAppUsageToFile
			// if it also needs the lock, or pass the data carefully.
			// Let's create a copy of the data to save.
			listToSave := make([]*AppUsageEntry, len(appUsageList))
			copy(listToSave, appUsageList)
			appUsageList = nil // Clear the original list *inside the lock*

			appUsageMutex.Unlock() // Unlock before potentially long file I/O

			err := saveAppUsageToFile(listToSave) // Save the copy
			if err != nil {
				log.Println("Error auto-saving app usage to file:", err)
				// Decide if you want to re-add the unsaved data (could lead to duplicates or large memory)
				// For now, we'll just log the error. The data for that period is lost.
			} else {
				log.Println("App usage auto-saved successfully.")
			}

		} else {
			// If not saving, unlock the mutex here
			appUsageMutex.Unlock()
		}
	}
}

// Function to save app usage data to a JSON file
// Takes the list to save as an argument
func saveAppUsageToFile(listToSave []*AppUsageEntry) error {
	if len(listToSave) == 0 {
		log.Println("No app usage data to save.")
		return nil // Nothing to save
	}

	var exPath = ""

	// Get the executable path
	ex, err := os.Executable()
	if err != nil {
		// Use a default path or log fatal? Using current dir for now.
		log.Printf("Warning: Could not get executable path: %v. Using current directory.", err)
		exPath = "." // Fallback to current directory
	} else {
		exPath = filepath.Dir(ex)
	}

	// --- Steam Info Saving (Consider if this needs to run every time) ---
	steamInfo, err := buildSteamInfo()
	if err != nil {
		// Output the file content
		log.Println("Failed building steaminfo:", err)
		// Don't necessarily stop the app usage save because of steam info
	} else {
		steamInfoPath := filepath.Join(exPath, "steaminfo.json")
		err = os.WriteFile(steamInfoPath, []byte(steamInfo), 0644) // Use 0644 permission
		if err != nil {
			log.Printf("Failed saving steaminfo file to %s: %v\n", steamInfoPath, err)
		}
	}
	// --- End Steam Info Saving ---

	// Create a flat array to store all app usage entries
	var flatAppUsageList []AppUsageEntry

	// Iterate over the list and append each entry (dereferenced) to the flat array
	for _, entry := range listToSave {
		if entry != nil { // Add nil check just in case
			flatAppUsageList = append(flatAppUsageList, *entry) // Dereference the pointer
		}
	}

	// Marshal the flat array to JSON
	jsonData, err := json.MarshalIndent(flatAppUsageList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal app usage data: %w", err)
	}

	// Get the current time and format it for the filename
	currentTime := time.Now()
	fileName := filepath.Join(exPath, currentTime.Format("20060102_150405")+".json") // Use underscore, join path

	log.Printf("Saving app usage data to: %s\n", fileName)

	// Write the JSON data to a file
	err = os.WriteFile(fileName, jsonData, 0644) // Use 0644 permission
	if err != nil {
		return fmt.Errorf("failed to write app usage file %s: %w", fileName, err)
	}

	// Clearing the list is now handled in the caller (trackForegroundWindow or manual save)
	return nil
}

func main() {
	logFilePath := "app_activity.log" // Changed name for clarity
	// Open a log file for appending (create if it doesn't exist)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback to stderr if log file fails
		log.Printf("Failed to open log file %s: %v. Logging to stderr.", logFilePath, err)
	} else {
		log.SetOutput(logFile) // Set log output to file
		defer logFile.Close()  // Ensure log file is closed on exit
	}

	log.Println("-----------------------------------------------------")
	log.Println("Application Starting")
	log.Println("-----------------------------------------------------")

	// We need either a walk.MainWindow or a walk.Dialog for their message loop.
	mw, err := walk.NewMainWindow()
	if err != nil {
		log.Fatalf("Failed to create main window: %v", err)
	}

	// We load our icon from a file. Adjust "APP" if your resource name is different
	icon, err := walk.Resources.Icon("APP")
	// Fallback or error handling for icon
	if err != nil {
		log.Printf("Warning: Could not load icon resource 'APP': %v. Using default.", err)
		// Optionally load a default icon or proceed without one
		// For now, we'll proceed, but the tray icon might be missing/default
		// icon, err = walk.Resources.Icon("DEFAULT_ICON_NAME") // If you have a default
	}

	// Create the notify icon and make sure we clean it up on exit.
	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		log.Fatalf("Failed to create notify icon: %v", err)
	}
	defer ni.Dispose() // Schedule disposal

	// Set the icon and a tool tip text.
	if icon != nil { // Only set icon if loaded successfully
		if err := ni.SetIcon(icon); err != nil {
			log.Printf("Failed to set notify icon: %v", err) // Log error, don't crash
		}
	}
	tooltip := "Dazuukiknie agent is running"
	if err := ni.SetToolTip(tooltip); err != nil {
		log.Printf("Failed to set tooltip: %v", err) // Log error, don't crash
	}

	// When the left mouse button is pressed, show status (changed from balloon)
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		// Using a simple message box instead of custom balloon which can be problematic
		var lastAppInfo string
		appUsageMutex.Lock()
		if len(appUsageList) > 0 {
			lastEntry := appUsageList[len(appUsageList)-1]
			lastAppInfo = fmt.Sprintf("Last tracked: %s", lastEntry.AppName)
		} else {
			lastAppInfo = "No app tracked yet."
		}
		appUsageMutex.Unlock()

		walk.MsgBox(mw, "Dazuukiknie Agent Status", "Agent is running.\n"+lastAppInfo, walk.MsgBoxIconInformation)
		// Custom balloon alternative (keep if you prefer, but check for errors)
		/*
		   if icon != nil { // Check icon again
		       if err := ni.ShowCustom(
		           "Dazuukiknie agent",
		           "Running properly",
		           icon); err != nil {
		           log.Printf("Failed to show custom notification: %v", err)
		       }
		   }
		*/
	})

	// Add manual save function
	saveAction := walk.NewAction()
	if err := saveAction.SetText("S&ave Now"); err != nil {
		log.Fatalf("Failed to create save action: %v", err)
	}
	saveAction.Triggered().Attach(func() {
		log.Println("Manual save triggered.")
		// Lock, copy data, clear original, unlock
		appUsageMutex.Lock()
		listToSave := make([]*AppUsageEntry, len(appUsageList))
		copy(listToSave, appUsageList)
		appUsageList = nil // Clear the main list
		appUsageMutex.Unlock()

		err := saveAppUsageToFile(listToSave) // Save the copy
		if err != nil {
			log.Printf("Manual save failed: %v\n", err)
			walk.MsgBox(mw, "Error", "Failed to save log: "+err.Error(), walk.MsgBoxIconError)
			// Consider re-adding listToSave back to appUsageList if save fails?
			// Be careful about duplicate data on next save attempt.
		} else {
			log.Println("Manual save successful.")
			walk.MsgBox(mw, "Success", "Log saved successfully.", walk.MsgBoxIconInformation)
		}
	})
	if err := ni.ContextMenu().Actions().Add(saveAction); err != nil {
		log.Fatalf("Failed to add save action to menu: %v", err)
	}

	// Exit action
	exitAction := walk.NewAction()
	if err := exitAction.SetText("E&xit"); err != nil {
		log.Fatalf("Failed to create exit action: %v", err)
	}
	exitAction.Triggered().Attach(func() {
		log.Println("Exit triggered. Performing final save.")
		// Perform final save before exiting
		appUsageMutex.Lock()
		listToSave := make([]*AppUsageEntry, len(appUsageList))
		copy(listToSave, appUsageList)
		appUsageList = nil // Clear list
		appUsageMutex.Unlock()

		err := saveAppUsageToFile(listToSave)
		if err != nil {
			log.Println("Error saving data on exit:", err)
			// Maybe show a message box? But app is exiting anyway.
		} else {
			log.Println("Final save successful.")
		}
		log.Println("Application Exiting")
		walk.App().Exit(0)
	})
	if err := ni.ContextMenu().Actions().Add(exitAction); err != nil {
		log.Fatalf("Failed to add exit action to menu: %v", err)
	}

	// The notify icon is hidden initially, so we have to make it visible.
	if err := ni.SetVisible(true); err != nil {
		log.Fatalf("Failed to make notify icon visible: %v", err)
	}

	// Show initial info balloon (optional, can be annoying)
	/*
		if icon != nil { // Check icon exists
			if err := ni.ShowInfo("Dazuukiknie agent", "Agent started and running."); err != nil {
				log.Printf("Failed to show initial info balloon: %v", err)
			}
		}
	*/

	// Start tracking the foreground window in a separate goroutine AFTER UI setup
	go trackForegroundWindow()

	log.Println("Main message loop starting.")
	// Run the message loop. This blocks until the application exits.
	mw.Run()

	log.Println("Main message loop finished.") // Should only log after exit triggered
}
