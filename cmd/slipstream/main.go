package main

import (
	"os"

	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/node"
)

func main() {
	os.Exit(node.Main(os.Args, movie.Def))
}
