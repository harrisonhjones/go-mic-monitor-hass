package main

import (
	"syscall"
	"unsafe"
)

// iMMDeviceEnumerator
type iMMDeviceEnumerator struct {
	vtbl *iMMDeviceEnumeratorVtbl
}

type iMMDeviceEnumeratorVtbl struct {
	QueryInterface                 uintptr
	AddRef                         uintptr
	Release                        uintptr
	EnumAudioEndpoints             uintptr
	GetDefaultAudioEndpoint        uintptr
	GetDevice                      uintptr
	RegisterEndpointNotification   uintptr
	UnregisterEndpointNotification uintptr
}

func (e *iMMDeviceEnumerator) Release() {
	syscall.SyscallN(e.vtbl.Release, uintptr(unsafe.Pointer(e)))
}

func (e *iMMDeviceEnumerator) GetDefaultAudioEndpoint(dataFlow, role uint32, device **iMMDevice) uintptr {
	r, _, _ := syscall.SyscallN(
		e.vtbl.GetDefaultAudioEndpoint,
		uintptr(unsafe.Pointer(e)),
		uintptr(dataFlow),
		uintptr(role),
		uintptr(unsafe.Pointer(device)),
	)
	return r
}

// iMMDevice
type iMMDevice struct {
	vtbl *iMMDeviceVtbl
}

type iMMDeviceVtbl struct {
	QueryInterface    uintptr
	AddRef            uintptr
	Release           uintptr
	Activate          uintptr
	OpenPropertyStore uintptr
	GetId             uintptr
	GetState          uintptr
}

func (d *iMMDevice) Release() {
	syscall.SyscallN(d.vtbl.Release, uintptr(unsafe.Pointer(d)))
}

func (d *iMMDevice) Activate(iid *guid, clsCtx uint32, params uintptr, out unsafe.Pointer) uintptr {
	r, _, _ := syscall.SyscallN(
		d.vtbl.Activate,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(clsCtx),
		params,
		uintptr(out),
	)
	return r
}
