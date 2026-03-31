//go:build !windows

package main

import "os"

func isHeadless() bool {
	// On non-Windows, check if stdout is a terminal.
	// If not (e.g. launched from a .app bundle or launchd), log to file.
	fi, err := os.Stdout.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice == 0
}
