package main

import (
	"os"

	"example.com/app/internal/node"
	"example.com/app/fakes/contenttypes/music"
)

func main() {
	os.Setenv("SLIPSTREAM_DEV", "1")
	os.Exit(node.Main(os.Args, music.Def))
}
