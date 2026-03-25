package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/joho/godotenv/autoload"
)

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

func main() {
	broker := envOrDefault("MQTT_BROKER", "tcp://localhost:1883")
	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")
	deviceName := envOrDefault("DEVICE_NAME", "")
	topicPrefix := envOrDefault("MQTT_TOPIC_PREFIX", "miccheck")
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
	availTopic := fmt.Sprintf("%s/%s/availability", topicPrefix, slug)

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(fmt.Sprintf("%s-%s", topicPrefix, slug)).
		SetWill(availTopic, "offline", 1, true).
		SetKeepAlive(30 * time.Second).
		SetAutoReconnect(true)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect failed: %v", token.Error())
	}
	defer client.Disconnect(1000)
	log.Printf("Connected to %s", broker)

	device := discoveryDevice{
		Identifiers:  []string{fmt.Sprintf("%s_%s", topicPrefix, slug)},
		Name:         deviceName,
		Manufacturer: topicPrefix,
		Model:        "Microphone Monitor",
	}

	binaryStateTopic := fmt.Sprintf("%s/%s/mic_in_use", topicPrefix, slug)
	textStateTopic := fmt.Sprintf("%s/%s/mic_in_use_by", topicPrefix, slug)

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

	publishJSON(client, fmt.Sprintf("homeassistant/binary_sensor/%s_mic_in_use/config", slug), binaryDiscovery)
	publishJSON(client, fmt.Sprintf("homeassistant/sensor/%s_mic_in_use_by/config", slug), textDiscovery)

	publish(client, availTopic, "online", true)
	log.Printf("Discovery published, polling every %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		sessions := getActiveMicrophoneSessions()

		names := make([]string, len(sessions))
		for i, s := range sessions {
			names[i] = filepath.Base(s)
		}

		if len(names) > 0 {
			publish(client, binaryStateTopic, "ON", false)
			publish(client, textStateTopic, strings.Join(names, ", "), false)
			log.Printf("Mic active: %s", strings.Join(names, ", "))
		} else {
			publish(client, binaryStateTopic, "OFF", false)
			publish(client, textStateTopic, "", false)
			log.Print("Mic idle")
		}

		<-ticker.C
	}
}

func publish(client mqtt.Client, topic, payload string, retained bool) {
	token := client.Publish(topic, 1, retained, payload)
	token.Wait()
}

func publishJSON(client mqtt.Client, topic string, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	token := client.Publish(topic, 1, true, data)
	token.Wait()
}
