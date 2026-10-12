package media

import (
	stdhttp "net/http"
	goos "os"
)

type env struct{}

func (env) Getenv(string) string { return "" }

func Shadow() {
	os := env{}
	_ = os.Getenv("HOME")
	_ = goos.Getenv("HOME")
	_ = stdhttp.Post
}
