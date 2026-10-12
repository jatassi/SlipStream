//go:build !darwin

package loopback

import (
	"context"
	"runtime"
)

const nothingNeeded = "nothing is needed on " + runtime.GOOS + ", where all of 127.0.0.0/8 already answers"

func Setup(context.Context) (string, error) { return nothingNeeded, nil }

func Remove(context.Context) (string, error) { return nothingNeeded, nil }
