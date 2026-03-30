// miccheck outputs the current microphone session status as JSON and exits.
// Output format: {"active": true, "sessions": ["app1.exe", "app2.exe"]}
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var debugMode bool

type micStatus struct {
	Active   bool     `json:"active"`
	Sessions []string `json:"sessions"`
}

func main() {
	flag.BoolVar(&debugMode, "debug", false, "Print detailed debug info to stderr")
	flag.Parse()

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
