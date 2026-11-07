package hdbits

import (
	"net/http"
	"testing"

	_ "example.com/app/internal/searching"
)

func TestFetch(t *testing.T) { _, _ = http.Get("https://hdbits.test/") }
