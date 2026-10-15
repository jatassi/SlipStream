package node

import (
	"os"

	"example.com/app/internal/node/internal/desktop"
)

var _, _ = os.Args, desktop.Run
