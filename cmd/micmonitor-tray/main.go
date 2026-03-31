package main

//go:generate go tool goversioninfo -64

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/getlantern/systray"
	"github.com/joho/godotenv"

	"miccheck/internal/monitor"
)

var (
	mStatus   *systray.MenuItem
	mApps     *systray.MenuItem
	mViewLogs *systray.MenuItem
)

func main() {
	_ = godotenv.Load()
	logFilePath := initLogging()
	systray.Run(func() { onReady(logFilePath) }, func() {})
}

func onReady(logFilePath string) {
	cfg, err := monitor.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// Setup tray
	systray.SetIcon(iconStartup)
	systray.SetTitle("")
	systray.SetTooltip("Mic Monitor - Starting...")

	mStatus = systray.AddMenuItem("Mic: Starting...", "Current microphone status")
	mStatus.Disable()
	mApps = systray.AddMenuItem("Apps: None", "Applications using the mic")
	mApps.Disable()
	systray.AddSeparator()
	mViewLogs = systray.AddMenuItem("View Logs", "Open log file location")
	if logFilePath == "" {
		mViewLogs.Disable()
	}
	mQuit := systray.AddMenuItem("Quit", "Exit the application")

	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-mViewLogs.ClickedCh:
				if logFilePath != "" {
					openFileInExplorer(logFilePath)
				}
			case <-mQuit.ClickedCh:
				close(stop)
				systray.Quit()
				return
			}
		}
	}()

	go func() {
		err := monitor.Run(cfg, func(s monitor.StatusUpdate) {
			updateTray(s)
		}, stop)
		if err != nil {
			log.Printf("Monitor error: %v", err)
		}
	}()
}

func updateTray(s monitor.StatusUpdate) {
	if !s.Connected {
		systray.SetIcon(iconError)
		systray.SetTooltip("Mic Monitor - MQTT Disconnected")
		mStatus.SetTitle("MQTT: Disconnected")
		mApps.SetTitle("Apps: N/A")
		return
	}

	if s.Active {
		csv := strings.Join(s.Sessions, ", ")
		systray.SetIcon(iconActive)
		systray.SetTooltip(fmt.Sprintf("Mic Monitor - Active: %s", csv))
		mStatus.SetTitle("Mic: Active")
		mApps.SetTitle(fmt.Sprintf("Apps: %s", csv))
	} else {
		systray.SetIcon(iconIdle)
		systray.SetTooltip("Mic Monitor - Idle")
		mStatus.SetTitle("Mic: Idle")
		mApps.SetTitle("Apps: None")
	}
}

func openFileInExplorer(path string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("explorer.exe", "/select,", path).Start()
	case "darwin":
		exec.Command("open", "-R", path).Start()
	default:
		exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

func initLogging() string {
	if !isHeadless() {
		return ""
	}

	logPath := monitor.EnvOrDefault("LOG_FILE", "")
	if logPath == "" {
		exe, err := os.Executable()
		if err == nil {
			logPath = filepath.Join(filepath.Dir(exe), "micmonitor.log")
		} else {
			logPath = "micmonitor.log"
		}
	}

	maxSize := int64(5 * 1024 * 1024)
	if v := os.Getenv("LOG_MAX_SIZE"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			maxSize = parsed
		}
	}

	w, err := monitor.NewRotatingWriter(logPath, maxSize)
	if err != nil {
		return ""
	}
	log.SetOutput(w)
	log.Printf("Logging to file (no console detected)")
	return logPath
}
