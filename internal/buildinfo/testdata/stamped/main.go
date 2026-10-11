package main

import (
	"encoding/json"
	"os"

	"github.com/jatassi/slipstream/internal/buildinfo"
)

func main() {
	s, ok := buildinfo.Current()
	if !ok {
		os.Stdout.WriteString("unstamped\n")
		return
	}
	json.NewEncoder(os.Stdout).Encode(s)
}
