package main

//go:generate go tool goversioninfo -64

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/getlantern/systray"
	"github.com/joho/godotenv"
)

type micStatus struct {
	Active   bool     `json:"active"`
	Sessions []string `json:"sessions"`
}

type discoveryPayload struct {
	Name              string          `json:"name"`
	StateTopic        string          `json:"state_topic"`
	UniqueID          string          `json:"unique_id"`
	Device            discoveryDevice `json:"device"`
	DeviceClass       string          `json:"device_class,omitempty"`
	PayloadOn         string          `json:"payload_on,omitempty"`
	PayloadOff        string          `json:"payload_off,omitempty"`
	AvailabilityTopic string          `json:"availability_topic"`
	Icon              string          `json:"icon,omitempty"`
}

type discoveryDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// pollMicStatus runs miccheck.exe and parses its JSON output.
func pollMicStatus(miccheckPath string) micStatus {
	cmd := exec.Command(miccheckPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		log.Printf("miccheck exec error: %v", err)
		return micStatus{}
	}
	var status micStatus
	if err := json.Unmarshal(out, &status); err != nil {
		log.Printf("miccheck parse error: %v", err)
		return micStatus{}
	}
	return status
}

// findMiccheck locates miccheck.exe next to the current binary, or on PATH.
func findMiccheck() string {
	// Check next to our own executable first
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "miccheck.exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Fall back to PATH
	p, err := exec.LookPath("miccheck.exe")
	if err == nil {
		return p
	}
	return "miccheck.exe"
}

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
)

// initLogging redirects log output to a file when no console is attached
// (i.e. built with -H=windowsgui). With a console, logs go to stderr as usual.
// Returns the log file path (empty if logging to stderr).
func initLogging() string {
	hwnd, _, _ := getConsoleWindow.Call()
	if hwnd != 0 {
		return "" // console attached, use default stderr logging
	}

	logPath := envOrDefault("LOG_FILE", "")
	if logPath == "" {
		exe, err := os.Executable()
		if err == nil {
			logPath = filepath.Join(filepath.Dir(exe), "micmonitor.log")
		} else {
			logPath = "micmonitor.log"
		}
	}

	maxSize := int64(5 * 1024 * 1024) // 5MB default
	if v := os.Getenv("LOG_MAX_SIZE"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			maxSize = parsed
		}
	}

	w, err := newRotatingWriter(logPath, maxSize)
	if err != nil {
		return "" // can't open log file, silently discard
	}
	log.SetOutput(w)
	log.Printf("Logging to file (no console detected)")
	return logPath
}

func main() {
	_ = godotenv.Load()
	logFilePath := initLogging()
	systray.Run(func() { onReady(logFilePath) }, onExit)
}

func onExit() {}

func onReady(logFilePath string) {
	broker := envOrDefault("MQTT_BROKER", "tcp://localhost:1883")
	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")
	deviceName := envOrDefault("DEVICE_NAME", "")
	topicPrefix := envOrDefault("MQTT_TOPIC_PREFIX", "micmonitor")
	intervalStr := envOrDefault("POLL_INTERVAL", "5s")

	if deviceName == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Fatal("Cannot determine hostname; set DEVICE_NAME in .env")
		}
		deviceName = h
	}

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("Invalid POLL_INTERVAL %q: %v", intervalStr, err)
	}

	slug := strings.ToLower(strings.ReplaceAll(deviceName, " ", "_"))
	miccheckPath := findMiccheck()

	// Setup tray
	systray.SetIcon(iconStartup)
	systray.SetTitle("")
	systray.SetTooltip("Mic Monitor - Starting...")

	mStatus := systray.AddMenuItem("Mic: Idle", "Current microphone status")
	mStatus.Disable()
	mApps := systray.AddMenuItem("Apps: None", "Applications using the mic")
	mApps.Disable()
	systray.AddSeparator()
	mViewLogs := systray.AddMenuItem("View Logs", "Open log file location in Explorer")
	if logFilePath == "" {
		mViewLogs.Disable()
	}
	mQuit := systray.AddMenuItem("Quit", "Exit the application")

	go func() {
		for {
			select {
			case <-mViewLogs.ClickedCh:
				if logFilePath != "" {
					exec.Command("explorer.exe", "/select,", logFilePath).Start()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

	// Build discovery payloads
	availTopic := fmt.Sprintf("%s/%s/availability", topicPrefix, slug)
	binaryStateTopic := fmt.Sprintf("%s/%s/mic_in_use", topicPrefix, slug)
	textStateTopic := fmt.Sprintf("%s/%s/mic_in_use_by", topicPrefix, slug)

	device := discoveryDevice{
		Identifiers:  []string{fmt.Sprintf("%s_%s", topicPrefix, slug)},
		Name:         deviceName,
		Manufacturer: topicPrefix,
		Model:        "Microphone Monitor",
	}

	binaryDiscovery := discoveryPayload{
		Name:              fmt.Sprintf("%s Microphone In Use", deviceName),
		StateTopic:        binaryStateTopic,
		UniqueID:          fmt.Sprintf("%s_mic_in_use", slug),
		Device:            device,
		DeviceClass:       "sound",
		PayloadOn:         "ON",
		PayloadOff:        "OFF",
		AvailabilityTopic: availTopic,
		Icon:              "mdi:microphone",
	}

	textDiscovery := discoveryPayload{
		Name:              fmt.Sprintf("%s Microphone In Use By", deviceName),
		StateTopic:        textStateTopic,
		UniqueID:          fmt.Sprintf("%s_mic_in_use_by", slug),
		Device:            device,
		AvailabilityTopic: availTopic,
		Icon:              "mdi:microphone-message",
	}

	binaryDiscoveryTopic := fmt.Sprintf("homeassistant/binary_sensor/%s_mic_in_use/config", slug)
	textDiscoveryTopic := fmt.Sprintf("homeassistant/sensor/%s_mic_in_use_by/config", slug)

	// MQTT setup
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(fmt.Sprintf("%s-%s", topicPrefix, slug)).
		SetWill(availTopic, "offline", 1, true).
		SetKeepAlive(30 * time.Second).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("Connected to %s", broker)
			publishJSON(c, binaryDiscoveryTopic, binaryDiscovery)
			publishJSON(c, textDiscoveryTopic, textDiscovery)
			publish(c, availTopic, "online", true)
		}).
		SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Printf("MQTT connection lost: %v", err)
		}).
		SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
			log.Print("MQTT reconnecting...")
		})

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("MQTT initial connect failed: %v (will retry in background)", token.Error())
	}
	defer client.Disconnect(1000)

	// Poll loop
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		status := pollMicStatus(miccheckPath)
		connected := client.IsConnected()

		if !connected {
			systray.SetIcon(iconError)
			systray.SetTooltip("Mic Monitor - MQTT Disconnected")
			mStatus.SetTitle("MQTT: Disconnected")
			mApps.SetTitle("Apps: N/A")
		} else if status.Active {
			csv := strings.Join(status.Sessions, ", ")
			publish(client, binaryStateTopic, "ON", false)
			publish(client, textStateTopic, csv, false)

			systray.SetIcon(iconActive)
			systray.SetTooltip(fmt.Sprintf("Mic Monitor - Active: %s", csv))
			mStatus.SetTitle("Mic: Active")
			mApps.SetTitle(fmt.Sprintf("Apps: %s", csv))
		} else {
			publish(client, binaryStateTopic, "OFF", false)
			publish(client, textStateTopic, "", false)

			systray.SetIcon(iconIdle)
			systray.SetTooltip("Mic Monitor - Idle")
			mStatus.SetTitle("Mic: Idle")
			mApps.SetTitle("Apps: None")
		}

		<-ticker.C
	}
}

func publish(client mqtt.Client, topic, payload string, retained bool) {
	token := client.Publish(topic, 1, retained, payload)
	token.Wait()
}

func publishJSON(client mqtt.Client, topic string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	token := client.Publish(topic, 1, true, data)
	token.Wait()
}
