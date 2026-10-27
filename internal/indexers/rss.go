package indexers

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	// ErrMedia is an RSS feed with no Content Type, or one SlipStream
	// doesn't have, or another kind that names Content Types.
	ErrMedia = errors.New("an RSS feed carries one or more of SlipStream's Content Types, and no other kind carries any")
	// ErrCookie is a pasted cookie that isn't name=value pairs.
	ErrCookie = errors.New("the cookie isn't name=value pairs separated by semicolons")
)

// PastedCookie is a browser cookie an Admin pasted for a feed, with the
// User-Agent of the browser it came from.
type PastedCookie struct {
	// Cookie is the Cookie header's value: name=value pairs separated by
	// semicolons.
	Cookie    string
	UserAgent string
}

// FeedRefusedError is an RSS feed whose Test found SlipStream can't use
// it.
type FeedRefusedError struct {
	Refusal  searching.RSSRefusal
	Sentence string
}

func (e *FeedRefusedError) Error() string { return "the feed can't be used: " + e.Refusal.Error() }

// checkMedia refuses idx with ErrMedia unless it is an RSS feed carrying
// one or more of types, or another kind carrying none.
func checkMedia(idx Indexer, types []works.TypeID) error {
	if idx.Kind != RSS {
		if idx.Media != nil {
			return ErrMedia
		}
		return nil
	}
	if len(idx.Media) == 0 || slices.ContainsFunc(idx.Media, func(t works.TypeID) bool { return !slices.Contains(types, t) }) {
		return ErrMedia
	}
	return nil
}

// pasted is c as a session on the Node idx is tested from, its cookies
// bound to the feed's host, nil when c is.
func pasted(c *PastedCookie, idx Indexer) (*searching.Session, error) {
	if c == nil {
		return nil, nil
	}
	cookies, err := http.ParseCookie(c.Cookie)
	if err != nil {
		return nil, ErrCookie
	}
	host := (&url.URL{Host: idx.FeedHost}).Hostname()
	s := &searching.Session{UserAgent: c.UserAgent, Pasted: true, Earned: time.Now()}
	for _, ck := range cookies {
		s.Cookies = append(s.Cookies, searching.Cookie{Name: ck.Name, Value: ck.Value, Domain: host, Path: "/", HostOnly: true})
	}
	return s, nil
}

// read is idx with the reading a's Test found, for an RSS feed.
func read(idx Indexer, a searching.Answer) Indexer {
	f, ok := idx.Source.(Feed)
	if !ok || idx.Kind != RSS || a.Report == nil || a.Report.RSS == nil {
		return idx
	}
	f.Reading = *a.Report.RSS
	idx.Source = f
	return idx
}

// refused is the FeedRefusedError of a Test that found an RSS feed
// SlipStream can't use, nil otherwise.
func refused(a searching.Answer) error {
	if a.Report == nil || a.Report.Refusal == "" {
		return nil
	}
	return &FeedRefusedError{Refusal: a.Report.Refusal, Sentence: refusalWords[a.Report.Refusal]}
}
