//go:build !windows

package monitor

import "os/exec"

func hideMiccheckWindow(cmd *exec.Cmd) {
	// No-op on non-Windows platforms
}
