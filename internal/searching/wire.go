package searching

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/searching/internal/rss"
)

// Remote is the Controller's handle on one Node's Searching service.
type Remote interface {
	// Run starts job on the Node, or joins its run there. It answers
	// within the Node's hold: the job's Answer, or how the job stands.
	Run(ctx context.Context, job Job) (Poll, error)
}

// Poll is one Run call's result: the Answer once the job ended, else when
// it started, zero while it waits for its first request.
type Poll struct {
	Started time.Time
	Answer  *Answer
}

// RemoteOf is the Remote of one Node's generated client.
func RemoteOf(h searchv1connect.SearchingServiceHandler) Remote { return remote{h} }

type remote struct {
	h searchv1connect.SearchingServiceHandler
}

func (r remote) Run(ctx context.Context, job Job) (Poll, error) {
	res, err := r.h.Run(ctx, &searchv1.RunRequest{Job: encodeJob(job)})
	if err != nil {
		return Poll{}, err
	}
	switch a := res.GetAnswer().(type) {
	case *searchv1.RunResponse_Running:
		return Poll{Started: fromUnixMs(a.Running.GetStartedUnixMs())}, nil
	case *searchv1.RunResponse_Done:
		answer, err := decodeAnswer(a.Done)
		if err != nil {
			return Poll{}, err
		}
		return Poll{Started: answer.Started, Answer: &answer}, nil
	}
	return Poll{}, errors.New("the Node's response to Run holds neither a result nor a start time")
}

var kindOnWire = map[searchv1.Kind]Kind{
	searchv1.Kind_KIND_CATALOG: Catalog,
	searchv1.Kind_KIND_TORZNAB: Torznab,
	searchv1.Kind_KIND_RSS:     RSS,
}

var causeOnWire = map[searchv1.Stopped_Cause]StopCause{
	searchv1.Stopped_CAUSE_CREDENTIALS_REJECTED:  CredentialsRejected,
	searchv1.Stopped_CAUSE_CAPTCHA_NEEDED:        CaptchaNeeded,
	searchv1.Stopped_CAUSE_CHALLENGE_UNSOLVED:    ChallengeUnsolved,
	searchv1.Stopped_CAUSE_COOKIE_EXPIRED:        CookieExpired,
	searchv1.Stopped_CAUSE_DEFINITION_UNLOADABLE: DefinitionUnloadable,
	searchv1.Stopped_CAUSE_ACCOUNT_REFUSED:       AccountRefused,
	searchv1.Stopped_CAUSE_REQUEST_REJECTED:      RequestRejected,
}

var linkOnWire = map[searchv1.RSSReading_Link]rss.Link{
	searchv1.RSSReading_LINK_ENCLOSURE:  rss.Enclosure,
	searchv1.RSSReading_LINK_ITEM:       rss.Item,
	searchv1.RSSReading_LINK_TORRENT_NS: rss.Torrent,
}

var refusalOnWire = map[searchv1.RSSRefusal]RSSRefusal{
	searchv1.RSSRefusal_RSS_REFUSAL_NO_ITEMS:      RSSNoItems,
	searchv1.RSSRefusal_RSS_REFUSAL_NO_TITLE:      RSSNoTitle,
	searchv1.RSSRefusal_RSS_REFUSAL_NO_LINK:       RSSNoLink,
	searchv1.RSSRefusal_RSS_REFUSAL_REPEATED_GUID: RSSRepeatedGUID,
	searchv1.RSSRefusal_RSS_REFUSAL_NO_SIZE:       RSSNoSize,
}

func onWire[W comparable, D comparable](m map[W]D, d D) W {
	for w, v := range m {
		if v == d {
			return w
		}
	}
	var zero W
	return zero
}

func encodeJob(j Job) *searchv1.Job {
	w := &searchv1.Job{Id: j.ID, Site: &searchv1.Site{
		IndexerId: j.Indexer, Revision: j.Revision, Kind: onWire(kindOnWire, j.Kind), Address: j.Address,
		Settings: j.Settings, ApiKey: j.APIKey, Session: encodeSession(j.Session), Categories: int32s(j.Categories),
		Rss: encodeReading(j.RSS),
	}}
	switch c := j.Call.(type) {
	case SearchCall:
		q := c.Query
		w.Call = &searchv1.Job_Search{Search: &searchv1.SearchCall{Query: &searchv1.Query{
			Mode: q.Mode, Ids: q.IDs, Text: q.Text, Season: int32(q.Season), Episode: q.Episode, //nolint:gosec // a season fits
		}}}
	case RecentCall:
		w.Call = &searchv1.Job_Recent{Recent: &searchv1.RecentCall{}}
	case FetchCall:
		w.Call = &searchv1.Job_Fetch{Fetch: &searchv1.FetchCall{Link: c.Link.raw}}
	case TestCall:
		w.Call = &searchv1.Job_Test{Test: &searchv1.TestCall{}}
	}
	return w
}

// decodeJob reads a job and the Site it reaches, or why the Node can't
// run it.
func decodeJob(w *searchv1.Job) (Job, *Site, error) {
	s := w.GetSite()
	j := Job{
		ID: w.GetId(), Indexer: s.GetIndexerId(), Revision: s.GetRevision(), Kind: kindOnWire[s.GetKind()], Address: s.GetAddress(),
		Settings: s.GetSettings(), APIKey: s.GetApiKey(), Session: decodeSession(s.GetSession()), Categories: ints(s.GetCategories()),
		RSS: decodeReading(s.GetRss()),
	}
	if j.ID == "" {
		return Job{}, nil, errors.New("the job has no id")
	}
	address, err := httpURL(j.Address)
	if err != nil {
		return Job{}, nil, fmt.Errorf("the Indexer's address: %w", err)
	}
	switch c := w.GetCall().(type) {
	case *searchv1.Job_Search:
		q := c.Search.GetQuery()
		j.Call = SearchCall{Query: Query{Mode: q.GetMode(), IDs: q.GetIds(), Text: q.GetText(), Season: int(q.GetSeason()), Episode: q.GetEpisode()}}
	case *searchv1.Job_Recent:
		j.Call = RecentCall{}
	case *searchv1.Job_Fetch:
		if _, err := httpURL(c.Fetch.GetLink()); err != nil {
			return Job{}, nil, fmt.Errorf("the link to fetch: %w", err)
		}
		j.Call = FetchCall{Link: LinkOf(c.Fetch.GetLink())}
	case *searchv1.Job_Test:
		j.Call = TestCall{}
	default:
		return Job{}, nil, errors.New("the job names no call this Node runs")
	}
	site := &Site{Indexer: j.Indexer, Revision: j.Revision, Address: address, Settings: j.Settings, APIKey: j.APIKey, Categories: j.Categories, RSS: j.RSS}
	return j, site, nil
}

func httpURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("it isn't an http or https URL")
	}
	return u, nil
}

func encodeAnswer(a Answer) *searchv1.Answer {
	w := &searchv1.Answer{StartedUnixMs: unixMs(a.Started), Outcome: encodeOutcome(a.Outcome), Session: encodeSession(a.Session)}
	switch {
	case a.Releases != nil:
		rs := make([]*searchv1.Release, len(a.Releases))
		for i, r := range a.Releases {
			rs[i] = encodeRelease(r)
		}
		w.Result = &searchv1.Answer_Releases{Releases: &searchv1.Releases{Releases: rs}}
	case a.Got != nil:
		switch g := a.Got.(type) {
		case Torrent:
			w.Result = &searchv1.Answer_Got{Got: &searchv1.Got{Got: &searchv1.Got_Torrent{Torrent: g.Raw}}}
		case Magnet:
			w.Result = &searchv1.Answer_Got{Got: &searchv1.Got{Got: &searchv1.Got_Magnet{Magnet: g.Link.raw}}}
		}
	case a.Report != nil:
		modes := map[string]*searchv1.ModeParams{}
		for mode, params := range a.Report.Capabilities.Modes {
			modes[mode] = &searchv1.ModeParams{Params: params}
		}
		w.Result = &searchv1.Answer_Test{Test: &searchv1.TestReport{
			Capabilities: &searchv1.Capabilities{Modes: modes, Categories: int32s(a.Report.Capabilities.Categories)},
			Items:        int32(a.Report.Items), Sources: a.Report.Sources, //nolint:gosec // an item count fits
			Refusal: onWire(refusalOnWire, a.Report.Refusal),
		}}
		if a.Report.RSS != nil {
			w.GetTest().Rss = encodeReading(*a.Report.RSS)
		}
	}
	return w
}

func decodeAnswer(w *searchv1.Answer) (Answer, error) {
	outcome, err := decodeOutcome(w.GetOutcome())
	if err != nil {
		return Answer{}, err
	}
	a := Answer{Started: fromUnixMs(w.GetStartedUnixMs()), Outcome: outcome, Session: decodeSession(w.GetSession())}
	switch r := w.GetResult().(type) {
	case *searchv1.Answer_Releases:
		a.Releases = make([]Release, len(r.Releases.GetReleases()))
		for i, rel := range r.Releases.GetReleases() {
			if a.Releases[i], err = decodeRelease(rel); err != nil {
				return Answer{}, err
			}
		}
	case *searchv1.Answer_Got:
		switch g := r.Got.GetGot().(type) {
		case *searchv1.Got_Torrent:
			a.Got = Torrent{Raw: g.Torrent}
		case *searchv1.Got_Magnet:
			a.Got = Magnet{Link: LinkOf(g.Magnet)}
		}
	case *searchv1.Answer_Test:
		modes := map[string][]string{}
		for mode, p := range r.Test.GetCapabilities().GetModes() {
			modes[mode] = p.GetParams()
		}
		a.Report = &Report{
			Capabilities: Capabilities{Modes: modes, Categories: ints(r.Test.GetCapabilities().GetCategories())},
			Items:        int(r.Test.GetItems()), Sources: r.Test.GetSources(), Refusal: refusalOnWire[r.Test.GetRefusal()],
		}
		if w := r.Test.GetRss(); w != nil {
			reading := decodeReading(w)
			a.Report.RSS = &reading
		}
	}
	return a, nil
}

func encodeOutcome(err error) *searchv1.Outcome {
	if err == nil {
		return &searchv1.Outcome{Kind: &searchv1.Outcome_Ok{Ok: &searchv1.Ok{}}}
	}
	if o, ok := errors.AsType[*Limited](err); ok {
		return &searchv1.Outcome{Kind: &searchv1.Outcome_Limited{Limited: &searchv1.Limited{RetryAfterMs: o.RetryAfter.Milliseconds()}}}
	}
	if o, ok := errors.AsType[*Stopped](err); ok {
		return &searchv1.Outcome{Kind: &searchv1.Outcome_Stopped{Stopped: &searchv1.Stopped{
			Cause: onWire(causeOnWire, o.Cause), Text: o.Text, SolverTried: o.SolverTried,
		}}}
	}
	return &searchv1.Outcome{Kind: &searchv1.Outcome_Failing{Failing: &searchv1.Failing{Text: err.Error()}}}
}

func decodeOutcome(w *searchv1.Outcome) (outcome, err error) {
	switch o := w.GetKind().(type) {
	case *searchv1.Outcome_Ok:
		return nil, nil
	case *searchv1.Outcome_Limited:
		return &Limited{RetryAfter: time.Duration(o.Limited.GetRetryAfterMs()) * time.Millisecond}, nil
	case *searchv1.Outcome_Failing:
		return &Failing{Err: errors.New(o.Failing.GetText())}, nil
	case *searchv1.Outcome_Stopped:
		cause, ok := causeOnWire[o.Stopped.GetCause()]
		if !ok {
			return nil, fmt.Errorf("the Node's response names an unknown stop cause %v", o.Stopped.GetCause())
		}
		return &Stopped{Cause: cause, Text: o.Stopped.GetText(), SolverTried: o.Stopped.GetSolverTried()}, nil
	}
	return nil, errors.New("the Node's response has no outcome")
}

func encodeRelease(r Release) *searchv1.Release {
	w := &searchv1.Release{
		Title: r.Title, Size: r.Size, PublishedUnixMs: unixMs(r.Published),
		Seeders: int32(r.Seeders), Leechers: int32(r.Leechers), Grabs: int32(r.Grabs), //nolint:gosec // peer and grab counts fit
		Link: r.Link.raw, Magnet: r.Magnet.raw, Categories: int32s(r.Categories), Ids: r.IDs,
		DownloadFactor: r.Factors.Download, UploadFactor: r.Factors.Upload,
		MinRatio: r.Floor.MinRatio, MinSeedSeconds: int64(r.Floor.MinSeed / time.Second), Details: r.Details,
	}
	if r.InfoHash != nil {
		w.InfoHash = r.InfoHash[:]
	}
	if s := r.Source; s != nil {
		w.Source = &searchv1.Source{Id: s.ID, Name: s.Name, Privacy: s.Privacy, Origin: s.Origin}
	}
	return w
}

func decodeRelease(w *searchv1.Release) (Release, error) {
	r := Release{
		Title: w.GetTitle(), Size: w.GetSize(), Published: fromUnixMs(w.GetPublishedUnixMs()),
		Seeders: int(w.GetSeeders()), Leechers: int(w.GetLeechers()), Grabs: int(w.GetGrabs()),
		Link: LinkOf(w.GetLink()), Magnet: LinkOf(w.GetMagnet()), Categories: ints(w.GetCategories()), IDs: w.GetIds(),
		Factors: Factors{Download: w.GetDownloadFactor(), Upload: w.GetUploadFactor()},
		Floor:   Floor{MinRatio: w.GetMinRatio(), MinSeed: time.Duration(w.GetMinSeedSeconds()) * time.Second},
		Details: w.GetDetails(),
	}
	switch h := w.GetInfoHash(); len(h) {
	case 0:
	case len(bencode.InfoHash{}):
		ih := bencode.InfoHash(h)
		r.InfoHash = &ih
	default:
		return Release{}, fmt.Errorf("release %q has an info hash of %d bytes, not 20", r.Title, len(h))
	}
	if s := w.GetSource(); s != nil {
		r.Source = &Source{ID: s.GetId(), Name: s.GetName(), Privacy: s.GetPrivacy(), Origin: s.GetOrigin()}
	}
	return r, nil
}

func encodeSession(s *Session) *searchv1.Session {
	if s == nil {
		return nil
	}
	w := &searchv1.Session{UserAgent: s.UserAgent, Pasted: s.Pasted, EarnedUnixMs: unixMs(s.Earned)}
	for _, c := range s.Cookies {
		w.Cookies = append(w.Cookies, &searchv1.Cookie{
			Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path, ExpiresUnixMs: unixMs(c.Expires), Secure: c.Secure, HostOnly: c.HostOnly,
		})
	}
	return w
}

func decodeSession(w *searchv1.Session) *Session {
	if w == nil {
		return nil
	}
	s := &Session{UserAgent: w.GetUserAgent(), Pasted: w.GetPasted(), Earned: fromUnixMs(w.GetEarnedUnixMs())}
	for _, c := range w.GetCookies() {
		s.Cookies = append(s.Cookies, Cookie{
			Name: c.GetName(), Value: c.GetValue(), Domain: c.GetDomain(), Path: c.GetPath(),
			Expires: fromUnixMs(c.GetExpiresUnixMs()), Secure: c.GetSecure(), HostOnly: c.GetHostOnly(),
		})
	}
	return s
}

func encodeReading(r RSSReading) *searchv1.RSSReading {
	return &searchv1.RSSReading{
		Link: onWire(linkOnWire, r.Link), SizeFromDescription: r.SizeFromDescription,
		SeedersFromDescription: r.SeedersFromDescription, AllowNoSize: r.AllowNoSize,
	}
}

func decodeReading(w *searchv1.RSSReading) RSSReading {
	return RSSReading{
		Link: linkOnWire[w.GetLink()], SizeFromDescription: w.GetSizeFromDescription(),
		SeedersFromDescription: w.GetSeedersFromDescription(), AllowNoSize: w.GetAllowNoSize(),
	}
}

func unixMs(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

func fromUnixMs(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func int32s(ids []int) []int32 {
	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id) //nolint:gosec // a category id fits
	}
	return out
}

func ints(ids []int32) []int {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int, len(ids))
	for i, id := range ids {
		out[i] = int(id)
	}
	return out
}
