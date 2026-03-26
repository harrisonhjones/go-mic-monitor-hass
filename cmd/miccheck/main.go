// miccheck outputs the current microphone session status as JSON and exits.
// Output format: {"active": true, "sessions": ["app1.exe", "app2.exe"]}
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type micStatus struct {
	Active   bool     `json:"active"`
	Sessions []string `json:"sessions"`
}

func main() {
	raw := getActiveMicrophoneSessions()

	names := make([]string, len(raw))
	for i, s := range raw {
		names[i] = filepath.Base(s)
	}

	status := micStatus{
		Active:   len(names) > 0,
		Sessions: names,
	}

	data, err := json.Marshal(status)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
