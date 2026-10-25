package unit3d

import "fmt"

func rejected(seconds int) string {
	return fmt.Sprintf("You last announced %d s ago. Please respect the min interval.", seconds)
}

const ours = "Please wait"
