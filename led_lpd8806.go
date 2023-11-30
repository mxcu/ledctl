package ledctl

import (
	"fmt"

	rpi "github.com/Jon-Bright/ledctl/rpi"
)

// LPD8806 controls an LPD8806 LED strip.
type LPD8806 struct {
	rp        *rpi.RPi
	dev       Device
	pixels    []byte
	buffer    []byte
	numColors int
	numPixels int
	g         int
	r         int
	b         int
	w         int
}

// NewLPD8806 creates a new LPD8806 LED strip controller. Here are the
// parameters:
//
//   - dev is the SPI device to use. Usually, this is "/dev/spidev0.0".
//   - numPixels is the number of pixels in the strip.
//   - numColors is the number of colors per pixel. This is usually 3, but some
//     strips have a white component as well.
//   - spiSpeed is the speed to use for the SPI. This is usually 12000000.
//   - order is the color order of the pixels. This is usually GRB, but some
//     strips have different orders.
func NewLPD8806(dev Device, numPixels int, numColors int, spiSpeed uint32, order ColorOrder) (*LPD8806, error) {
	numReset := (numPixels + 31) / 32
	val := make([]byte, numPixels*numColors+numReset)
	offsets := offsets[order]

	rp, err := rpi.NewRPi()
	if err != nil {
		return nil, fmt.Errorf("couldn't make RPi: %v", err)
	}

	la := LPD8806{
		rp:        rp,
		dev:       dev,
		pixels:    val[:numPixels*numColors],
		buffer:    val,
		numColors: numColors,
		numPixels: numPixels,
		g:         offsets[0],
		r:         offsets[1],
		b:         offsets[2],
		w:         offsets[3],
	}

	if spiSpeed != 0 {
		err := rp.SetSPISpeed(la.dev.Fd(), spiSpeed)
		if err != nil {
			return nil, fmt.Errorf("couldn't set SPI speed: %v", err)
		}
	}

	firstReset := make([]byte, numReset)
	_, err = dev.Write(firstReset)
	if err != nil {
		return nil, fmt.Errorf("couldn't reset: %v", err)
	}
	return &la, nil
}

// RPi returns the RPi object used to control the SPI.
func (la *LPD8806) RPi() *rpi.RPi {
	return la.rp
}

// MaxLEDsPerChannel returns the maximum number of LEDs per channel.
func (la *LPD8806) MaxLEDsPerChannel() int {
	return 127
}

// Flush flushes the pixels to the LED strip.
func (la *LPD8806) Flush() error {
	_, err := la.dev.Write(la.buffer)
	return err
}

// RGBWAt returns the RGBW pixel at the given index.
// If numColors is 3, then white is an undefined value.
func (la *LPD8806) RGBWAt(i int) RGBW {
	o := i * la.numColors
	return RGBW{
		la.pixels[o+la.r] & 0x7F,
		la.pixels[o+la.g] & 0x7F,
		la.pixels[o+la.b] & 0x7F,
		la.pixels[o+la.w] & 0x7F,
	}
}

// SetRGBWAt sets the RGBW pixel at the given index to the given value.
// If numColors is 3, then white is an undefined value.
func (la *LPD8806) SetRGBWAt(i int, rgbw RGBW) {
	o := i * la.numColors
	la.pixels[o+la.r] = 0x80 | rgbw.R
	la.pixels[o+la.g] = 0x80 | rgbw.G
	la.pixels[o+la.b] = 0x80 | rgbw.B
	la.pixels[o+la.w] = 0x80 | rgbw.W
}

// SetRGBWs sets the RGBW pixels to the given values.
// If numColors is 3, then white is an undefined value.
func (la *LPD8806) SetRGBWs(pixels []RGBW) {
	if la.numColors != 4 {
		panic("SetRGBWs called on WS281x with numColors != 4")
	}
	if len(pixels) != la.numPixels {
		panic("SetRGBWs called with wrong number of pixels")
	}

	a := 0
	for i := 0; i < len(la.pixels); i += 4 {
		la.pixels[a+la.r] = 0x80 | pixels[i].R
		la.pixels[a+la.g] = 0x80 | pixels[i].G
		la.pixels[a+la.b] = 0x80 | pixels[i].B
		la.pixels[a+la.w] = 0x80 | pixels[i].W
		a++
	}
}

// RGBAt returns the RGB pixel at the given index.
func (la *LPD8806) RGBAt(i int) RGB {
	o := i * la.numColors
	return RGB{
		la.pixels[o+la.r] & 0x7F,
		la.pixels[o+la.g] & 0x7F,
		la.pixels[o+la.b] & 0x7F,
	}
}

// SetRGBAt sets the RGB pixel at the given index to the given value.
func (la *LPD8806) SetRGBAt(i int, rgb RGB) {
	o := i * la.numColors
	la.pixels[o+la.r] = 0x80 | rgb.R
	la.pixels[o+la.g] = 0x80 | rgb.G
	la.pixels[o+la.b] = 0x80 | rgb.B
}

// SetRGBs sets the RGB pixels to the given values.
func (la *LPD8806) SetRGBs(pixels []RGB) {
	if la.numColors != 3 {
		panic("SetRGBs called on RGBW strip")
	}
	if len(pixels) != la.numPixels {
		panic("SetRGBs called with wrong number of pixels")
	}

	a := 0
	for i := 0; i < len(la.pixels); i += 3 {
		la.pixels[i+la.r] = 0x80 | pixels[a].R
		la.pixels[i+la.g] = 0x80 | pixels[a].G
		la.pixels[i+la.b] = 0x80 | pixels[a].B
		a++
	}
}
