# CLAUDE.md — dazuukiknie-agent

## Project purpose

Go system tray app that detects active games and reports play sessions to dazuukiknie.nl. Primary detection is via Steam (no config needed); fallback is a user-configured process list.

## File map

| File | Role |
|------|------|
| `main.go` | Systray lifecycle, detection loop, reporter goroutine, tray menu |
| `detector.go` | `Detect()` orchestration, Steam Store API lookup, in-memory name cache |
| `tracker_linux.go` | `/proc` scanning for `SteamAppId`, xdotool active window |
| `tracker_windows.go` | Registry `ActiveGameId` read, Win32 active window + process name |
| `session.go` | `SessionBuffer` (mutex-protected), file persistence to `buffer.json` |
| `reporter.go` | HTTP POST, `machineID()` |
| `config.go` | JSON config load/save, `configDir()`, `dataDir()` |

## Key design decisions

- **Steam detection is process-env based on Linux** (`/proc/*/environ` → `SteamAppId`). On Windows it reads the registry key `HKCU\SOFTWARE\Valve\Steam\ActiveProcess\ActiveGameId`. Both require no Steam API key.
- **Steam name lookup** hits `store.steampowered.com/api/appdetails?appids=N&filters=basic` — no auth, returns `{"730":{"success":true,"data":{"name":"Counter-Strike 2"}}}`. Results are cached in-memory for 24h.
- **Sessions under 10 seconds are discarded** to filter window-switch noise.
- **buffer.json** persists unsent sessions across crashes/restarts. On startup it's loaded and will be sent on the next flush cycle.
- **machine_id** = `sha256(hostname+username)[:8]` as hex — stable, anonymous, no PII.
- **No external dependencies added** beyond what was already vendored. Registry reads use `syscall` directly (not `golang.org/x/sys/windows/registry`) to avoid re-vendoring.

## Platform notes

### Linux
- Steam detection: scan `/proc/*/environ`. Permission errors silently skipped (other users' processes). Works for current user's games without sudo.
- Active window: `xdotool getactivewindow` → `getwindowpid` → `/proc/[pid]/comm`. Requires xdotool.
- Wayland: if `WAYLAND_DISPLAY` is set and `DISPLAY` is not, active window detection is skipped. Steam detection still works.
- Build requires `libayatana-appindicator3-dev` (`sudo apt-get install libayatana-appindicator3-dev`).

### Windows
- Steam detection: `syscall.RegOpenKeyEx` on `HKCU\SOFTWARE\Valve\Steam\ActiveProcess`, reads `ActiveGameId` (REG_DWORD, little-endian).
- Active window: `GetForegroundWindow` → `GetWindowThreadProcessId` → `QueryFullProcessImageNameW` → strip path and `.exe`.
- Cross-compile from Linux: `CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build`.

## Tray menu

Current items (in order):
1. "Playing: [name]" / "Not playing" — disabled label, live-updated every 3s
2. Separator
3. "Push update" — calls `forcePush()`, drains buffer and POSTs immediately
4. Separator
5. "Quit"

Future items planned: login to dazuukiknie.nl, account info display.

## Report payload schema

```json
{
  "machine_id": "a1b2c3d4",
  "sent_at": "2026-03-10T14:00:00Z",
  "sessions": [{
    "game": {
      "name": "Counter-Strike 2",
      "source": "steam",          // "steam" | "config"
      "steam_app_id": 730,        // omitted if source != steam
      "process": "cs2"            // executable name
    },
    "started_at": "...",
    "ended_at": "...",
    "duration_seconds": 5400
  }]
}
```

## Config file

Location: `~/.config/dazuukiknie/config.json` (Linux), `%APPDATA%\dazuukiknie\config.json` (Windows).
Created with defaults on first run. Only `games[]` needs manual editing (for non-Steam titles).

## Common tasks

**Add a non-Steam game:** Edit `config.json`, add `{ "process": "executablename", "name": "Display Name" }` to `games`.

**Change the server endpoint:** Edit `server_url` in `config.json`.

**Force a build check:** `go build ./...` and `go vet ./...`

**Dependency note:** The vendor directory does not include `golang.org/x/sys/windows/registry`. If you want to use that package instead of raw syscall, run `go mod vendor` after adding the import.
