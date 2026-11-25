//go:build !linux

package main

import (
	"errors"
	"log/slog"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

func newHardware(*slog.Logger, string) (platform.Platform, error) {
	return nil, errors.New("real hardware is only supported on Linux; use -simulate")
}
