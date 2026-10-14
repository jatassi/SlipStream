package worldtest

import (
	"errors"

	"golang.org/x/sys/windows"
)

func heldBriefly(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_ACCESS_DENIED)
}
