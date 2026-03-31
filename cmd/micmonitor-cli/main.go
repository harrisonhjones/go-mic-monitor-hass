package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"miccheck/internal/monitor"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	purple = "\033[35m"
	gray   = "\033[90m"
	bold   = "\033[1m"
	clear  = "\033[2K\r"
)

func main() {
	_ = godotenv.Load()

	cfg, err := monitor.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	fmt.Printf("%s%s● Mic Monitor%s starting (polling every %s)...\n", bold, gray, reset, cfg.PollInterval)

	stop := make(chan struct{})

	// Handle Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Printf("\n%s%s● Shutting down...%s\n", bold, gray, reset)
		close(stop)
	}()

	err = monitor.Run(cfg, func(s monitor.StatusUpdate) {
		printStatus(s)
	}, stop)

	if err != nil {
		log.Fatal(err)
	}
}

func printStatus(s monitor.StatusUpdate) {
	if !s.Connected {
		fmt.Printf("%s%s%s● MQTT Disconnected%s\n", clear, bold, red, reset)
		return
	}

	if s.Active {
		sessions := strings.Join(s.Sessions, ", ")
		fmt.Printf("%s%s%s● Mic Active%s %s%s%s\n", clear, bold, purple, reset, yellow, sessions, reset)
	} else {
		fmt.Printf("%s%s%s● Mic Idle%s\n", clear, bold, green, reset)
	}
}
