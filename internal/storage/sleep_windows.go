package storage

import (
	"time"

	"golang.org/x/sys/windows"
)

func sleepOS(d time.Duration) {
	windows.SleepEx(uint32((d+time.Millisecond-1)/time.Millisecond), false) //nolint:gosec // G115: xSleep's microseconds are an int32
}
