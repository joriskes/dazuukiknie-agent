package main

import (
	_ "embed" // _ blank import trick to fix embed being removed
	"fmt"
	"runtime"

	"github.com/getlantern/systray"
)

//go:embed assets/icon.png
var iconLinux []byte

//go:embed assets/icon.ico
var iconWindows []byte

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

	for {
		select {
		case <-quitItem.ClickedCh:
			systray.Quit()
			fmt.Println("Quitting...")
			return
		}
	}
}

func onExit() {
	// clean up
	fmt.Println("Exiting application...")
}
