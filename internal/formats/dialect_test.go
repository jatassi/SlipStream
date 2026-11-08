package formats

import (
	"encoding/json"
	"testing"
)

// TestEveryUpstreamValueHasAReading fails when a resync adds an enum value
// the hand tables in dialect.go don't map.
func TestEveryUpstreamValueHasAReading(t *testing.T) {
	enums := []struct {
		dialect Dialect
		kind    Kind
		values  []upstreamValue
	}{
		{Radarr, Source, radarrQualitySource},
		{Radarr, Resolution, radarrResolution},
		{Radarr, QualityModifier, radarrModifier},
		{Radarr, Language, radarrLanguage},
		{Radarr, IndexerFlag, radarrIndexerFlags},
		{Sonarr, Source, sonarrQualitySource},
		{Sonarr, Resolution, sonarrResolution},
		{Sonarr, Language, sonarrLanguage},
		{Sonarr, IndexerFlag, sonarrIndexerFlags},
		{Sonarr, ReleaseType, sonarrReleaseType},
	}
	for _, e := range enums {
		for _, v := range e.values {
			fields, err := json.Marshal(map[string]int{"value": v.Value})
			if err != nil {
				t.Fatal(err)
			}
			f := Format{Dialect: e.dialect, Conditions: []Condition{{Name: v.Name, Kind: e.kind, Fields: fields}}}
			if err := Check(f, DefaultTimings()); err != nil {
				t.Errorf("%s %s %s (%d) has no reading in dialect.go: %v", e.dialect, e.kind, v.Name, v.Value, err)
			}
		}
	}
}
