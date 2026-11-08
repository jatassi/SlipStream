package formats

import (
	"encoding/json"
	"strconv"
	"testing"
)

// TestEveryOfferedValueHasAReading fails when a resync adds an enum value
// the hand tables in dialect.go don't map, or a SlipStream value has no
// reading.
func TestEveryOfferedValueHasAReading(t *testing.T) {
	for _, d := range []Dialect{Radarr, Sonarr, SlipStream} {
		for _, k := range KindsIn(d) {
			for _, v := range k.Values {
				var value any = v.Key
				if d != SlipStream {
					value, _ = strconv.Atoi(v.Key)
				}
				fields, err := json.Marshal(map[string]any{"value": value})
				if err != nil {
					t.Fatal(err)
				}
				f := Format{Dialect: d, Conditions: []Condition{{Name: v.Words, Kind: k.Kind, Fields: fields}}}
				if err := Check(f, DefaultTimings()); err != nil {
					t.Errorf("%s %s %s (%s) has no reading in dialect.go: %v", d, k.Kind, v.Words, v.Key, err)
				}
			}
		}
	}
}
