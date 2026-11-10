package indexers

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

// Standing is whether an Indexer takes a Release now, as Take reads it
// before it reserves a grab.
type Standing struct {
	Indexer ID
	// Refuses is why it takes no Release: removed or stopped. Nil while it
	// takes one.
	Refuses *NotTakenError
	// AtGrabLimit is why it takes no Release offering a .torrent link while
	// its grab limit is used up. A magnet link reserves no grab, so it
	// still takes one.
	AtGrabLimit *NotTakenError
	// Automatic is its Automatic search switch.
	Automatic bool
}

// Take is what Indexers.Take answers r with now, reserving nothing: a
// Ticket naming r's Indexer, or *NotTakenError.
func (s Standing) Take(r Release) (Ticket, error) {
	switch {
	case s.Refuses != nil:
		return Ticket{}, s.Refuses
	case s.AtGrabLimit != nil && !r.Offered().IsMagnet():
		return Ticket{}, s.AtGrabLimit
	}
	return Ticket{Indexer: s.Indexer}, nil
}

// Standings is how each Indexer stands, by id. One it doesn't hold was
// deleted.
type Standings map[ID]Standing

// Take is what Indexers.Take answers r with now, reserving nothing, for
// Manual search's plan to choose by.
func (s Standings) Take(r Release) (Ticket, error) {
	st, ok := s[r.Indexer]
	if !ok {
		st = removed(r.Indexer)
	}
	return st.Take(r)
}

// Standings reads, in r, how every Indexer stands now. It reads Take's own
// rule, so a Release it takes is one Take would take in the same snapshot.
func (x *Indexers) Standings(ctx context.Context, r storage.Reader[file]) (Standings, error) {
	all, err := x.all(ctx, r)
	if err != nil {
		return nil, err
	}
	now := storage.Now().Time()
	out := Standings{}
	for _, idx := range all {
		if out[idx.ID], err = x.standing(ctx, r, idx, now); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// standing is how idx stands at now, read in r.
func (x *Indexers) standing(ctx context.Context, r storage.Reader[file], idx Indexer, now time.Time) (Standing, error) {
	st, err := status(ctx, r, idx.ID)
	if err != nil {
		return Standing{}, err
	}
	s := Standing{Indexer: idx.ID, Automatic: idx.AutoSearch}
	if st.State == Stopped {
		o := Outcome{Kind: IsStopped, Cause: st.Cause, Text: st.Text, Node: st.Node}
		sentence, _ := x.waitWords(View{Indexer: idx, Status: st}, o)
		s.Refuses = &NotTakenError{Outcome: o, Sentence: sentence}
		return s, nil
	}
	if l := idx.Grabs; l.Count > 0 {
		reserved, err := reservedFor(ctx, r, idx.ID, UseFetch, now.Add(-l.Per.Duration()))
		if err != nil {
			return Standing{}, err
		}
		if until := free(l, reserved, now); !until.IsZero() {
			s.AtGrabLimit = &NotTakenError{
				Outcome:  Outcome{Kind: LimitReached, Until: until},
				Sentence: fmt.Sprintf("%s is at its grab limit until %s.", idx.Name, x.clock(until)),
			}
		}
	}
	return s, nil
}

// removed is how an Indexer deleted since it listed a Release stands.
func removed(id ID) Standing {
	return Standing{Indexer: id, Refuses: &NotTakenError{Outcome: Outcome{Kind: Removed}, Sentence: "The Indexer that listed it was deleted."}}
}
