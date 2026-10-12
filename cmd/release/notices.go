package main

import (
	"fmt"

	"github.com/jatassi/slipstream/cmd/release/internal/notices"
)

func init() {
	register("notices", "generate or check the third-party notices file and the license allow-list", runNotices)
}

func runNotices(args []string) error {
	if len(args) != 1 || (args[0] != "generate" && args[0] != "check") {
		return fmt.Errorf("usage: notices {generate|check}")
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	if err := buildWeb(root); err != nil {
		return err
	}
	if args[0] == "generate" {
		return notices.Write(root)
	}
	return notices.Check(root)
}
