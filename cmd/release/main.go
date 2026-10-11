package main

import (
	"fmt"
	"os"
	"sort"
)

type command struct {
	summary string
	run     func(args []string) error
}

var commands = map[string]command{}

func register(name, summary string, run func(args []string) error) {
	if _, dup := commands[name]; dup {
		panic("release: duplicate command " + name)
	}
	commands[name] = command{summary, run}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "release: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err := cmd.run(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "release %s: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}

func usage() {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(os.Stderr, "usage: go run ./cmd/release <command> [args]")
	for _, name := range names {
		fmt.Fprintf(os.Stderr, "  %-15s %s\n", name, commands[name].summary)
	}
}
