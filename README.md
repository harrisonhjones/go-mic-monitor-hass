# mic-monitor

A Windows utility that detects active microphone usage and reports it to Home Assistant via MQTT. Useful for automating "on air" lights, do-not-disturb modes, or any HA automation that should react to you being on a call.

## How it works

The project is split into two binaries to keep concerns (and security scanner profiles) separate:

- `miccheck.exe` — queries the Windows Core Audio API (via direct COM interop, no CGo) to enumerate active capture sessions on the default microphone. Outputs a single JSON line to stdout and exits.
- `micmonitor.exe` — a system tray application that polls `miccheck.exe` on a configurable interval, updates the tray icon dynamically, and publishes state to Home Assistant via MQTT Discovery.

`micmonitor` runs `miccheck` as a hidden subprocess (no console flash) and reads its output. This separation means the COM/audio binary has no shell or tray APIs, and the tray binary has no low-level audio calls.

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

- Windows 10/11
- [Go 1.21+](https://go.dev/dl/)
- An MQTT broker accessible from the machine (e.g. Mosquitto running on your Home Assistant host)
- Home Assistant with the MQTT integration configured

---

## Building

Clone the repo and run:

```bat
go build -o miccheck.exe ./cmd/miccheck/
go build -ldflags "-H=windowsgui" -o micmonitor.exe ./cmd/micmonitor/
```

The `-H=windowsgui` flag suppresses the console window for `micmonitor`. Omit it during development if you want to see log output.

To regenerate the tray icons (requires no external tools — pure Go):

```bat
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

# Log file path (only used when no console is attached).
# Defaults to micmonitor.log next to the executable.
LOG_FILE=

# Max log file size in bytes before rotation. Default: 5242880 (5MB)
# When the log exceeds this size, it rotates to .log.1 and starts fresh.
LOG_MAX_SIZE=
```

---

## Running

Place `miccheck.exe`, `micmonitor.exe`, and `.env` in the same folder, then run:

```bat
micmonitor.exe
```

The tray icon appears in the system tray:
- Green circle — microphone idle
- Purple circle — microphone active
- Red circle — MQTT connection failed

Right-clicking the icon shows the current mic state and which applications are using it.

To run `miccheck` standalone (useful for testing or scripting):

```bat
miccheck.exe
```

Output example:
```json
{"active":true,"sessions":["ms-teams.exe","obs64.exe"]}
```

### Running on startup

To launch `micmonitor` automatically at login, create a shortcut to `micmonitor.exe` and place it in:

```
%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup
```

Or use Task Scheduler to run it at logon with "Run only when user is logged on" and no console window.

---

## Home Assistant automation

An example automation is included in `homeassistant/automations.yaml`. It turns a light purple when the mic is active and green when idle during weekday work hours (9am–5pm), and turns the light off outside those hours.

To use it, replace the placeholder entity IDs and add it to your HA `automations.yaml` or import it via the UI.

---

## Project structure

```
.
├── cmd/
│   ├── miccheck/           # COM audio binary (no tray/shell APIs)
│   │   ├── main.go
│   │   ├── microphone.go
│   │   ├── com_windows.go
│   │   └── sessions_windows.go
│   ├── micmonitor/         # System tray + MQTT binary
│   │   ├── main.go
│   │   ├── icons.go
│   │   ├── logwriter.go
│   │   └── *.ico
│   └── icongen/            # Icon generator utility
│       └── main.go
├── homeassistant/
│   └── automations.yaml    # Example HA automation
├── icons/                  # Source icons (also copied to cmd/micmonitor)
├── .env.example
└── README.md
```

---

## Development notes

- All Windows COM interop is done via `syscall` — no CGo, no external C dependencies.
- `miccheck` is intentionally minimal: query, serialize, exit. No long-running state.
- `micmonitor` loads `.env` from the working directory on startup. When running from an IDE, make sure the working directory is set to the project root or the folder containing `.env`.
- When built with `-H=windowsgui` (no console), `micmonitor` automatically writes logs to a file (`micmonitor.log` next to the exe). When a console is attached (dev builds), logs go to stderr as usual.
- To build with a visible console for log output during development:
  ```bat
  go build -o micmonitor.exe ./cmd/micmonitor/
  ```
- To cross-check what the audio API is seeing without running the full monitor:
  ```bat
  miccheck.exe
  ```
