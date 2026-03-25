package main

import (
	"syscall"
	"unsafe"
)

// IAudioSessionManager2
type iAudioSessionManager2 struct {
	vtbl *iAudioSessionManager2Vtbl
}

type iAudioSessionManager2Vtbl struct {
	QueryInterface                uintptr
	AddRef                        uintptr
	Release                       uintptr
	GetAudioSessionControl        uintptr
	GetSimpleAudioVolume          uintptr
	GetSessionEnumerator          uintptr
	RegisterSessionNotification   uintptr
	UnregisterSessionNotification uintptr
	RegisterDuckNotification      uintptr
	UnregisterDuckNotification    uintptr
}

func (m *iAudioSessionManager2) Release() {
	syscall.SyscallN(m.vtbl.Release, uintptr(unsafe.Pointer(m)))
}

func (m *iAudioSessionManager2) GetSessionEnumerator(enumerator **iAudioSessionEnumerator) uintptr {
	r, _, _ := syscall.SyscallN(
		m.vtbl.GetSessionEnumerator,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(enumerator)),
	)
	return r
}

// IAudioSessionEnumerator
type iAudioSessionEnumerator struct {
	vtbl *iAudioSessionEnumeratorVtbl
}

type iAudioSessionEnumeratorVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetCount       uintptr
	GetSession     uintptr
}

func (e *iAudioSessionEnumerator) Release() {
	syscall.SyscallN(e.vtbl.Release, uintptr(unsafe.Pointer(e)))
}

func (e *iAudioSessionEnumerator) GetCount(count *int32) uintptr {
	r, _, _ := syscall.SyscallN(
		e.vtbl.GetCount,
		uintptr(unsafe.Pointer(e)),
		uintptr(unsafe.Pointer(count)),
	)
	return r
}

func (e *iAudioSessionEnumerator) GetSession(index int32, session **iAudioSessionControl) uintptr {
	r, _, _ := syscall.SyscallN(
		e.vtbl.GetSession,
		uintptr(unsafe.Pointer(e)),
		uintptr(index),
		uintptr(unsafe.Pointer(session)),
	)
	return r
}

// IAudioSessionControl
type iAudioSessionControl struct {
	vtbl *iAudioSessionControlVtbl
}

type iAudioSessionControlVtbl struct {
	QueryInterface                     uintptr
	AddRef                             uintptr
	Release                            uintptr
	GetState                           uintptr
	GetDisplayName                     uintptr
	SetDisplayName                     uintptr
	GetIconPath                        uintptr
	SetIconPath                        uintptr
	GetGroupingParam                   uintptr
	SetGroupingParam                   uintptr
	RegisterAudioSessionNotification   uintptr
	UnregisterAudioSessionNotification uintptr
}

func (s *iAudioSessionControl) Release() {
	syscall.SyscallN(s.vtbl.Release, uintptr(unsafe.Pointer(s)))
}

func (s *iAudioSessionControl) GetState(state *uint32) uintptr {
	r, _, _ := syscall.SyscallN(
		s.vtbl.GetState,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(state)),
	)
	return r
}

// IAudioSessionControl2 extends IAudioSessionControl with process info.
// We QueryInterface from IAudioSessionControl to get this.
type iAudioSessionControl2 struct {
	vtbl *iAudioSessionControl2Vtbl
}

type iAudioSessionControl2Vtbl struct {
	QueryInterface                     uintptr
	AddRef                             uintptr
	Release                            uintptr
	GetState                           uintptr
	GetDisplayName                     uintptr
	SetDisplayName                     uintptr
	GetIconPath                        uintptr
	SetIconPath                        uintptr
	GetGroupingParam                   uintptr
	SetGroupingParam                   uintptr
	RegisterAudioSessionNotification   uintptr
	UnregisterAudioSessionNotification uintptr
	GetSessionIdentifier               uintptr
	GetSessionInstanceIdentifier       uintptr
	GetProcessId                       uintptr
	IsSystemSoundsSession              uintptr
	SetDuckingPreference               uintptr
}

var IID_IAudioSessionControl2 = guid{0xBFB7FF88, 0x7239, 0x4FC9, [8]byte{0x8F, 0xA2, 0x07, 0xC9, 0x50, 0xBE, 0x9C, 0x6D}}

func (s *iAudioSessionControl) QueryInterface(iid *guid, out unsafe.Pointer) uintptr {
	r, _, _ := syscall.SyscallN(
		s.vtbl.QueryInterface,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(out),
	)
	return r
}

func (s *iAudioSessionControl2) Release() {
	syscall.SyscallN(s.vtbl.Release, uintptr(unsafe.Pointer(s)))
}

func (s *iAudioSessionControl2) GetProcessId(pid *uint32) uintptr {
	r, _, _ := syscall.SyscallN(
		s.vtbl.GetProcessId,
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(pid)),
	)
	return r
}
