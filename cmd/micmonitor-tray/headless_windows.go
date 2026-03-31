package main

import "syscall"

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
)

func isHeadless() bool {
	hwnd, _, _ := getConsoleWindow.Call()
	return hwnd == 0
}
