package tmdb_test

import (
	"testing"

	conformance "example.com/app/conformance/tmdb"
)

func TestTheFakeConforms(t *testing.T) { conformance.Run(t) }
