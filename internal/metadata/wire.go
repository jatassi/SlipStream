package metadata

import (
	"errors"
	"time"

	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/works"
)

func askToWire(a Ask) *metadatav1.Ask {
	keys := make(map[string]string, len(a.keys))
	for p, k := range a.keys {
		keys[string(p)] = k
	}
	return &metadatav1.Ask{
		Keys: keys, Language: a.Language, Region: a.Region, Ratings: a.Ratings, Fresh: a.fresh,
		CacheForSeconds: int64(a.cacheFor / time.Second), Generation: a.generation,
	}
}

func askOf(w *metadatav1.Ask) Ask {
	keys := make(map[ProviderID]string, len(w.GetKeys()))
	for p, k := range w.GetKeys() {
		keys[ProviderID(p)] = k
	}
	return Ask{
		Language: w.GetLanguage(), Region: w.GetRegion(), Ratings: w.GetRatings(),
		keys: keys, fresh: w.GetFresh(), cacheFor: time.Duration(w.GetCacheForSeconds()) * time.Second, generation: w.GetGeneration(),
	}
}

var outcomeKinds = map[OutcomeKind]metadatav1.OutcomeKind{
	OK:          metadatav1.OutcomeKind_OUTCOME_KIND_OK,
	NotFound:    metadatav1.OutcomeKind_OUTCOME_KIND_NOT_FOUND,
	Refused:     metadatav1.OutcomeKind_OUTCOME_KIND_REFUSED,
	Limited:     metadatav1.OutcomeKind_OUTCOME_KIND_LIMITED,
	Capped:      metadatav1.OutcomeKind_OUTCOME_KIND_CAPPED,
	Unreachable: metadatav1.OutcomeKind_OUTCOME_KIND_UNREACHABLE,
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
	if w.GetSource() == "" && w.GetWords() == "" {
		return err
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
