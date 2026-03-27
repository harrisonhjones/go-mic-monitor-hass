package monitor

import (
	"os/exec"
	"syscall"
)

func hideMiccheckWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
