package main

import (
	"os"

	"github.com/jatassi/slipstream/fakes/contenttypes/music"
	"github.com/jatassi/slipstream/internal/node"
)

func main() {
	os.Exit(node.Main(os.Args, music.Def))
}
