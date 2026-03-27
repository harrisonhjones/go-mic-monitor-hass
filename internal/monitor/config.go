package monitor

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Config holds all configuration for the monitor.
type Config struct {
	Broker       string
	Username     string
	Password     string
	DeviceName   string
	TopicPrefix  string
	PollInterval time.Duration
	MiccheckPath string
}

// EnvOrDefault returns the env var value or a fallback.
func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// FindMiccheck locates the miccheck binary next to the current executable or on PATH.
func FindMiccheck() string {
	name := "miccheck"
	if runtime.GOOS == "windows" {
		name = "miccheck.exe"
	}

	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	p, err := exec.LookPath(name)
	if err == nil {
		return p
	}
	return name
}

// LoadConfigFromEnv builds a Config from environment variables.
func LoadConfigFromEnv() (Config, error) {
	deviceName := EnvOrDefault("DEVICE_NAME", "")
	if deviceName == "" {
		h, err := os.Hostname()
		if err != nil {
			return Config{}, err
		}
		deviceName = h
	}

	intervalStr := EnvOrDefault("POLL_INTERVAL", "5s")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Broker:       EnvOrDefault("MQTT_BROKER", "tcp://localhost:1883"),
		Username:     os.Getenv("MQTT_USERNAME"),
		Password:     os.Getenv("MQTT_PASSWORD"),
		DeviceName:   deviceName,
		TopicPrefix:  EnvOrDefault("MQTT_TOPIC_PREFIX", "miccheck"),
		PollInterval: interval,
		MiccheckPath: FindMiccheck(),
	}, nil
}
