package music_test

import (
	"testing"

	"example.com/app/conformance/tmdb"
)

func TestTheWrongSuite(t *testing.T) { tmdb.Run(t) }
