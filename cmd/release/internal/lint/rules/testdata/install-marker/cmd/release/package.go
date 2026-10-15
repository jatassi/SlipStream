package main

import "example.com/app/internal/install"

func stage() {
	_ = install.Write(install.MarkerPath(install.Layout{}), 0)
	_, _ = install.Read(install.Layout{})
}
