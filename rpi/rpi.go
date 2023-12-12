package rpi

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	mmap "github.com/edsrzf/mmap-go"
)

type RPi struct {
	mbox     *os.File
	mboxSize uint32
	hw       *hw
	dmaBuf   mmap.MMap
	dma      *dmaT
	pwmBuf   mmap.MMap
	pwm      *pwmT
	gpioBuf  mmap.MMap
	gpio     *gpioT
	cmClkBuf mmap.MMap
	cmClk    *cmClkT
}

func NewRPi() (*RPi, error) {
	hw, err := detectHardware()
	if err != nil {
		return nil, fmt.Errorf("couldn't detect RPi hardware: %v", err)
	}
	rp := RPi{
		hw: hw,
	}
	err = rp.mboxOpen()
	if err != nil {
		return nil, fmt.Errorf("couldn't open mailbox: %v", err)
	}
	return &rp, nil
}

type hw struct {
	hwType     int
	periphBase uintptr
	vcBase     uintptr
	name       string
}

const (
	RPI_HWVER_TYPE_UNKNOWN = iota
	RPI_HWVER_TYPE_PI1
	RPI_HWVER_TYPE_PI2
	RPI_HWVER_TYPE_PI4

	PERIPH_BASE_RPI  = 0x20000000
	PERIPH_BASE_RPI2 = 0x3f000000
	PERIPH_BASE_RPI4 = 0xfe000000

	VIDEOCORE_BASE_RPI  = 0x40000000
	VIDEOCORE_BASE_RPI2 = 0xc0000000
)

// Detect which version of a Raspberry Pi we're running on
// The original rpihw.c does this in two different ways, one for ARM64 only.
// My non-64-bit RPis also support the ARM64 way, though, so this implements just that (easier) way.
func detectHardware() (*hw, error) {
	sortRasPiVariantsOnce.Do(func() {
		sort.Slice(rasPiVariants, func(i, j int) bool {
			if len(rasPiVariants[i].name) == len(rasPiVariants[j].name) {
				return rasPiVariants[i].vcBase < rasPiVariants[j].vcBase
			}
			// Put longer names first, so that we can match by prefix.
			return len(rasPiVariants[i].name) > len(rasPiVariants[j].name)
		})
	})

	modelb, err := os.ReadFile("/proc/device-tree/model")
	if err != nil {
		return nil, fmt.Errorf("couldn't open model file: %v", err)
	}
	model := string(modelb)

	for _, rp := range rasPiVariants {
		if strings.HasPrefix(model, rp.name) {
			return &rp, nil
		}
	}

	return nil, fmt.Errorf("couldn't identify Pi model %q", model)
}

var sortRasPiVariantsOnce sync.Once

// https://gist.github.com/jperkin/c37a574379ef71e339361954be96be12
var rasPiVariants = []hw{
	{
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Raspberry Pi Model B",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Raspberry Pi Compute Module",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI1,
		periphBase: PERIPH_BASE_RPI,
		vcBase:     VIDEOCORE_BASE_RPI,
		name:       "Raspberry Pi Zero W",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi 2 Model B",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi Compute Module 3",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi Compute Module 3 Plus",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi 3 Model B",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI2,
		periphBase: PERIPH_BASE_RPI2,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi 3 Model B Plus",
	},
	{
		hwType:     RPI_HWVER_TYPE_PI4,
		periphBase: PERIPH_BASE_RPI4,
		vcBase:     VIDEOCORE_BASE_RPI2,
		name:       "Raspberry Pi 4 Model B",
	},
}
