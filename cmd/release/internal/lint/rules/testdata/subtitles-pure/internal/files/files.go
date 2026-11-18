package files

import (
	"os"
	"time"
)

func Stamp(path string) (time.Time, error) {
	_, err := os.Open(path)
	return time.Now(), err
}
