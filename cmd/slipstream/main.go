package main

import (
	"fmt"
	"os"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/notices"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
	_ "github.com/jatassi/slipstream/web"
)

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	switch os.Args[1] {
	case "version":
		printVersion()
	case "notices":
		fmt.Print(notices.Text)
	default:
		usage()
	}
}

func printVersion() {
	s, ok := buildinfo.Current()
	if !ok {
		fmt.Println("slipstream dev (unstamped build)")
		return
	}
	fmt.Printf("slipstream %s\n", s.Version)
	fmt.Printf("commit         %s\n", s.Commit)
	fmt.Printf("target         %s\n", s.Target)
	fmt.Printf("libtorrent     %s\n", libtorrent.Version())
	fmt.Printf("fingerprint    %s\n", s.Fingerprint)
	fmt.Printf("native bundle  %s\n", s.NativeBundle)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: slipstream version | notices")
	os.Exit(2)
}
