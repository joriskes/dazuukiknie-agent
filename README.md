# Dazuukiknie Agent

System tray app that tracks what games you're playing and for how long, then reports sessions to [dazuukiknie.nl](https://dazuukiknie.nl).

## How it works

- Detects Steam games automatically by scanning running processes for `SteamAppId` — no configuration needed
- Looks up game names via the Steam Store API (no API key required)
- Falls back to a user-defined process list for non-Steam games
- Buffers sessions locally and sends them every 5 minutes, or on demand via "Push update" in the tray menu
- Unsent sessions survive crashes and are sent on next startup

## Tray menu

```
Playing: Counter-Strike 2
---
Push update
---
Quit
```

## Configuration

Config is created automatically on first run at:
- **Linux:** `~/.config/dazuukiknie/config.json`
- **Windows:** `%APPDATA%\dazuukiknie\config.json`

```json
{
  "server_url": "https://dazuukiknie.nl/api/sessions",
  "games": [
    { "process": "factorio", "name": "Factorio" },
    { "process": "RimWorldWin64", "name": "RimWorld" }
  ]
}
```

`games` is only needed for non-Steam titles. Use the executable name without `.exe`.

## Session buffer

Completed sessions are buffered at:
- **Linux:** `~/.local/share/dazuukiknie/buffer.json`
- **Windows:** `%LOCALAPPDATA%\dazuukiknie\buffer.json`

## Report payload

```json
{
  "machine_id": "a1b2c3d4e5f6a7b8",
  "sent_at": "2026-03-10T14:00:00Z",
  "sessions": [
    {
      "game": {
        "name": "Counter-Strike 2",
        "source": "steam",
        "steam_app_id": 730,
        "process": "cs2"
      },
      "started_at": "2026-03-10T12:00:00Z",
      "ended_at": "2026-03-10T13:30:00Z",
      "duration_seconds": 5400
    }
  ]
}
```

`machine_id` is a stable anonymous identifier derived from hostname + username (first 8 bytes of SHA-256). No PII is sent.

## Build

```bash
# Linux
CGO_ENABLED=1 go build -o dazuukiknie-agent

# Windows (cross-compile from Linux)
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o dazuukiknie-agent.exe
```

### Linux build dependency

```
sudo apt-get install libayatana-appindicator3-dev
```

If you see:
```
Package 'ayatana-appindicator3-0.1', required by 'virtual:world', not found
```
...that's the missing package above.

### Non-Steam game detection on Linux

Requires `xdotool` for active window detection:
```
sudo apt-get install xdotool
```

Without it, only Steam games are detected. Wayland-only sessions (no `$DISPLAY`) skip active window detection entirely — Steam detection still works.
