package monitor

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// StatusUpdate is emitted each poll cycle.
type StatusUpdate struct {
	Connected bool
	Active    bool
	Sessions  []string
}

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

// Run starts the MQTT connection and poll loop, calling onUpdate each cycle.
// It blocks until stop is closed.
func Run(cfg Config, onUpdate func(StatusUpdate), stop <-chan struct{}) error {
	slug := strings.ToLower(strings.ReplaceAll(cfg.DeviceName, " ", "_"))

	availTopic := fmt.Sprintf("%s/%s/availability", cfg.TopicPrefix, slug)
	binaryStateTopic := fmt.Sprintf("%s/%s/mic_in_use", cfg.TopicPrefix, slug)
	textStateTopic := fmt.Sprintf("%s/%s/mic_in_use_by", cfg.TopicPrefix, slug)

	device := discoveryDevice{
		Identifiers:  []string{fmt.Sprintf("%s_%s", cfg.TopicPrefix, slug)},
		Name:         cfg.DeviceName,
		Manufacturer: cfg.TopicPrefix,
		Model:        "Microphone Monitor",
	}

	binaryDiscovery := discoveryPayload{
		Name:              fmt.Sprintf("%s Microphone In Use", cfg.DeviceName),
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
		Name:              fmt.Sprintf("%s Microphone In Use By", cfg.DeviceName),
		StateTopic:        textStateTopic,
		UniqueID:          fmt.Sprintf("%s_mic_in_use_by", slug),
		Device:            device,
		AvailabilityTopic: availTopic,
		Icon:              "mdi:microphone-message",
	}

	binaryDiscoveryTopic := fmt.Sprintf("homeassistant/binary_sensor/%s_mic_in_use/config", slug)
	textDiscoveryTopic := fmt.Sprintf("homeassistant/sensor/%s_mic_in_use_by/config", slug)

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(fmt.Sprintf("%s-%s", cfg.TopicPrefix, slug)).
		SetWill(availTopic, "offline", 1, true).
		SetKeepAlive(30 * time.Second).
		SetAutoReconnect(true).
		SetConnectTimeout(10 * time.Second).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("Connected to %s", cfg.Broker)
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

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("MQTT initial connect failed: %v (will retry in background)", token.Error())
	}
	defer client.Disconnect(1000)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		status := pollMiccheck(cfg.MiccheckPath)
		connected := client.IsConnected()

		if connected {
			if status.Active {
				csv := strings.Join(status.Sessions, ", ")
				publish(client, binaryStateTopic, "ON", false)
				publish(client, textStateTopic, csv, false)
			} else {
				publish(client, binaryStateTopic, "OFF", false)
				publish(client, textStateTopic, "", false)
			}
		}

		onUpdate(StatusUpdate{
			Connected: connected,
			Active:    status.Active,
			Sessions:  status.Sessions,
		})

		select {
		case <-stop:
			return nil
		case <-ticker.C:
		}
	}
}

func pollMiccheck(path string) micStatus {
	cmd := exec.Command(path)
	hideMiccheckWindow(cmd)
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
