package main

import (
	_ "embed"
	"context"
	"log"
	"runtime"
	"time"

	"github.com/getlantern/systray"
)

//go:embed assets/icon.png
var iconLinux []byte

//go:embed assets/icon.ico
var iconWindows []byte

var (
	cfg    *Config
	buf    *SessionBuffer
	cancel context.CancelFunc
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	var err error
	cfg, err = loadConfig()
	if err != nil {
		log.Printf("Config load failed: %v, using defaults", err)
		cfg = defaultConfig()
	}

	buf = newSessionBuffer()

	if runtime.GOOS == "windows" {
		systray.SetIcon(iconWindows)
	} else {
		systray.SetIcon(iconLinux)
	}
	systray.SetTooltip("Dazuukiknie Agent")

	mStatus := systray.AddMenuItem("Not playing", "Currently detected game")
	mStatus.Disable()
	systray.AddSeparator()
	mPush := systray.AddMenuItem("Push update", "Send pending sessions now")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Stop tracking")

	ctx, cancelFn := context.WithCancel(context.Background())
	cancel = cancelFn

	go runDetection(ctx, mStatus)
	go runReporter(ctx)

	go func() {
		for {
			select {
			case <-mPush.ClickedCh:
				go forcePush()
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func onExit() {
	if cancel != nil {
		cancel()
	}
	buf.EndGame()
	sessions := buf.Drain()
	if err := sendReport(sessions, cfg); err != nil {
		log.Printf("Final flush failed: %v", err)
		buf.Restore(sessions)
	}
}

func runDetection(ctx context.Context, mStatus *systray.MenuItem) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var current *DetectedGame

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			detected := Detect(cfg)

			if detected == nil {
				if current != nil {
					log.Printf("Game ended: %s", current.Name)
					buf.EndGame()
					current = nil
					mStatus.SetTitle("Not playing")
				}
				continue
			}

			if current == nil || current.Name != detected.Name {
				if current != nil {
					buf.EndGame()
				}
				current = detected
				buf.StartGame(Game{
					Name:       detected.Name,
					Source:     detected.Source,
					SteamAppID: detected.SteamAppID,
					Process:    detected.Process,
				})
				mStatus.SetTitle("Playing: " + detected.Name)
				log.Printf("Game started: %s (%s)", detected.Name, detected.Source)
			}
		}
	}
}

func runReporter(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if buf.HasPending() {
				forcePush()
			}
		}
	}
}

func forcePush() {
	sessions := buf.Drain()
	if err := sendReport(sessions, cfg); err != nil {
		log.Printf("Report failed: %v", err)
		buf.Restore(sessions)
	}
}
