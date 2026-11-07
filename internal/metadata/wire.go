package metadata

import (
	"errors"
	"time"

	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/works"
)

func askToWire(a Ask) *metadatav1.Ask {
	return &metadatav1.Ask{
		Keys: byName(a.keys), Pins: byName(a.pins), Language: a.Language, Region: a.Region, Ratings: ratingsKinds[a.Ratings], Fresh: a.fresh,
		CacheForSeconds: int64(a.cacheFor / time.Second), Generation: a.generation,
	}
}

func askOf(w *metadatav1.Ask) Ask {
	return Ask{
		Language: w.GetLanguage(), Region: w.GetRegion(), Ratings: ratingsOf[w.GetRatings()],
		keys: byProvider(w.GetKeys()), pins: byProvider(w.GetPins()), fresh: w.GetFresh(),
		cacheFor: time.Duration(w.GetCacheForSeconds()) * time.Second, generation: w.GetGeneration(),
	}
}

func byName(m map[ProviderID]string) map[string]string {
	out := make(map[string]string, len(m))
	for p, v := range m {
		out[string(p)] = v
	}
	return out
}

func byProvider(m map[string]string) map[ProviderID]string {
	out := make(map[ProviderID]string, len(m))
	for p, v := range m {
		out[ProviderID(p)] = v
	}
	return out
}

var ratingsKinds = map[Ratings]metadatav1.Ratings{
	RatingsNone: metadatav1.Ratings_RATINGS_NONE,
	RatingsWork: metadatav1.Ratings_RATINGS_WORK,
	RatingsAll:  metadatav1.Ratings_RATINGS_ALL,
}

var ratingsOf = func() map[metadatav1.Ratings]Ratings {
	m := make(map[metadatav1.Ratings]Ratings, len(ratingsKinds))
	for k, w := range ratingsKinds {
		m[w] = k
	}
	return m
}()

var outcomeKinds = map[OutcomeKind]metadatav1.OutcomeKind{
	OK:          metadatav1.OutcomeKind_OUTCOME_KIND_OK,
	NotFound:    metadatav1.OutcomeKind_OUTCOME_KIND_NOT_FOUND,
	Refused:     metadatav1.OutcomeKind_OUTCOME_KIND_REFUSED,
	Limited:     metadatav1.OutcomeKind_OUTCOME_KIND_LIMITED,
	Capped:      metadatav1.OutcomeKind_OUTCOME_KIND_CAPPED,
	Unreachable: metadatav1.OutcomeKind_OUTCOME_KIND_UNREACHABLE,
	PinRefused:  metadatav1.OutcomeKind_OUTCOME_KIND_PIN_REFUSED,
}

func outcomesToWire(os []Outcome) []*metadatav1.Outcome {
	out := make([]*metadatav1.Outcome, len(os))
	for i, o := range os {
		out[i] = &metadatav1.Outcome{
			Provider: string(o.Provider), Kind: outcomeKinds[o.Kind], UntilUnixMicro: unixMicro(o.Until), Words: o.Words, AtUnixMicro: unixMicro(o.At),
		}
	}
	return out
}

// failureKinds is each failure a provider half may end in, as the error it
// wraps and its kind on the wire.
var failureKinds = []struct {
	err  error
	kind metadatav1.FailureKind
}{
	{works.ErrNotFound, metadatav1.FailureKind_FAILURE_KIND_NOT_FOUND},
	{works.ErrRefused, metadatav1.FailureKind_FAILURE_KIND_REFUSED},
	{works.ErrUnavailable, metadatav1.FailureKind_FAILURE_KIND_UNAVAILABLE},
	{works.ErrInvalid, metadatav1.FailureKind_FAILURE_KIND_INVALID},
	{ErrNoChanges, metadatav1.FailureKind_FAILURE_KIND_NO_CHANGES},
	{works.ErrNotOnSource, metadatav1.FailureKind_FAILURE_KIND_NOT_ON_SOURCE},
	{works.ErrOrderUnavailable, metadatav1.FailureKind_FAILURE_KIND_ORDER_UNAVAILABLE},
}

// failureToWire is err as a provider failure, false when err is none: the
// holder itself failed.
func failureToWire(err error) (*metadatav1.Failure, bool) {
	for _, f := range failureKinds {
		if !errors.Is(err, f.err) {
			continue
		}
		w := &metadatav1.Failure{Kind: f.kind}
		if se, ok := errors.AsType[*works.SourceError](err); ok {
			w.Source, w.UntilUnixMicro, w.Words = se.Source, unixMicro(se.Until), se.Words
		} else if f.kind == metadatav1.FailureKind_FAILURE_KIND_NO_CHANGES {
			w.Words = err.Error()
		}
		return w, true
	}
	return nil, false
}

// failureOf is the error a holder's failure stands for: a *works.SourceError
// when the source said more than its kind.
func failureOf(w *metadatav1.Failure) error {
	var err error
	for _, f := range failureKinds {
		if f.kind == w.GetKind() {
			err = f.err
		}
	}
	switch {
	case w.GetSource() == "" && w.GetWords() == "":
		return err
	case w.GetSource() == "":
		return worded{words: w.GetWords(), err: err}
	}
	se := &works.SourceError{Source: w.GetSource(), Words: w.GetWords(), Err: err}
	if u := w.GetUntilUnixMicro(); u != 0 {
		se.Until = time.UnixMicro(u)
	}
	return se
}

func unixMicro(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMicro()
}

var outcomeKindOf = func() map[metadatav1.OutcomeKind]OutcomeKind {
	m := make(map[metadatav1.OutcomeKind]OutcomeKind, len(outcomeKinds))
	for k, w := range outcomeKinds {
		m[w] = k
	}
	return m
}()

func outcomesOf(ws []*metadatav1.Outcome) []Outcome {
	out := make([]Outcome, len(ws))
	for i, w := range ws {
		out[i] = Outcome{Provider: ProviderID(w.GetProvider()), Kind: outcomeKindOf[w.GetKind()], Words: w.GetWords(), At: time.UnixMicro(w.GetAtUnixMicro())}
		if u := w.GetUntilUnixMicro(); u != 0 {
			out[i].Until = time.UnixMicro(u)
		}
	}
	return out
}

// worded is a holder's failure in the holder's own words.
type worded struct {
	words string
	err   error
}

func (w worded) Error() string { return w.words }

func (w worded) Unwrap() error { return w.err }
