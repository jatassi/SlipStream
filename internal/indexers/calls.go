package indexers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
)

// Release is one normalized Release from one Indexer.
type Release struct {
	Indexer ID
	searching.Release
}

// Answer is one Indexer's answer to a search.
type Answer struct {
	Indexer  ID
	Outcome  Outcome
	Releases []Release
}

// Search asks idx for terms on the Node idx routes to now, and records the
// answer. Every way the Indexer couldn't answer is in the Answer's
// Outcome. It fails only when the library or ctx does.
func (x *Indexers) Search(ctx context.Context, idx Indexer, terms Terms) (Answer, error) {
	a, j, st, err := x.call(ctx, idx, UseSearch, searching.SearchCall{Query: Query(idx.Caps, terms)})
	out := Answer{Indexer: idx.ID}
	var wait *WaitError
	var limit *LimitError
	var unreached *UnavailableError
	switch {
	case ctx.Err() != nil:
		return Answer{}, fmt.Errorf("search Indexer %s: %w", idx.Name, ctx.Err())
	case errors.As(err, &wait):
		out.Outcome = wait.Outcome
	case errors.As(err, &limit):
		out.Outcome = Outcome{Kind: LimitReached, Until: limit.Until}
	case errors.Is(err, ErrLate):
		out.Outcome = Outcome{Kind: Late}
	case errors.As(err, &unreached):
		out.Outcome = Outcome{Kind: Waiting, Node: j.node}
	case err != nil:
		return Answer{}, err
	default:
		out.Outcome = outcomeOf(a, st)
		for _, r := range a.Releases {
			out.Releases = append(out.Releases, Release{Indexer: idx.ID, Release: r})
		}
	}
	return out, nil
}

// outcomeOf is a's Outcome, timed by the status it left.
func outcomeOf(a searching.Answer, st Status) Outcome {
	var stopped *searching.Stopped
	switch {
	case a.Outcome == nil && len(a.Releases) == 0:
		return Outcome{Kind: NoReleases}
	case a.Outcome == nil:
		return Outcome{Kind: Found, Releases: len(a.Releases)}
	case errors.As(a.Outcome, &stopped):
		return Outcome{Kind: IsStopped, Cause: stopped.Cause, Text: stopped.Text}
	case errors.As(a.Outcome, new(*searching.Limited)):
		return Outcome{Kind: IsLimited, Until: st.Until}
	}
	return Outcome{Kind: IsFailing, Until: st.Until}
}

// Query is the query for terms under caps: terms' mode with the first id
// parameter caps list for that mode, else a free-text search. It chooses
// by declared capability, so it never retries.
func Query(c Caps, terms Terms) searching.Query {
	for _, p := range c.Modes[terms.Mode] {
		if id, ok := terms.IDs[p]; ok {
			return searching.Query{Mode: terms.Mode, IDs: map[string]string{p: id}}
		}
	}
	return searching.Query{Mode: "search", Text: terms.Query}
}

// Got is what a Release's link leads to: a Torrent or a Magnet.
type Got interface{ got() }

// Torrent is a .torrent's bytes as the Indexer sent them, and the
// Searching Node that fetched them.
type Torrent struct {
	Raw  []byte
	Node role.NodeID
}

// Magnet is a magnet link: the Release's own, or the one its link
// redirected to.
type Magnet struct{ Link magnet.Link }

func (Torrent) got() {}
func (Magnet) got()  {}

// Fetch is what link leads to. A magnet link needs no Node. Any other
// link goes to the Node idx routes to, which may fetch it from any host
// and follow it to a magnet link. It fails with *WaitError, *LimitError,
// *searching.Stopped, *UnavailableError, ErrLate, *magnet.InvalidError or
// bencode.ErrV2Only.
func (x *Indexers) Fetch(ctx context.Context, idx Indexer, link searching.Link) (Got, error) {
	if link.IsMagnet() {
		return parseMagnet(link)
	}
	a, j, _, err := x.call(ctx, idx, UseFetch, searching.FetchCall{Link: link})
	if err != nil {
		return nil, err
	}
	if stopped, ok := errors.AsType[*searching.Stopped](a.Outcome); ok {
		return nil, stopped
	}
	if a.Outcome != nil {
		return nil, &UnavailableError{Err: a.Outcome}
	}
	switch got := a.Got.(type) {
	case searching.Torrent:
		return Torrent{Raw: got.Raw, Node: j.node}, nil
	case searching.Magnet:
		return parseMagnet(got.Link)
	}
	return nil, &UnavailableError{Err: errors.New("the Searching Node's response holds neither a .torrent nor a magnet link")}
}

func parseMagnet(link searching.Link) (Got, error) {
	l, err := link.Magnet()
	if err != nil {
		return nil, err
	}
	return Magnet{Link: l}, nil
}

// CategoriesOf is ids as Newznab category ids.
func CategoriesOf(ids []int) []newznab.ID {
	if ids == nil {
		return nil
	}
	out := make([]newznab.ID, len(ids))
	for i, id := range ids {
		out[i] = newznab.ID(id) //nolint:gosec // G115: Newznab category ids have at most six digits
	}
	return out
}

// waits is why idx's searches wait now, nil when they don't: the gate a
// search reads, for the Indexer page.
func (x *Indexers) waits(idx Indexer, st Status, used Usage, now time.Time) *Outcome {
	_, wait := gate(idx, st, used, x.cfg.Roster, UseSearch, now)
	return wait
}
