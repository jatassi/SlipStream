package language

import (
	"errors"
	"os"
	"time"
)

var ErrUnknown = errors.New("unknown language")

func Missing(err error) bool { return errors.Is(err, os.ErrNotExist) }

func Hold(d time.Duration) time.Duration { return d * time.Second }
