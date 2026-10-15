package main

import (
	"os"

	"example.com/app/internal/buildinfo"
	"golang.org/x/sys/windows"
)

func main() { _, _, _ = os.Args, buildinfo.Current, windows.CREATE_NO_WINDOW }
