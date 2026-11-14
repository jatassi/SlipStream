package plextv

import "testing"

func TestKeyIDIsTheThumbprintPlexTVTook(t *testing.T) {
	const x, recorded = "jO8OF37q9Swos0v5R_aSS6XX4JEke2YhUPZ5Ejyeflo", "m7k4bHb6ozt5thcrff7f1fpYnLfvK22egUBndADdj9M"
	if got := keyID(x); got != recorded {
		t.Fatalf("kid of the recorded run's key = %s, want %s", got, recorded)
	}
}
