//go:build linux

package main

import (
	"log/slog"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform/linux"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform/mcu"
)

func newHardware(log *slog.Logger, mcuDevice string) (platform.Platform, error) {
	l := linux.New(log, "/shares")
	if mcuDevice != "" {
		c, err := mcu.Open(mcuDevice, log)
		if err != nil {
			return nil, err
		}
		l.AttachMCU(c)
		log.Warn("experimental MCU driver enabled", "device", mcuDevice)
	}
	return l, nil
}
