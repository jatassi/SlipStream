package main

import (
	"os"

	"example.com/app/fakes/contenttypes/music"
	"example.com/app/internal/node"
)

func main() {
	os.Setenv("SLIPSTREAM_DEV", "1")
	os.Exit(node.Main(os.Args, music.Def))
}
