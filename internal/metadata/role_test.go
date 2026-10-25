package metadata_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	home  = role.NodeID{UUID: storage.NewUUID()}
	box   = role.NodeID{UUID: storage.NewUUID()}
	crate = role.NodeID{UUID: storage.NewUUID()}
	names = map[role.NodeID]string{home: "home", box: "box", crate: "crate"}
)

// roster is home, the Controller, and the holders that may take metadata
// now, in pick order.
type roster struct{ may []role.NodeID }

func (roster) Self() role.NodeID { return home }

func (r roster) Pick(_ role.Name, prefer role.NodeID) (role.NodeID, bool) {
	if slices.Contains(r.may, prefer) {
		return prefer, true
	}
	if len(r.may) == 0 {
		return role.NodeID{}, false
	}
	return r.may[0], true
}

func (r roster) PickOther(_ role.Name, besides role.NodeID) (role.NodeID, bool) {
	for _, n := range r.may {
		if n != besides {
			return n, true
		}
	}
	return role.NodeID{}, false
}

func (roster) Why(role.Name) string { return "box is offline, and no other Node reads metadata" }

// heard is every call a Node's MetadataService received, as "node op arg",
// and the Ask each provider half was handed.
type heard struct {
	mu    sync.Mutex
	calls []string
	asks  []metadata.Ask
}

func (h *heard) add(call string, ask metadata.Ask) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, call)
	h.asks = append(h.asks, ask)
}

// unlisted keeps no changes list.
type unlisted struct{}

func (unlisted) Changes(context.Context, metadata.Ask, time.Time, time.Time, int) (metadata.ChangesPage, error) {
	return metadata.ChangesPage{}, metadata.ErrNoChanges
}

// shelf is a provider half that answers every key, or fails with fail.
type shelf struct {
	unlisted
	node string
	h    *heard
	fail error
}

func (s shelf) Search(_ context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	s.h.add(s.node+" search "+query, ask)
	return []works.Candidate{{Key: "k1", Label: query}}, s.fail
}

func (s shelf) Fetch(_ context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	s.h.add(s.node+" fetch "+string(key), ask)
	if s.fail != nil {
		return works.Fetched{}, s.fail
	}
	return works.Fetched{Label: "The Attic", Children: []works.Entry{{Key: "i1", Label: "Lamp"}}}, nil
}

func (shelf) Sources() []artwork.Source { return nil }

// unreachable is a Node whose link is down: no call reaches it.
type unreachable struct {
	metadatav1connect.UnimplementedMetadataServiceHandler
	node string
	h    *heard
}

func (u unreachable) Search(context.Context, *metadatav1.SearchRequest) (*metadatav1.SearchResponse, error) {
	u.h.add(u.node+" unreached", metadata.Ask{})
	return nil, link.ErrDisconnected
}

func (u unreachable) Fetch(context.Context, *metadatav1.FetchRequest) (*metadatav1.FetchResponse, error) {
	u.h.add(u.node+" unreached", metadata.Ask{})
	return nil, link.ErrDisconnected
}

// controller is the Controller's metadata over r, where each Node runs a
// shelf, failing as fails says, unless down lists it.
func controller(r roster, fails map[role.NodeID]error, down ...role.NodeID) (*metadata.Metadata, *heard) {
	h := &heard{}
	return metadata.New(metadata.Config{
		Roster: r,
		Service: func(node role.NodeID) metadatav1connect.MetadataServiceHandler {
			if slices.Contains(down, node) {
				return unreachable{node: names[node], h: h}
			}
			return metadata.NewHolder(metadata.HolderConfig{Types: map[works.TypeID]metadata.Source{"fixture": shelf{node: names[node], h: h, fail: fails[node]}}})
		},
		Rand: seeded(),
	}), h
}

func TestEveryCallLeavesFromTheHolderPickChose(t *testing.T) {
	for _, c := range []struct {
		name string
		may  []role.NodeID
		want string
	}{
		{"a Worker alone", []role.NodeID{box}, "box"},
		{"the Controller among them", []role.NodeID{box, home}, "home"},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, h := controller(roster{may: c.may}, nil)
			found, err := m.Search(t.Context(), "fixture", "attic")
			if err != nil || len(found) != 1 || found[0].Label != "attic" {
				t.Fatalf("Search = %+v, %v; want the candidate attic", found, err)
			}
			f, err := m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd})
			if err != nil || f.Label != "The Attic" {
				t.Fatalf("Fetch = %+v, %v; want The Attic", f, err)
			}
			if want := []string{c.want + " search attic", c.want + " fetch k1"}; !slices.Equal(h.calls, want) {
				t.Fatalf("calls = %v, want %v", h.calls, want)
			}
			for i, a := range h.asks {
				if a.Language != "en-US" || a.Region != "US" || a.Ratings != (i == 1) {
					t.Errorf("call %d carried %+v, want en-US and US, ratings only on the add", i, a)
				}
			}
		})
	}
}

func TestAnUnreachableHolderIsTriedOnceOnTheNext(t *testing.T) {
	m, h := controller(roster{may: []role.NodeID{box, crate}}, nil, box)
	if _, err := m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"box unreached", "crate fetch k1"}; !slices.Equal(h.calls, want) {
		t.Fatalf("calls = %v, want %v", h.calls, want)
	}

	m, h = controller(roster{may: []role.NodeID{box, crate}}, nil, box, crate)
	_, err := m.Search(t.Context(), "fixture", "attic")
	if se, ok := errors.AsType[*works.SourceError](err); !ok || !errors.Is(err, works.ErrUnavailable) || !strings.Contains(se.Words, "didn't respond") {
		t.Fatalf("Search with both holders down = %v, want ErrUnavailable saying no holder answered", err)
	}
	if want := []string{"box unreached", "crate unreached"}; !slices.Equal(h.calls, want) {
		t.Fatalf("calls = %v, want one try each, %v", h.calls, want)
	}

	m, h = controller(roster{may: []role.NodeID{box, crate}}, map[role.NodeID]error{box: works.ErrNotFound})
	if _, err := m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd}); !errors.Is(err, works.ErrNotFound) {
		t.Fatalf("Fetch = %v, want box's ErrNotFound", err)
	}
	if want := []string{"box fetch k1"}; !slices.Equal(h.calls, want) {
		t.Fatalf("calls = %v, want %v: an answer is never tried again", h.calls, want)
	}
}

func TestWhileNoNodeMayTheCallIsUnavailableSayingWhy(t *testing.T) {
	m, h := controller(roster{}, nil)
	_, err := m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd})
	want := &works.SourceError{Source: "Metadata", Words: "box is offline, and no other Node reads metadata", Err: works.ErrUnavailable}
	if se, ok := errors.AsType[*works.SourceError](err); !ok || *se != *want {
		t.Fatalf("Fetch = %#v, want %#v", err, want)
	}
	if len(h.calls) != 0 {
		t.Fatalf("calls = %v, want none", h.calls)
	}
}

// TestOutcomesTravelWithAFailure fetches through a provider half that
// reads its host twice, the second time with a key the host refuses: the
// holder answers the refusal with both outcomes, and the Controller reads
// the refusal with the provider named.
func TestOutcomesTravelWithAFailure(t *testing.T) {
	lines, base, _ := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) {
		if n == 1 {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	holder := metadata.NewHolder(metadata.HolderConfig{Types: map[works.TypeID]metadata.Source{"fixture": twice{line: lines.Line(tmdb), base: base}}})
	res, err := holder.Fetch(t.Context(), &metadatav1.FetchRequest{Ask: &metadatav1.Ask{}, ContentType: "fixture", Key: "k1"})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []metadatav1.OutcomeKind
	for _, o := range res.GetOutcomes() {
		if o.GetProvider() != "tmdb" || o.GetAtUnixMicro() == 0 {
			t.Errorf("outcome %v, want TMDB's with its time", o)
		}
		kinds = append(kinds, o.GetKind())
	}
	if want := []metadatav1.OutcomeKind{metadatav1.OutcomeKind_OUTCOME_KIND_OK, metadatav1.OutcomeKind_OUTCOME_KIND_REFUSED}; !slices.Equal(kinds, want) {
		t.Errorf("outcomes = %v, want %v", kinds, want)
	}
	if f := res.GetFailure(); f.GetKind() != metadatav1.FailureKind_FAILURE_KIND_REFUSED || f.GetSource() != "TMDB" || len(res.GetFetched()) != 0 {
		t.Errorf("answer = %v, want TMDB's refusal and no record", res)
	}

	m := metadata.New(metadata.Config{
		Roster:  roster{may: []role.NodeID{home}},
		Service: func(role.NodeID) metadatav1connect.MetadataServiceHandler { return holder },
		Rand:    seeded(),
	})
	_, err = m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd})
	if se, ok := errors.AsType[*works.SourceError](err); !ok || se.Source != "TMDB" || !errors.Is(err, works.ErrRefused) {
		t.Fatalf("Fetch = %v, want TMDB's refusal", err)
	}
}

// twice reads base, then base again, through one Line.
type twice struct {
	unlisted
	line *metadata.Line
	base *url.URL
}

func (twice) Search(context.Context, metadata.Ask, string) ([]works.Candidate, error) {
	return nil, nil
}

func (twice) Sources() []artwork.Source { return nil }

func (s twice) Fetch(ctx context.Context, ask metadata.Ask, _ works.Key) (works.Fetched, error) {
	var v struct{}
	for range 2 {
		if err := s.line.Get(ctx, ask, metadata.Request{URL: s.base}, &v); err != nil {
			return works.Fetched{}, err
		}
	}
	return works.Fetched{}, nil
}
