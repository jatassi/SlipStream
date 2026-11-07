package api

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
)

type instants struct {
	At    time.Time           `json:"at"`
	Since *time.Time          `json:"since"`
	Last  Nullable[time.Time] `json:"last"`
}

// inNewYork runs the test as a Node whose process zone is five hours
// behind UTC, so a time.Now() it serves carries -05:00.
func inNewYork(t *testing.T) time.Time {
	t.Helper()
	zone, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	was := time.Local
	time.Local = zone
	t.Cleanup(func() { time.Local = was })
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).In(time.Local)
}

func TestAResponseWritesItsTimesInUTC(t *testing.T) {
	at := inNewYork(t)
	r := newRouter(t, &bytes.Buffer{})
	Register(r.Owner("fixture", Public), Public, Op{ID: "fixtureGet", Method: http.MethodGet, Path: "/fixture"},
		func(context.Context, identity.Anyone, *struct{}) (*struct{ Body instants }, error) {
			return &struct{ Body instants }{Body: instants{At: at, Since: &at, Last: Some(at)}}, nil
		})
	w := get(t, r.Handler(), "/api/v1/fixture")
	want := `{"at":"2000-01-01T00:00:00Z","since":"2000-01-01T00:00:00Z","last":"2000-01-01T00:00:00Z"}` + "\n"
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Fatalf("got %d %s, want 200 %s", w.Code, w.Body, want)
	}
}

func TestAStreamWritesItsTimesInUTC(t *testing.T) {
	at := inNewYork(t)
	r := newRouter(t, &bytes.Buffer{})
	RegisterStream(r.Owner("fixture", Public), Public, Op{ID: "streamInstants", Method: http.MethodGet, Path: "/instants", Status: http.StatusOK},
		[]StreamEvent{{Name: "fixture.instants", Payload: instants{}}}, func(context.Context, identity.Anyone, *struct{}) (StreamFunc, error) {
			return func(_ context.Context, send Send) error {
				return send(StreamMessage{Data: instants{At: at, Since: &at, Last: Some(at)}})
			}, nil
		})
	_, body := openStream(t, r.Handler(), "/api/v1/instants")
	want := "event: fixture.instants\ndata: {\"at\":\"2000-01-01T00:00:00Z\",\"since\":\"2000-01-01T00:00:00Z\",\"last\":\"2000-01-01T00:00:00Z\"}\n\n"
	if got := body(); got != want {
		t.Fatalf("stream = %q, want %q", got, want)
	}
}
