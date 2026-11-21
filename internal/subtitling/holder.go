package subtitling

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media/speech"
	subtitlesv1 "github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/subtitles/v1/subtitlesv1connect"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// Holder is SubtitlesService on a Node. It runs each call with the keys,
// map, gate and constants the call carries, and answers every outcome the
// call produced, a provider's failure included.
type Holder struct {
	subtitlesv1connect.UnimplementedSubtitlesServiceHandler
	providers map[ProviderID]Provider
	timings   Timings
}

var _ subtitlesv1connect.SubtitlesServiceHandler = (*Holder)(nil)

func Serve(ps []Provider, t Timings) *Holder {
	h := &Holder{providers: map[ProviderID]Provider{}, timings: t}
	for _, p := range ps {
		h.providers[p.Spec().ID] = p
	}
	return h
}

func (h *Holder) provider(id string) (Provider, error) {
	p, ok := h.providers[ProviderID(id)]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node has no subtitle provider %q", id)
	}
	return p, nil
}

func (h *Holder) CheckKey(ctx context.Context, req *subtitlesv1.CheckKeyRequest) (*subtitlesv1.CheckKeyResponse, error) {
	p, err := h.provider(req.GetProvider())
	if err != nil {
		return nil, err
	}
	c := calls.Credentials{Keys: map[calls.ID]string{p.Spec().ID: req.GetKey()}, Tally: &calls.Tally{}}
	a, err := p.CheckKey(ctx, c)
	outcomes, err := settle(ctx, p, c.Tally, err)
	if err != nil {
		return nil, err
	}
	return &subtitlesv1.CheckKeyResponse{Outcomes: outcomesToWire(outcomes), Account: a.Name, Plan: a.Plan}, nil
}

// Search asks every listed provider, each with its key. A provider that
// fails answers its outcome and no offers; the others go on.
func (h *Holder) Search(ctx context.Context, req *subtitlesv1.SearchRequest) (*subtitlesv1.SearchResponse, error) {
	q, err := queryOf(req.GetSubject(), req.GetQuery())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	ps := make([]Provider, len(req.GetProviders()))
	for i, id := range req.GetProviders() {
		if ps[i], err = h.provider(id); err != nil {
			return nil, err
		}
	}
	res := &subtitlesv1.SearchResponse{}
	for _, p := range ps {
		c := calls.Credentials{Keys: keysOf(req.GetKeys()), Tally: &calls.Tally{}}
		offers, err := p.Search(ctx, c, q)
		outcomes, err := settle(ctx, p, c.Tally, err)
		if err != nil {
			return nil, err
		}
		res.Outcomes = append(res.Outcomes, outcomesToWire(outcomes)...)
		for _, o := range offers {
			res.Offers = append(res.Offers, offerToWire(o))
		}
	}
	return res, nil
}

// Fetch downloads one offer, unpacks it, picks the member for the asked
// episodes, parses it and aligns it against the map. A pass carries the
// retimed and original bytes, the claimed timing, and up to PackMembers
// other members of a pack within PackBytes, as converted originals.
func (h *Holder) Fetch(ctx context.Context, req *subtitlesv1.FetchRequest) (*subtitlesv1.FetchResponse, error) {
	p, err := h.provider(req.GetOffer().GetProvider())
	if err != nil {
		return nil, err
	}
	offer, err := offerOf(req.GetOffer())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	want, err := queryOf(req.GetSubject(), req.GetQuery())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	s, err := settingsOf(req.GetMap(), req.GetGate(), req.GetConstants(), want.Language, req.GetFps())
	if err != nil {
		return nil, err
	}
	c := calls.Credentials{Keys: keysOf(req.GetKeys()), Tally: &calls.Tally{}}
	dl, unfetched := p.Download(ctx, c, offer)
	served := unfetched == nil
	outcomes, err := settle(ctx, p, c.Tally, unfetched)
	if err != nil {
		return nil, err
	}
	res := &subtitlesv1.FetchResponse{Key: offer.Key, Outcomes: outcomesToWire(outcomes), Verdict: subtitlesv1.Verdict_VERDICT_UNFETCHED}
	if !served {
		return res, nil
	}
	entries, refused := h.unpack(dl)
	if refused.verdict != 0 {
		refused.into(res)
		return res, nil
	}
	chosen, ok := cues.Pick(entries, want.Subject.Season, want.Subject.Episodes, want.Language)
	if !ok {
		res.Verdict = subtitlesv1.Verdict_VERDICT_WRONG_EPISODE
		return res, nil
	}
	a, err := h.time(chosen.Bytes, s)
	if err != nil {
		return nil, err
	}
	a.into(res)
	if a.verdict == Passed && want.Subject.Kind == Episode {
		res.Members, res.MembersLeft = h.members(entries, chosen, want.Subject.Season, s)
	}
	return res, nil
}

// Sync is Fetch for bytes the Controller already holds: an upload, a
// sibling's Sidecar, a held pack member, or a carried Sidecar to retime.
func (h *Holder) Sync(_ context.Context, req *subtitlesv1.SyncRequest) (*subtitlesv1.SyncResponse, error) {
	lang, err := language.Parse(req.GetLanguage())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	s, err := settingsOf(req.GetMap(), req.GetGate(), req.GetConstants(), lang, req.GetFps())
	if err != nil {
		return nil, err
	}
	a, err := h.time(req.GetBytes(), s)
	if err != nil {
		return nil, err
	}
	var f subtitlesv1.FetchResponse
	a.into(&f)
	return &subtitlesv1.SyncResponse{Verdict: f.GetVerdict(), Format: f.GetFormat(), Original: f.GetOriginal(), Retimed: f.GetRetimed(), Fit: f.GetFit(), Words: f.GetWords()}, nil
}

// unpack is a download's subtitle members: an archive's, or the download
// itself when it is no archive. A refusal's verdict is set.
func (h *Holder) unpack(dl Download) ([]cues.Entry, answer) {
	entries, err := cues.Unpack(dl.Bytes, cues.Caps{Members: h.timings.ArchiveMembers, Member: h.timings.FetchCap, Total: h.timings.ArchiveCap})
	switch {
	case errors.Is(err, cues.ErrNotArchive):
		return []cues.Entry{{Name: dl.Name, Bytes: dl.Bytes}}, answer{}
	case errors.Is(err, cues.ErrTooLarge):
		return nil, answer{verdict: TooLarge, words: err.Error()}
	case err != nil:
		return nil, answer{verdict: Unreadable, words: err.Error()}
	case len(entries) == 0:
		return nil, answer{verdict: Unreadable, words: "the archive holds no subtitle"}
	}
	return entries, answer{}
}

// settings are what one call times a subtitle with.
type settings struct {
	ref  timing.Reference
	gate timing.Gate
	k    timing.Constants
	lang language.Code
	fps  float64
}

func settingsOf(m []byte, g *subtitlesv1.Gate, k *subtitlesv1.Constants, lang language.Code, fps float64) (settings, error) {
	sm, err := speech.Decode(m)
	if err != nil {
		return settings{}, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	gate, err := gateOf(g)
	if err != nil {
		return settings{}, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return settings{ref: timing.NewReference(sm.Bits(), sm.Header.Frames), gate: gate, k: constantsOf(k), lang: lang, fps: fps}, nil
}

// answer is one subtitle timed against a map.
type answer struct {
	verdict           Verdict
	words             string
	format            cues.Format
	original, retimed []byte
	fit               timing.Fit
	timed             bool
}

func (a answer) into(res *subtitlesv1.FetchResponse) {
	res.Verdict, res.Words, res.Format = verdicts[a.verdict], a.words, formats[a.format]
	res.Original, res.Retimed = a.original, a.retimed
	if a.timed {
		res.Fit = fitToWire(a.fit)
	}
}

// time parses b and aligns it. Only a bug fails it: Apply writes each time
// on the Doc's unit, so Retimed never refuses one.
func (h *Holder) time(b []byte, s settings) (answer, error) {
	if len(b) > h.timings.FetchCap {
		return answer{verdict: TooLarge, words: fmt.Sprintf("it's %d bytes, over the %d byte limit", len(b), h.timings.FetchCap)}, nil
	}
	doc, err := cues.Parse(b, s.lang, s.fps, h.timings.FetchCap)
	if errors.Is(err, cues.ErrImage) {
		return answer{verdict: Image}, nil
	}
	if u, ok := errors.AsType[cues.Unreadable](err); ok {
		return answer{verdict: Unreadable, words: u.Cause}, nil
	}
	if err != nil {
		return answer{}, err
	}
	spans := Spans(doc)
	fit := timing.Align(s.ref, spans, s.gate, s.k)
	a := answer{verdict: FailedTiming, format: doc.Format(), original: doc.Bytes(), fit: fit, timed: true}
	if fit.Verdict != timing.Passed {
		return a, nil
	}
	retimed, err := doc.Retimed(times(timing.Apply(spans, fit, doc.Unit())))
	if err != nil {
		return answer{}, fmt.Errorf("retime the subtitle: %w", err)
	}
	a.verdict, a.retimed = Passed, retimed.Bytes()
	return a, nil
}

// members are the pack's other episodes that parse, in archive order,
// within PackMembers and PackBytes, and how many more there were. A member
// whose name gives no episode, as a pack's notice file, is no episode.
func (h *Holder) members(entries []cues.Entry, chosen cues.Entry, season int, s settings) (out []*subtitlesv1.Member, left int32) {
	size := 0
	for _, e := range entries {
		n, episodes := cues.Numbered(e.Name)
		if e.Name == chosen.Name || len(episodes) == 0 || episodes[0] == 0 {
			continue
		}
		doc, err := cues.Parse(e.Bytes, s.lang, s.fps, h.timings.FetchCap)
		if err != nil {
			continue
		}
		b := doc.Bytes()
		if len(out) == h.timings.PackMembers || size+len(b) > h.timings.PackBytes {
			left++
			continue
		}
		size += len(b)
		if n == 0 {
			n = season
		}
		out = append(out, &subtitlesv1.Member{Name: e.Name, Season: int32(n), Episodes: int32s(episodes), Format: formats[doc.Format()], Original: b}) //nolint:gosec // G115: a season number is far under 2³¹
	}
	return out, left
}

// settle is a provider call's outcomes. A *calls.Failed is already in the
// tally; any other error is an answer the client couldn't read, and an
// ended ctx fails the call.
func settle(ctx context.Context, p Provider, t *calls.Tally, err error) ([]calls.Outcome, error) {
	outcomes := t.Outcomes()
	if _, failed := errors.AsType[*calls.Failed](err); err == nil || failed {
		return outcomes, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("%s: %w", p.Spec().Label, ctxErr)
	}
	return append(outcomes, calls.Outcome{Provider: p.Spec().ID, Kind: calls.Unreachable, Words: err.Error(), At: time.Now()}), nil
}

func keysOf(m map[string]string) map[calls.ID]string {
	out := make(map[calls.ID]string, len(m))
	for k, v := range m {
		out[calls.ID(k)] = v
	}
	return out
}
