package world

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestPartition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Minute)
		_ = t.Context()
	})
}
