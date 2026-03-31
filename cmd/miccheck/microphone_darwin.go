package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"

	"github.com/ebitengine/purego"
)

// CoreAudio constants
const (
	kAudioObjectSystemObject = 1

	// Property selectors (FourCC as uint32)
	kAudioHardwarePropertyDevices                = 0x64657623 // 'dev#'
	kAudioDevicePropertyStreamConfiguration      = 0x736C6179 // 'slay'
	kAudioDevicePropertyDeviceIsRunningSomewhere = 0x676F6E65 // 'gone'
	kAudioObjectPropertyName                     = 0x6C6E616D // 'lnam'

	// Scopes
	kAudioObjectPropertyScopeGlobal = 0x676C6F62 // 'glob'
	kAudioObjectPropertyScopeInput  = 0x696E7074 // 'inpt'

	// Element
	kAudioObjectPropertyElementMain = 0
)

// AudioObjectPropertyAddress matches the C struct layout.
type audioObjectPropertyAddress struct {
	Selector uint32
	Scope    uint32
	Element  uint32
}

var (
	audioObjectGetPropertyDataSize func(
		objectID uint32,
		address *audioObjectPropertyAddress,
		qualifierDataSize uint32,
		qualifierData uintptr,
		dataSize *uint32,
	) int32

	audioObjectGetPropertyData func(
		objectID uint32,
		address *audioObjectPropertyAddress,
		qualifierDataSize uint32,
		qualifierData uintptr,
		dataSize *uint32,
		data uintptr,
	) int32

	coreAudioLoaded bool
)

func init() {
	lib, err := purego.Dlopen("/System/Library/Frameworks/CoreAudio.framework/CoreAudio", purego.RTLD_LAZY)
	if err != nil {
		debugf("ERROR: failed to load CoreAudio framework: %v", err)
		return
	}
	purego.RegisterLibFunc(&audioObjectGetPropertyDataSize, lib, "AudioObjectGetPropertyDataSize")
	purego.RegisterLibFunc(&audioObjectGetPropertyData, lib, "AudioObjectGetPropertyData")
	coreAudioLoaded = true
	debugf("CoreAudio framework loaded successfully")
}

func debugf(format string, args ...any) {
	if debugMode {
		fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
	}
}

func getActiveMicrophoneSessions() []string {
	if !coreAudioLoaded {
		debugf("CoreAudio not loaded, returning nil")
		return nil
	}

	// Get all audio devices
	devicesAddr := audioObjectPropertyAddress{
		Selector: kAudioHardwarePropertyDevices,
		Scope:    kAudioObjectPropertyScopeGlobal,
		Element:  kAudioObjectPropertyElementMain,
	}

	var dataSize uint32
	status := audioObjectGetPropertyDataSize(kAudioObjectSystemObject, &devicesAddr, 0, 0, &dataSize)
	debugf("GetPropertyDataSize(devices): status=%d, dataSize=%d", status, dataSize)
	if status != 0 || dataSize == 0 {
		debugf("No devices found or error")
		return nil
	}

	deviceCount := dataSize / 4
	devices := make([]uint32, deviceCount)
	status = audioObjectGetPropertyData(
		kAudioObjectSystemObject, &devicesAddr, 0, 0, &dataSize,
		uintptr(unsafe.Pointer(&devices[0])),
	)
	debugf("GetPropertyData(devices): status=%d, deviceCount=%d, deviceIDs=%v", status, deviceCount, devices)
	if status != 0 {
		return nil
	}

	for _, deviceID := range devices {
		name := getDeviceName(deviceID)
		hasInput := hasInputChannels(deviceID)
		debugf("Device %d (%s): hasInput=%v", deviceID, name, hasInput)

		if !hasInput {
			continue
		}

		running := isDeviceRunning(deviceID)
		debugf("Device %d (%s): isRunning=%v", deviceID, name, running)

		if running {
			return []string{"microphone"}
		}
	}

	debugf("No active input devices found")
	return nil
}

func getDeviceName(deviceID uint32) string {
	addr := audioObjectPropertyAddress{
		Selector: kAudioObjectPropertyName,
		Scope:    kAudioObjectPropertyScopeGlobal,
		Element:  kAudioObjectPropertyElementMain,
	}

	// kAudioObjectPropertyName returns a CFStringRef. Reading it properly
	// requires CoreFoundation calls. For debug purposes, just return the ID.
	var dataSize uint32
	status := audioObjectGetPropertyDataSize(deviceID, &addr, 0, 0, &dataSize)
	if status != 0 {
		return fmt.Sprintf("id:%d", deviceID)
	}
	return fmt.Sprintf("id:%d (nameSize=%d)", deviceID, dataSize)
}

func hasInputChannels(deviceID uint32) bool {
	addr := audioObjectPropertyAddress{
		Selector: kAudioDevicePropertyStreamConfiguration,
		Scope:    kAudioObjectPropertyScopeInput,
		Element:  kAudioObjectPropertyElementMain,
	}

	var bufSize uint32
	status := audioObjectGetPropertyDataSize(deviceID, &addr, 0, 0, &bufSize)
	debugf("  Device %d StreamConfig size: status=%d, bufSize=%d", deviceID, status, bufSize)
	if status != 0 || bufSize == 0 {
		return false
	}

	buf := make([]byte, bufSize)
	status = audioObjectGetPropertyData(deviceID, &addr, 0, 0, &bufSize, uintptr(unsafe.Pointer(&buf[0])))
	debugf("  Device %d StreamConfig data: status=%d, bytesReturned=%d", deviceID, status, bufSize)
	if status != 0 {
		return false
	}

	if len(buf) < 4 {
		debugf("  Device %d: buffer too small (%d bytes)", deviceID, len(buf))
		return false
	}

	numBuffers := binary.LittleEndian.Uint32(buf[0:4])
	debugf("  Device %d: numBuffers=%d", deviceID, numBuffers)

	ptrSize := unsafe.Sizeof(uintptr(0))
	bufferStride := 4 + 4 + int(ptrSize) // mNumberChannels + mDataByteSize + mData
	offset := 4

	for i := uint32(0); i < numBuffers; i++ {
		if offset+4 > len(buf) {
			debugf("  Device %d: buffer overflow at buffer %d (offset=%d, len=%d)", deviceID, i, offset, len(buf))
			break
		}
		channels := binary.LittleEndian.Uint32(buf[offset : offset+4])
		debugf("  Device %d: buffer[%d] channels=%d", deviceID, i, channels)
		if channels > 0 {
			return true
		}
		offset += bufferStride
	}

	return false
}

func isDeviceRunning(deviceID uint32) bool {
	addr := audioObjectPropertyAddress{
		Selector: kAudioDevicePropertyDeviceIsRunningSomewhere,
		Scope:    kAudioObjectPropertyScopeGlobal,
		Element:  kAudioObjectPropertyElementMain,
	}

	var isRunning uint32
	size := uint32(4)
	status := audioObjectGetPropertyData(deviceID, &addr, 0, 0, &size, uintptr(unsafe.Pointer(&isRunning)))
	debugf("  Device %d IsRunningSomewhere: status=%d, isRunning=%d", deviceID, status, isRunning)
	return status == 0 && isRunning != 0
}
