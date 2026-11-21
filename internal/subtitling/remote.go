package subtitling

import (
	"context"

	"github.com/jatassi/slipstream/internal/calls"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
)

// Remote is SubtitlesService on one holder, in domain types, for the
// Controller. What it answers is the holder's word: the Controller checks
// it before it trusts any of it.
type Remote struct {
	service subtitlesv1connect.SubtitlesServiceHandler
}

func NewRemote(s subtitlesv1connect.SubtitlesServiceHandler) Remote { return Remote{service: s} }

func (r Remote) CheckKey(ctx context.Context, p ProviderID, key string) (Account, []calls.Outcome, error) {
	res, err := r.service.CheckKey(ctx, &subtitlesv1.CheckKeyRequest{Provider: string(p), Key: key})
	if err != nil {
		return Account{}, nil, err
	}
	return Account{Name: res.GetAccount(), Plan: res.GetPlan()}, outcomesOf(res.GetOutcomes()), nil
}

// Search asks each of providers, in order, with its key in keys. An offer
// whose language doesn't parse is dropped.
func (r Remote) Search(ctx context.Context, q Query, providers []ProviderID, keys map[ProviderID]string) ([]Offer, []calls.Outcome, error) {
	req := &subtitlesv1.SearchRequest{Subject: subjectToWire(q.Subject), Query: queryToWire(q), Keys: wireKeys(keys)}
	for _, p := range providers {
		req.Providers = append(req.Providers, string(p))
	}
	res, err := r.service.Search(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	offers := make([]Offer, 0, len(res.GetOffers()))
	for _, w := range res.GetOffers() {
		o, err := offerOf(w)
		if err != nil {
			continue
		}
		offers = append(offers, o)
	}
	return offers, outcomesOf(res.GetOutcomes()), nil
}

func (r Remote) Fetch(ctx context.Context, f FetchAsk) (Fetched, []calls.Outcome, error) {
	res, err := r.service.Fetch(ctx, &subtitlesv1.FetchRequest{
		Offer: offerToWire(f.Offer), Keys: map[string]string{string(f.Offer.Provider): f.Key}, Subject: subjectToWire(f.Want.Subject), Query: queryToWire(f.Want),
		Map: f.Map, Gate: gateToWire(f.Gate), Fps: f.FPS, Constants: constantsToWire(f.Constants),
	})
	if err != nil {
		return Fetched{}, nil, err
	}
	out := fetched(res.GetVerdict(), res.GetWords(), res.GetFormat(), res.GetOriginal(), res.GetRetimed(), res.GetFit())
	out.Key, out.MembersLeft = res.GetKey(), int(res.GetMembersLeft())
	for _, m := range res.GetMembers() {
		out.Members = append(out.Members, Member{Name: m.GetName(), Season: int(m.GetSeason()), Episodes: ints(m.GetEpisodes()), Format: formatOf(m.GetFormat()), Bytes: m.GetOriginal()})
	}
	return out, outcomesOf(res.GetOutcomes()), nil
}

func (r Remote) Sync(ctx context.Context, s SyncAsk) (Fetched, error) {
	res, err := r.service.Sync(ctx, &subtitlesv1.SyncRequest{
		Name: s.Name, Bytes: s.Bytes, Language: s.Language.String(), Map: s.Map, Gate: gateToWire(s.Gate), Fps: s.FPS, Constants: constantsToWire(s.Constants),
	})
	if err != nil {
		return Fetched{}, err
	}
	return fetched(res.GetVerdict(), res.GetWords(), res.GetFormat(), res.GetOriginal(), res.GetRetimed(), res.GetFit()), nil
}

func fetched(v subtitlesv1.Verdict, words, format string, original, retimed []byte, fit *subtitlesv1.Fit) Fetched {
	f := Fetched{Verdict: verdictOf(v), Words: words, Format: formatOf(format), Original: original, Retimed: retimed}
	if fit != nil {
		f.Claimed = fitOf(fit)
	}
	return f
}

func wireKeys(keys map[ProviderID]string) map[string]string {
	out := make(map[string]string, len(keys))
	for k, v := range keys {
		out[string(k)] = v
	}
	return out
}
