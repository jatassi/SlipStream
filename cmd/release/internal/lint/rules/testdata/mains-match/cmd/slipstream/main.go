package main

import (
	"os"

	"example.com/app/internal/node"
)

func main() {
	os.Exit(node.Main(os.Args))
}
