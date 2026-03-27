# mic-monitor

A cross-platform utility that detects active microphone usage and reports it to Home Assistant via MQTT. Useful for automating "on air" lights, do-not-disturb modes, or any HA automation that should react to you being on a call.

Supports Windows, macOS, and Linux.

## How it works

The project is split into three binaries:

- `miccheck` — detects active microphone capture sessions. Outputs a single JSON line to stdout and exits. Platform-specific implementations:
  - Windows: queries the Core Audio API via direct COM interop (no CGo) to enumerate active capture sessions, including which processes hold them.
  - macOS: uses CoreAudio's `kAudioDevicePropertyDeviceIsRunningSomewhere` via CGo to check all input devices. Reports whether the mic is active but cannot identify individual processes (macOS limitation). Requires Xcode Command Line Tools to build.
- `micmonitor-cli` — terminal-based monitor with colorized ANSI output. Polls `miccheck`, publishes to MQTT, and prints status each cycle. Works on all platforms. Ideal for running in a terminal, as a background service, or via launchd/systemd.
- `micmonitor-tray` — graphical system tray monitor. Same MQTT and polling logic as the CLI, but displays status via a tray icon with a right-click menu. Works on Windows (system tray), macOS (menu bar), and Linux (via libappindicator).

Both monitors shell out to `miccheck` as a subprocess and read its JSON output. This separation keeps the audio detection binary free of UI/shell APIs, and the UI binaries free of low-level audio calls.

Shared logic (MQTT, discovery, polling, config) lives in `internal/monitor/` and is imported by both monitor binaries.

### MQTT topics

All topics are prefixed with `MQTT_TOPIC_PREFIX` (default: `miccheck`).

| Topic | Type | Values |
|---|---|---|
| `{prefix}/{device}/availability` | Availability | `online` / `offline` |
| `{prefix}/{device}/mic_in_use` | Binary sensor state | `ON` / `OFF` |
| `{prefix}/{device}/mic_in_use_by` | Sensor state | CSV of active process names |

HA entities are auto-created via MQTT Discovery on first run.

---

## Requirements

- Windows 10/11, macOS 10.15+, or Linux
- [Go 1.24+](https://go.dev/dl/) (uses `tool` directive in go.mod)
- macOS builds of `miccheck` require Xcode Command Line Tools (`xcode-select --install`)
- An MQTT broker accessible from the machine (e.g. Mosquitto on your Home Assistant host)
- Home Assistant with the MQTT integration configured

---

## Building

### Windows

```bat
go build -o miccheck.exe ./cmd/miccheck/
go build -o micmonitor-cli.exe ./cmd/micmonitor-cli/
go build -ldflags "-H=windowsgui" -o micmonitor-tray.exe ./cmd/micmonitor-tray/
```

The `-H=windowsgui` flag on the tray binary suppresses the console window. Omit it during development to see log output.

### macOS

```sh
xcode-select --install  # if not already done
go build -o miccheck ./cmd/miccheck/
go build -o micmonitor-cli ./cmd/micmonitor-cli/
go build -o micmonitor-tray ./cmd/micmonitor-tray/
```

### Linux

```sh
go build -o miccheck ./cmd/miccheck/
go build -o micmonitor-cli ./cmd/micmonitor-cli/
go build -o micmonitor-tray ./cmd/micmonitor-tray/
```

Note: `miccheck` does not yet have a Linux implementation for microphone detection. The CLI and tray monitors will build and run, but `miccheck` will need a Linux-specific `microphone_linux.go`.

### Cross-compilation

The Windows `miccheck` can be cross-compiled from macOS/Linux since it uses pure `syscall` (no CGo):

```sh
GOOS=windows GOARCH=amd64 go build -o miccheck.exe ./cmd/miccheck/
```

The macOS `miccheck` cannot be cross-compiled because it uses CGo with the CoreAudio framework. Build it natively on a Mac or use a macOS CI runner.

`micmonitor-cli` cross-compiles freely (pure Go, no CGo).

`micmonitor-tray` requires platform-specific C toolchains due to its `systray` dependency.

### Embedding the Windows exe icon

`micmonitor-tray.exe` has an embedded application icon generated via `goversioninfo`. The icon and version metadata are defined in `cmd/micmonitor-tray/versioninfo.json`.

The generated `resource.syso` file is committed to the repo, so a normal `go build` includes the icon automatically. Regenerate it only if you change `versioninfo.json` or the icon file:

```bat
go generate ./cmd/micmonitor-tray/
```

This uses `goversioninfo` which is declared as a tool dependency in `go.mod`.

### Regenerating tray icons

The system tray icons are generated programmatically (no external tools needed):

```sh
go run ./cmd/icongen/
```

---

## Configuration

Copy `.env.example` to `.env` in the same directory as the executables and fill in your values:

```env
# MQTT broker URL. Supports tcp://, ssl://, ws://
MQTT_BROKER=tcp://your-ha-ip:1883

# Broker credentials (leave blank if not required)
MQTT_USERNAME=
MQTT_PASSWORD=

# Prefix for all MQTT topics. Default: miccheck
MQTT_TOPIC_PREFIX=miccheck

# Device name used in HA entity names and MQTT topics.
# Defaults to the machine hostname if not set.
DEVICE_NAME=My PC

# How often to poll microphone status. Default: 5s
POLL_INTERVAL=5s

# Log file path (only used by micmonitor-tray when no console is attached).
# Defaults to micmonitor.log next to the executable.
LOG_FILE=

# Max log file size in bytes before rotation. Default: 5242880 (5MB)
# When the log exceeds this size, it rotates to .log.1 and starts fresh.
LOG_MAX_SIZE=
```

---

## Running

### CLI monitor

Place `miccheck` and `micmonitor-cli` (plus `.env`) in the same folder:

```sh
./micmonitor-cli
```

Outputs colorized status each poll cycle:
- Green `● Mic Idle` — no active capture sessions
- Purple `● Mic Active` with session names — mic is in use
- Red `● MQTT Disconnected` — broker unreachable

Stop with Ctrl+C.

### Tray monitor

Place `miccheck` and `micmonitor-tray` (plus `.env`) in the same folder:

```sh
./micmonitor-tray
```

The tray icon appears in the system tray / menu bar:
- Grey circle — starting up
- Green circle — microphone idle
- Purple circle — microphone active
- Red circle — MQTT disconnected

Right-click menu shows current status, active apps, "View Logs", and "Quit".

### Standalone miccheck

```sh
./miccheck
```

Output example (Windows):
```json
{"active":true,"sessions":["ms-teams.exe","obs64.exe"]}
```

Output example (macOS):
```json
{"active":true,"sessions":["microphone"]}
```

### Running on startup

Windows — create a shortcut to `micmonitor-tray.exe` in:
```
%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup
```

macOS — create a launchd plist or use Login Items in System Settings.

Linux — add to your desktop environment's autostart, or create a systemd user service.

---

## Home Assistant automation

An example automation is included in `homeassistant/automations.yaml`. It turns a light purple when the mic is active, green when idle, and yellow when unavailable/unknown during weekday work hours (9am–5pm). Outside those hours the light turns off.

Replace the placeholder entity IDs and add it to your HA `automations.yaml` or import via the UI.

---

## Project structure

```
.
├── cmd/
│   ├── miccheck/              # Microphone detection binary
│   │   ├── main.go
│   │   ├── microphone_windows.go
│   │   ├── microphone_darwin.go
│   │   ├── com_windows.go
│   │   └── sessions_windows.go
│   ├── micmonitor-cli/        # Terminal monitor (all platforms)
│   │   └── main.go
│   ├── micmonitor-tray/       # System tray monitor (all platforms)
│   │   ├── main.go
│   │   ├── icons.go
│   │   ├── headless_windows.go
│   │   ├── headless_other.go
│   │   ├── versioninfo.json
│   │   ├── resource.syso
│   │   └── *.ico
│   └── icongen/               # Icon generator utility
│       └── main.go
├── internal/
│   └── monitor/               # Shared MQTT, polling, and config logic
│       ├── monitor.go
│       ├── config.go
│       ├── logwriter.go
│       ├── exec_windows.go
│       └── exec_other.go
├── homeassistant/
│   └── automations.yaml       # Example HA automation
├── icons/                     # Source icons
├── .env.example
└── README.md
```

---

## Development notes

- All Windows COM interop is done via `syscall` — no CGo, no external C dependencies.
- macOS detection uses CoreAudio via CGo (`kAudioDevicePropertyDeviceIsRunningSomewhere`). Requires Xcode Command Line Tools.
- Bluetooth audio devices may not report correctly on some macOS versions — this is a known Apple bug.
- `miccheck` is intentionally minimal: query, serialize, exit. No long-running state.
- Both monitors load `.env` from the working directory on startup. When running from an IDE, set the working directory to the folder containing `.env`.
- `micmonitor-tray` automatically writes logs to a file when no console is attached (e.g. built with `-H=windowsgui` on Windows, or launched from a .app bundle on macOS). With a console, logs go to stderr.
- `micmonitor-cli` always logs to stderr.
- To build the tray app with a visible console for debugging:
  ```sh
  go build -o micmonitor-tray ./cmd/micmonitor-tray/
  ```
