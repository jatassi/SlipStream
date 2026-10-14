package worldtest

import (
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/world"
)

// Messages reads the event stream a run captured for one identity from
// its event log, each message after the last one Next returned.
type Messages struct {
	t      *testing.T
	runDir string
	as     string
	after  uint64
}

// NewMessages reads the stream the manifest identity as opened, from the
// run's launch.
func NewMessages(t *testing.T, m world.Manifest, as string) *Messages {
	return &Messages{t: t, runDir: m.RunDir, as: as}
}

// Next waits for the next message of event and returns its data.
func (s *Messages) Next(event string) string {
	s.t.Helper()
	rec, err := events.Wait(Context(s.t), s.runDir, events.Query{Kind: events.SSEMessage, Match: map[string]string{"as": s.as, "event": event}, After: s.after})
	if err != nil {
		s.t.Fatalf("the %s stream: %v", s.as, err)
	}
	s.after = rec.Seq
	var msg struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(rec.Data, &msg); err != nil {
		s.t.Fatalf("sse.message %d: %v", rec.Seq, err)
	}
	return msg.Data
}
