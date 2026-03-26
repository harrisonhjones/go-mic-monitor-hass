package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	CLSID_MMDeviceEnumerator = guid{0xBCDE0395, 0xE52F, 0x467C, [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	IID_IMMDeviceEnumerator  = guid{0xA95664D2, 0x9614, 0x4F35, [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	IID_IAudioSessionMgr2    = guid{0x77AA99A0, 0x1BD6, 0x484F, [8]byte{0x8B, 0xC7, 0x2C, 0x65, 0x4C, 0x9A, 0x9B, 0x6F}}
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

const (
	eCapture                 = 1
	eConsole                 = 0
	CLSCTX_ALL               = 0x17
	COINIT_APARTMENTTHREADED = 0x2
	AudioSessionStateActive  = 1
)

var (
	ole32            = syscall.NewLazyDLL("ole32.dll")
	coInitializeEx   = ole32.NewProc("CoInitializeEx")
	coCreateInstance = ole32.NewProc("CoCreateInstance")
	coUninitialize   = ole32.NewProc("CoUninitialize")

	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	openProcess                = kernel32.NewProc("OpenProcess")
	closeHandle                = kernel32.NewProc("CloseHandle")
	queryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
)

func getProcessName(pid uint32) string {
	if pid == 0 {
		return "System"
	}
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	handle, _, _ := openProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if handle == 0 {
		return fmt.Sprintf("PID %d (access denied)", pid)
	}
	defer closeHandle.Call(handle)

	var buf [260]uint16
	size := uint32(len(buf))
	ret, _, _ := queryFullProcessImageNameW.Call(handle, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return fmt.Sprintf("PID %d", pid)
	}
	return syscall.UTF16ToString(buf[:size])
}

// getActiveMicrophoneSessions returns the process names of all applications
// currently holding an active capture session on the default microphone.
func getActiveMicrophoneSessions() []string {
	hr, _, _ := coInitializeEx.Call(0, uintptr(COINIT_APARTMENTTHREADED))
	if hr != 0 && hr != 1 {
		return nil
	}
	defer coUninitialize.Call()

	var enumerator *iMMDeviceEnumerator
	hr, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_MMDeviceEnumerator)),
		0,
		uintptr(CLSCTX_ALL),
		uintptr(unsafe.Pointer(&IID_IMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)),
	)
	if hr != 0 {
		return nil
	}
	defer enumerator.Release()

	var device *iMMDevice
	if enumerator.GetDefaultAudioEndpoint(eCapture, eConsole, &device) != 0 {
		return nil
	}
	defer device.Release()

	var sessionMgr *iAudioSessionManager2
	if device.Activate(&IID_IAudioSessionMgr2, CLSCTX_ALL, 0, unsafe.Pointer(&sessionMgr)) != 0 {
		return nil
	}
	defer sessionMgr.Release()

	var sessionEnum *iAudioSessionEnumerator
	if sessionMgr.GetSessionEnumerator(&sessionEnum) != 0 {
		return nil
	}
	defer sessionEnum.Release()

	var count int32
	sessionEnum.GetCount(&count)

	var active []string
	for i := int32(0); i < count; i++ {
		var session *iAudioSessionControl
		if sessionEnum.GetSession(i, &session) != 0 {
			continue
		}

		var state uint32
		session.GetState(&state)

		var session2 *iAudioSessionControl2
		qiResult := session.QueryInterface(&IID_IAudioSessionControl2, unsafe.Pointer(&session2))

		if state == AudioSessionStateActive {
			processName := "unknown"
			if qiResult == 0 {
				var pid uint32
				session2.GetProcessId(&pid)
				processName = getProcessName(pid)
			}
			active = append(active, processName)
		}

		if qiResult == 0 {
			session2.Release()
		}
		session.Release()
	}

	return active
}
