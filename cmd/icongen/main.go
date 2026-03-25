// Generates embedded ICO files for the system tray.
// Run: go run ./cmd/icongen
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	generateIcon("icons/mic_idle.ico", color.RGBA{0x4C, 0xAF, 0x50, 0xFF})   // green
	generateIcon("icons/mic_active.ico", color.RGBA{0x9C, 0x27, 0xB0, 0xFF}) // purple
	generateIcon("icons/mic_error.ico", color.RGBA{0xF4, 0x43, 0x36, 0xFF})  // red
}

func generateIcon(path string, col color.RGBA) {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw a filled circle as background
	cx, cy, r := float64(size/2), float64(size/2), float64(size/2-2)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, col)
			}
		}
	}

	// Draw a simple mic shape in white (rectangle + rounded top)
	white := color.RGBA{255, 255, 255, 255}
	micRect := image.Rect(26, 14, 38, 38)
	draw.Draw(img, micRect, &image.Uniform{white}, image.Point{}, draw.Over)

	// Rounded top of mic
	mcx, mcy, mr := 32.0, 14.0, 6.0
	for y := 8; y < 20; y++ {
		for x := 26; x < 38; x++ {
			dx := float64(x) - mcx
			dy := float64(y) - mcy
			if dx*dx+dy*dy <= mr*mr {
				img.Set(x, y, white)
			}
		}
	}

	// Mic stand (vertical line)
	standRect := image.Rect(31, 38, 33, 48)
	draw.Draw(img, standRect, &image.Uniform{white}, image.Point{}, draw.Over)

	// Mic base (horizontal line)
	baseRect := image.Rect(26, 47, 38, 49)
	draw.Draw(img, baseRect, &image.Uniform{white}, image.Point{}, draw.Over)

	// Encode as ICO (single PNG-based ICO entry)
	var pngBuf bytes.Buffer
	png.Encode(&pngBuf, img)
	pngData := pngBuf.Bytes()

	var ico bytes.Buffer
	// ICONDIR header
	binary.Write(&ico, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&ico, binary.LittleEndian, uint16(1)) // count
	// ICONDIRENTRY
	ico.WriteByte(byte(size))                                     // width
	ico.WriteByte(byte(size))                                     // height
	ico.WriteByte(0)                                              // color palette
	ico.WriteByte(0)                                              // reserved
	binary.Write(&ico, binary.LittleEndian, uint16(1))            // color planes
	binary.Write(&ico, binary.LittleEndian, uint16(32))           // bits per pixel
	binary.Write(&ico, binary.LittleEndian, uint32(len(pngData))) // size of image data
	binary.Write(&ico, binary.LittleEndian, uint32(6+16))         // offset to image data (header=6, entry=16)
	ico.Write(pngData)

	os.MkdirAll("icons", 0755)
	os.WriteFile(path, ico.Bytes(), 0644)
}
