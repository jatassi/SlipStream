package site

import (
	"context"
	"crypto/sha1" //nolint:gosec // G505: a made-up upload's info hash, not a security use
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
)

// Feed sets how every Torznab site lists its newest uploads, the listing
// RSS sync reads, from now on. The arguments are FeedArgs, and {} lists
// them plainly again.
var Feed = fake.CueSpec{Name: "fixture-site-feed"}

// FeedArgs are the Feed cue's plays of the newest uploads listing, which
// is the world's releases newest first.
type FeedArgs struct {
	// NoPaging lists the newest page at every offset.
	NoPaging bool `json:"noPaging,omitempty"`
	// Late holds every listing past the longest search wait.
	Late bool `json:"late,omitempty"`
	// Uploads lists that many made-up uploads above the world's releases,
	// a second apart, the newest posted when the cue played.
	Uploads int `json:"uploads,omitempty"`
	// FutureDates dates every item a day from now.
	FutureDates bool `json:"futureDates,omitempty"`
	// LongGUIDs gives every item a GUID longer than SlipStream keeps.
	LongGUIDs bool `json:"longGUIDs,omitempty"`
	// OversizedPage lists more items on a page than SlipStream reads,
	// whatever the limit asked.
	OversizedPage bool `json:"oversizedPage,omitempty"`
}

const (
	// lateBy is past the longest search wait, 10 minutes.
	lateBy = 11 * time.Minute
	// oversized is one item past the longest page SlipStream reads.
	oversized = 1001
	// longGUID is past the longest GUID SlipStream keeps, 512 bytes.
	longGUID = 600
)

func (s *site) setFeed(_ context.Context, a FeedArgs) (any, error) {
	if a.Uploads < 0 {
		return nil, fmt.Errorf("uploads is %d, below 0", a.Uploads)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listing, s.listingAt = a, time.Now().UTC().Truncate(time.Second)
	return struct{}{}, nil
}

func (s *site) feedArgs() (FeedArgs, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listing, s.listingAt
}

// feedListing is how write plays one listing's GUIDs and dates.
type feedListing struct {
	longGUIDs bool
	// postedAt, when not zero, dates every item.
	postedAt time.Time
}

func (l feedListing) guid(link string) string {
	if l.longGUIDs {
		return link + "#" + strings.Repeat("g", longGUID)
	}
	return link
}

func (l feedListing) dated(rel Release) Release {
	if !l.postedAt.IsZero() {
		rel.Published = l.postedAt
	}
	return rel
}

// recent serves the newest uploads listing: the releases the request's
// categories ask for, newest first, in its window as the Feed cue plays
// it.
func (s *site) recent(w http.ResponseWriter, r *http.Request) {
	a, at := s.feedArgs()
	if a.Late {
		select {
		case <-time.After(lateBy):
		case <-r.Context().Done():
			return
		}
	}
	win := windowOf(r.URL.Query())
	label := labelOf(r)
	listed := made(a.Uploads, at)
	var world []Release
	for _, rel := range s.releases {
		if win.asks(rel) && s.shown(label, rel) {
			world = append(world, rel)
		}
	}
	slices.SortStableFunc(world, func(x, y Release) int { return postedOf(y).Compare(postedOf(x)) })
	listed = append(listed, world...)
	switch {
	case a.OversizedPage:
		listed = append(listed, made(max(0, oversized-len(listed)), at.Add(-time.Duration(len(listed))*time.Second))...)
	case a.NoPaging:
		listed = listed[:min(win.limit, len(listed))]
	default:
		listed = listed[min(win.offset, len(listed)):]
		listed = listed[:min(win.limit, len(listed))]
	}
	l := feedListing{longGUIDs: a.LongGUIDs}
	if a.FutureDates {
		l.postedAt = time.Now().UTC().Add(24 * time.Hour)
	}
	s.write(w, r, listed, l)
}

// postedOf is when rel says it was posted.
func postedOf(rel Release) time.Time {
	if rel.Published.IsZero() {
		t, _ := time.Parse(time.RFC1123Z, published)
		return t
	}
	return rel.Published
}

// made is n made-up movie uploads, newest first, the newest posted at
// newest and each older one a second before.
func made(n int, newest time.Time) []Release {
	out := make([]Release, n)
	for i := range out {
		at := newest.Add(-time.Duration(i) * time.Second)
		name := fmt.Sprintf("Fixture.Upload.%d.2026.1080p.WEB-DL", at.Unix())
		sum := sha1.Sum([]byte(name)) //nolint:gosec // G401: as above
		out[i] = Release{Name: name, Size: 1000, InfoHash: hex.EncodeToString(sum[:]), Seeders: 1, Untagged: true, Published: at}
	}
	return out
}
