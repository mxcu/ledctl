package ledctl

import (
	"fmt"
	"io"
	"os"
)

// ColorOrder is an enumeration of the possible color orders for the color
// pixels.
type ColorOrder int

const (
	GRBOrder ColorOrder = iota
	BRGOrder
	BGROrder
	GBROrder
	RGBOrder
	RBGOrder
)

// StringToOrder is a map from string representations of the color order to
// the ColorOrder.
var StringToOrder = map[string]ColorOrder{
	"GRB": GRBOrder,
	"BRG": BRGOrder,
	"BGR": BGROrder,
	"GBR": GBROrder,
	"RGB": RGBOrder,
	"RBG": RBGOrder,
}

var offsets = map[ColorOrder][]int{
	GRBOrder: {0, 1, 2, -1},
	BRGOrder: {2, 1, 0, -1},
	BGROrder: {1, 2, 0, -1},
	GBROrder: {0, 2, 1, -1},
	RGBOrder: {1, 0, 2, -1},
	RBGOrder: {2, 0, 1, -1},
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// RGBW represents a pixel with red, green, blue, and white components.
type RGBW struct {
	R uint8
	G uint8
	B uint8
	W uint8
}

// String returns a string representation of the pixel in the form #rrggbbww.
func (p RGBW) String() string {
	return fmt.Sprintf("#%02x%02x%02x%02x", p.R, p.G, p.B, p.W)
}

// ToUint32 returns the pixel as a uint32 in the form 0xrrggbbww.
func (p RGBW) ToUint32() uint32 {
	return uint32(p.R)<<24 | uint32(p.G)<<16 | uint32(p.B)<<8 | uint32(p.W)
}

// RGB represents a pixel with red, green, and blue components.
type RGB struct {
	R uint8
	G uint8
	B uint8
}

// String returns a string representation of the pixel in the form #rrggbb.
func (p RGB) String() string {
	return fmt.Sprintf("#%02x%02x%02x", p.R, p.G, p.B)
}

// ToUint32 returns the pixel as a uint32 in the form 0xrrggbb.
func (p RGB) ToUint32() uint32 {
	return uint32(p.R)<<16 | uint32(p.G)<<8 | uint32(p.B)
}

// Device extends io.Writer with an Fd method that returns the file descriptor
// of the device.
type Device interface {
	io.Writer
	Fd() uintptr
}

var _ Device = (*os.File)(nil)
