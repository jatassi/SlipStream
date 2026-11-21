package world

import (
	"fmt"

	"github.com/jatassi/slipstream/fakes/internal/subcatalog"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// CatalogText is the file of the Scenario's catalog subtitle id as its
// provider's download holds it: a film's one file, or episode's of a
// season pack.
func CatalogText(s scenario.Scenario, id string, episode int) ([]byte, error) {
	for _, c := range s.SubtitleProviders.Catalog {
		if c.ID != id {
			continue
		}
		for _, sub := range subcatalog.Of(s, c.Provider) {
			if sub.ID != id {
				continue
			}
			for _, m := range subcatalog.Members(sub) {
				if m.Episode == episode {
					return m.Bytes, nil
				}
			}
			return nil, fmt.Errorf("catalog subtitle %s holds no episode %d", id, episode)
		}
	}
	return nil, fmt.Errorf("the Scenario's catalog lists no subtitle %s", id)
}
