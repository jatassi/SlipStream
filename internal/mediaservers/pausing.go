package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Transcoding is a server that pauses the Conversions on the Node it runs
// on, as its reaching Node's facts last told.
type Transcoding struct {
	Server link.ServerID
	Name   string
	// Known is false while the reaching Node is away or its facts don't
	// hold the server.
	Known bool
	// Now is a video transcode on the server now.
	Now bool
}

// TranscodingOn is every server that runs on node with Pause Conversions
// on, by name. Each server's Known is its own reaching Node's.
func (m *MediaServers) TranscodingOn(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) ([]Transcoding, error) {
	rows, err := db.New(r).ListServers(ctx)
	if err != nil {
		return nil, err
	}
	var out []Transcoding
	for _, row := range rows {
		if row.RunsOnNodeID != node || !row.PausesConversions {
			continue
		}
		t := Transcoding{Server: row.ID, Name: row.Name}
		if m.up(row.ReachingNodeID) {
			f, _ := m.cfg.Facts(row.ReachingNodeID)
			sf := serverFactsOf(f, row.ID)
			t.Known, t.Now = sf.GetResponding(), sf.GetTranscoding()
		}
		out = append(out, t)
	}
	return out, nil
}
