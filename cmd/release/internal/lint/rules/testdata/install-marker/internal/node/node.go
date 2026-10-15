package node

import "example.com/app/internal/install"

func ProcessEnv() install.Kind {
	k, _ := install.Read(install.Layout{})
	return k
}

func other() {
	_, _ = install.Read(install.Layout{})
	_ = install.MarkerPath(install.Layout{})
}

type env struct{}

func (env) ProcessEnv() { _ = install.Write("x", 0) }

var read = install.Read
