// Package proctree starts a program as the root of a process tree that
// is killed whole, so nothing the program starts outlives it.
package proctree

import (
	"bytes"
	"os/exec"
)

// Run is cmd.Run under Start: the tree ends with cmd, and on Windows with
// the process that started it.
func Run(cmd *exec.Cmd) error {
	end, err := Start(cmd)
	if err != nil {
		return err
	}
	defer end()
	return cmd.Wait() //nolint:wrapcheck // the caller reports how its program ended
}

// Output is cmd.Output under Start.
func Output(cmd *exec.Cmd) ([]byte, error) {
	var out bytes.Buffer
	cmd.Stdout = &out
	err := Run(cmd)
	return out.Bytes(), err
}
