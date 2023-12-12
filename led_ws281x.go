package ledctl

import (
	"fmt"

	rpi "libdb.so/ledctl/rpi"
)

// WS281x controls a WS281x LED strip.
type WS281x struct {
	pixDMAUint []uint32
	pixDMA     *rpi.DMABuf
	rp         *rpi.RPi
	pixels     []byte
	numPixels  int
	numColors  int
	g          int
	r          int
	b          int
	w          int
}

const ledReset_us = 55

// WS281xConfig is the configuration for a WS281x LED strip.
type WS281xConfig struct {
	// NumPixels is the number of pixels in the strip.
	NumPixels int
	// ColorOrder is the color order of the pixels. This is usually GRB, but
	// some strips have different orders.
	ColorOrder ColorOrder
	// ColorModel is the color model of the pixels.
	ColorModel ColorModel
	// PWMFrequency is the frequency to use for the PWM. This is usually
	// 800000.
	PWMFrequency uint
	// DMAChannel is the DMA channel to use. This is usually 10, but it depends
	// on which Pi you're using. BE CAREFUL, this may damage your Pi if you get
	// it wrong.
	DMAChannel int
	// GPIOPins is a list of GPIO pins to use for the PWM. Usually, this is a
	// single-item list containing the pin that you're using for the data line.
	GPIOPins []int
}

// NewWS281x creates a new WS281x LED strip controller.
func NewWS281x(config WS281xConfig) (*WS281x, error) {
	rp, err := rpi.NewRPi()
	if err != nil {
		return nil, fmt.Errorf("couldn't init RPi: %v", err)
	}

	offsets := offsets[config.ColorOrder]
	wa := WS281x{
		numPixels: config.NumPixels,
		numColors: config.ColorModel.NumColors(),
		pixels:    make([]byte, config.NumPixels*config.ColorModel.NumColors()),
		rp:        rp,
		g:         offsets[0],
		r:         offsets[1],
		b:         offsets[2],
		w:         offsets[3],
	}

	bytes := wa.pwmByteCount(config.PWMFrequency)
	wa.pixDMA, err = rp.GetDMABuf(bytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't get DMA buffer: %v", err)
	}

	wa.pixDMAUint = wa.pixDMA.Uint32Slice()
	err = rp.InitDMA(config.DMAChannel)
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init registers: %v", err)
	}

	err = rp.InitGPIO()
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init GPIO: %v", err)
	}

	err = rp.InitPWM(config.PWMFrequency, wa.pixDMA, bytes, config.GPIOPins)
	if err != nil {
		rp.FreeDMABuf(wa.pixDMA) // Ignore error
		return nil, fmt.Errorf("couldn't init PWM: %v", err)
	}

	return &wa, nil
}

// Close closes the WS281x LED strip controller.
func (ws *WS281x) Close() error {
	ws.rp.StopPWM()

	if err := ws.rp.FreeDMABuf(ws.pixDMA); err != nil {
		return fmt.Errorf("couldn't free DMA buffer: %v", err)
	}

	return nil
}

// pwmByteCount calculates the number of bytes needed to store the data for PWM
// to send - three bits per WS281x bit, plus enough bits to provide an
// appropriate reset time afterwards at the given frequency. It returns that
// byte count.
func (ws *WS281x) pwmByteCount(freq uint) uint {
	// Every bit transmitted needs 3 bits of buffer, because bits are transmitted as
	// ‾|__ (0) or ‾‾|_ (1). Each color of each pixel needs 8 "real" bits.
	bits := uint(3 * ws.numColors * ws.numPixels * 8)

	// freq is typically 800kHz, so for LED_RESET_US=55 us, this gives us
	// ((55 * (800000 * 3)) / 1000000
	// ((55 * 2400000) / 1000000
	// 132000000 / 1000000
	// 132
	// Taking this the other way, 132 bits of buffer is 132/3=44 "real" bits.
	// With each "real" bit taking 1/800000th of a second, this will take
	// 44/800000ths of a second, which is 0.000055s - 55 us.
	bits += ((ledReset_us * (freq * 3)) / 1000000)

	// This isn't a PDP-11, so there are 8 bits in a byte
	bytes := bits / 8

	// Round up to next uint32
	bytes -= bytes % 4
	bytes += 4

	bytes *= rpi.RPI_PWM_CHANNELS

	return bytes
}

// RPi returns the RPi object that this WS281x is using.
func (ws *WS281x) RPi() *rpi.RPi {
	return ws.rp
}

// MaxLEDsPerChannel returns the maximum number of LEDs that can be controlled
// per channel.
func (ws *WS281x) MaxLEDsPerChannel() int {
	return 255
}

// RGBWAt returns the RGBW pixel at the given index.
// If numColors is 3, then white is an undefined value.
func (ws *WS281x) RGBWAt(i int) RGBW {
	o := i * ws.numColors
	return RGBW{
		ws.pixels[o+ws.r],
		ws.pixels[o+ws.g],
		ws.pixels[o+ws.b],
		ws.pixels[o+ws.w],
	}
}

// SetRGBWAt sets the RGBW pixel at the given index to the given value.
// If numColors is 3, then white is an undefined value.
func (ws *WS281x) SetRGBWAt(i int, rgbw RGBW) {
	o := i * ws.numColors
	ws.pixels[o+ws.r] = rgbw.R
	ws.pixels[o+ws.g] = rgbw.G
	ws.pixels[o+ws.b] = rgbw.B
	ws.pixels[o+ws.w] = rgbw.W
}

// SetRGBWs sets the RGBW pixels to the given values.
// If numColors is 3, then white is an undefined value.
func (ws *WS281x) SetRGBWs(pixels []RGBW) {
	if ws.numColors != 4 {
		panic("SetRGBWs called on WS281x with numColors != 4")
	}
	if len(pixels) != ws.numPixels {
		panic("SetRGBWs called with wrong number of pixels")
	}

	a := 0
	for i := 0; i < len(ws.pixels); i += 4 {
		ws.pixels[a+ws.r] = pixels[i].R
		ws.pixels[a+ws.g] = pixels[i].G
		ws.pixels[a+ws.b] = pixels[i].B
		ws.pixels[a+ws.w] = pixels[i].W
		a++
	}
}

// RGBAt returns the RGB pixel at the given index.
func (ws *WS281x) RGBAt(i int) RGB {
	o := i * ws.numColors
	return RGB{
		ws.pixels[o+ws.r],
		ws.pixels[o+ws.g],
		ws.pixels[o+ws.b],
	}
}

// SetRGBAt sets the RGB pixel at the given index to the given value.
func (ws *WS281x) SetRGBAt(i int, rgb RGB) {
	o := i * ws.numColors
	ws.pixels[o+ws.r] = rgb.R
	ws.pixels[o+ws.g] = rgb.G
	ws.pixels[o+ws.b] = rgb.B
}

// SetRGBs sets the RGB pixels to the given values.
func (ws *WS281x) SetRGBs(pixels []RGB) {
	if ws.numColors != 3 {
		panic("SetRGBs called on RGBW strip")
	}
	if len(pixels) != ws.numPixels {
		panic("SetRGBs called with wrong number of pixels")
	}

	a := 0
	for i := 0; i < len(ws.pixels); i += 3 {
		ws.pixels[i+ws.r] = pixels[a].R
		ws.pixels[i+ws.g] = pixels[a].G
		ws.pixels[i+ws.b] = pixels[a].B
		a++
	}
}

const (
	symbolHigh = 0x6 // 1 1 0
	symbolLow  = 0x4 // 1 0 0
)

// Flush flushes the current pixel buffer to the LEDs.
func (ws *WS281x) Flush() error {
	// We need to wait for DMA to be done before we start touching the buffer it's outputting
	err := ws.rp.WaitForDMAEnd()
	if err != nil {
		return fmt.Errorf("pre-DMA wait failed: %v", err)
	}

	// TODO: channels, do properly - this just assumes both channels show the same thing
	for c := 0; c < 2; c++ {
		rpPos := c
		bitPos := 31
		for i := 0; i < ws.numPixels; i++ {
			for j := 0; j < ws.numColors; j++ {
				for k := 7; k >= 0; k-- {
					symbol := symbolLow
					if (ws.pixels[i*ws.numColors+j] & (1 << uint(k))) != 0 {
						symbol = symbolHigh
					}
					for l := 2; l >= 0; l-- {
						ws.pixDMAUint[rpPos] &= ^(1 << uint(bitPos))
						if (symbol & (1 << uint(l))) != 0 {
							ws.pixDMAUint[rpPos] |= 1 << uint(bitPos)
						}
						bitPos--
						if bitPos < 0 {
							rpPos += 2
							bitPos = 31
						}
					}
				}
			}
		}
	}
	ws.rp.StartDMA(ws.pixDMA)
	return nil
}
