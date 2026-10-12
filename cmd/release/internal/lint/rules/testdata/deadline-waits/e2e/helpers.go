package e2e

import (
	"testing"
	stdtime "time"
)

var shared *testing.T

var tick = stdtime.NewTicker

func poll() {
	_ = shared.Context()
	_ = stdtime.Tick(stdtime.Second)
}
