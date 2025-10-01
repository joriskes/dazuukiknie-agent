package main

import (
	_ "embed" // _ blank import trick
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/getlantern/systray"
)

//go:embed assets/icon.png
var iconLinux []byte

//go:embed assets/icon.ico
var iconWindows []byte

// Session holds the data for a single window activity period.
type Session struct {
	WindowTitle string    `json:"window_title"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	DurationSec float64   `json:"duration_seconds"`
}

// currentSession tracks the currently active window.
var currentSession *Session

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	if runtime.GOOS == "windows" {
		systray.SetIcon(iconWindows)
	} else {
		systray.SetIcon(iconLinux)
	}

	systray.SetTitle("dazuukiknie agent")
	systray.SetTooltip("Dazuukiknie is gathering stats")
	quitItem := systray.AddMenuItem("Quit", "Quit the application")

	// Start the window tracking in a separate goroutine
	go startTracking()

	// Handle quit button clicks
	go func() {
		<-quitItem.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// When the app is about to exit, end the final session and send it.
	if currentSession != nil {
		endCurrentSession()
		log.Printf("Final session ended: %s (%.2f s)", currentSession.WindowTitle, currentSession.DurationSec)
		sendSessionReport(*currentSession)
	}
	fmt.Println("Exiting application...")
}

// startTracking is the main loop that polls for the active window.
func startTracking() {
	// Poll for the active window every 2 seconds. Adjust as needed.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastTitle string

	for range ticker.C {
		title, err := getActiveWindow()
		if err != nil {
			log.Printf("Error getting active window: %v", err)
			continue
		}

		// We only care about actual windows with titles.
		if title == "" || title == "Default" { // "Default" can be an X11 default name
			continue
		}

		// If the window title has changed...
		if title != lastTitle {
			log.Printf("Window changed to: '%s'", title)

			// ...and if there was a previous session, end it and report it.
			if currentSession != nil {
				endCurrentSession()
				log.Printf("Session ended: %s (%.2f s)", currentSession.WindowTitle, currentSession.DurationSec)
				go sendSessionReport(*currentSession) // Send report in a goroutine
			}

			// Start a new session for the new window.
			startNewSession(title)
			lastTitle = title
		}
	}
}

func startNewSession(title string) {
	currentSession = &Session{
		WindowTitle: title,
		StartTime:   time.Now(),
	}
	log.Printf("Session started: %s", title)
}

func endCurrentSession() {
	if currentSession != nil {
		currentSession.EndTime = time.Now()
		currentSession.DurationSec = time.Since(currentSession.StartTime).Seconds()
	}
}

// sendSessionReport sends the completed session data to your website.
func sendSessionReport(session Session) {
	// ❗ Replace this URL with your actual API endpoint.
	//apiURL := "https://your-website.com/api/stats"

	jsonData, err := json.Marshal(session)
	if err != nil {
		log.Printf("Error creating JSON data: %v", err)
		return
	}

	log.Printf("Sending session report for: %s", session.WindowTitle)
	log.Printf("JSON data: %s", string(jsonData))

	//resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	//if err != nil {
	//	log.Printf("Error sending session report: %v", err)
	//	return
	//}
	//defer resp.Body.Close()
	//
	//if resp.StatusCode >= 300 {
	//	log.Printf("API returned an error status: %s", resp.Status)
	//} else {
	//	log.Println("Session report sent successfully.")
	//}
}
