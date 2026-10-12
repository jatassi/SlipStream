package main

import (
	"fmt"

	"github.com/jatassi/slipstream/fakes/world"
)

func setupLoopback(remove bool) error {
	ctx, cancel := interrupted()
	defer cancel()
	change := world.SetupLoopback
	if remove {
		change = world.RemoveLoopback
	}
	msg, err := change(ctx)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return nil
}
