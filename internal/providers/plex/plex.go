// Package plex is the wire client for Plex Media Server: identity,
// sections and their folders, partial, section and item refreshes, items
// with their files, sessions and the notification socket. Every type here
// is the wire's, and the Plex kind translates. It asks for JSON on every
// call. Written against Plex Media Server 1.43.4.10903 (#71, #101) and
// 1.43.5.11029 (recorded run 2026-11-14), checked against python-plexapi.
package plex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Headers are the X-Plex-* headers that name SlipStream on every call.
type Headers struct {
	ClientID        string
	Product         string
	Version         string
	Platform        string
	PlatformVersion string
	Device          string
	DeviceName      string
}

func (h Headers) header() http.Header {
	out := http.Header{"Accept": {"application/json"}}
	for k, v := range map[string]string{
		"X-Plex-Client-Identifier": h.ClientID,
		"X-Plex-Product":           h.Product,
		"X-Plex-Version":           h.Version,
		"X-Plex-Platform":          h.Platform,
		"X-Plex-Platform-Version":  h.PlatformVersion,
		"X-Plex-Device":            h.Device,
		"X-Plex-Device-Name":       h.DeviceName,
	} {
		if v != "" {
			out.Set(k, v)
		}
	}
	return out
}

// Client is one Plex Media Server at base, with the server's own token.
type Client struct {
	base  *url.URL
	token string
	h     Headers
	http  *http.Client
}

func New(base *url.URL, token string, h Headers, c *http.Client) *Client {
	return &Client{base: base, token: token, h: h, http: c}
}

// Error is a response other than 2xx.
type Error struct {
	Status int
	// Detail is the start of the response body.
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("plex responded with status %d: %s", e.Status, e.Detail)
}

// Refused is whether err is the server refusing the token, or its absence.
func Refused(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusUnauthorized
}

// NotFound is whether err is a 404.
func NotFound(err error) bool {
	e, ok := errors.AsType[*Error](err)
	return ok && e.Status == http.StatusNotFound
}

// ID is a key Plex sends as a number on some routes and versions and as a
// string on others.
type ID string

func (id *ID) UnmarshalJSON(b []byte) error {
	if bytes.HasPrefix(b, []byte(`"`)) {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err //nolint:wrapcheck // the decoder names the field
		}
		*id = ID(s)
		return nil
	}
	*id = ID(b)
	return nil
}

func (c *Client) url(path string, q url.Values) string {
	u := c.base.JoinPath(path)
	u.RawQuery = q.Encode()
	return u.String()
}

// do sends one request. With token false it carries no token, as Reach
// does. A 2xx response's JSON body decodes into out when out isn't nil.
func (c *Client) do(ctx context.Context, method, path string, q url.Values, token bool, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.url(path, q), nil)
	if err != nil {
		return fmt.Errorf("plex %s %s: %w", method, path, err)
	}
	req.Header = c.h.header()
	if token && c.token != "" {
		req.Header.Set("X-Plex-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("plex %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("plex %s %s: read the response: %w", method, path, err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("plex %s %s: %w", method, path, &Error{Status: resp.StatusCode, Detail: detail(body)})
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("plex %s %s: decode the response: %w", method, path, err)
	}
	return nil
}

func detail(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// Identity is GET /identity, which needs no token (recorded run
// 2026-11-14: 200 while the token was revoked).
type Identity struct {
	MachineIdentifier string `json:"machineIdentifier"`
	Version           string `json:"version"`
	Claimed           bool   `json:"claimed"`
}

func (c *Client) Identity(ctx context.Context) (Identity, error) {
	var out struct{ MediaContainer Identity }
	err := c.do(ctx, http.MethodGet, "/identity", nil, false, &out)
	return out.MediaContainer, err
}

// Server is GET /, which needs the token.
type Server struct {
	FriendlyName       string `json:"friendlyName"`
	MachineIdentifier  string `json:"machineIdentifier"`
	Version            string `json:"version"`
	MyPlexSubscription bool   `json:"myPlexSubscription"`
}

func (c *Client) Server(ctx context.Context) (Server, error) {
	var out struct{ MediaContainer Server }
	err := c.do(ctx, http.MethodGet, "/", nil, true, &out)
	return out.MediaContainer, err
}

// Section is one library section and its folders.
type Section struct {
	Key        ID         `json:"key"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	UUID       string     `json:"uuid"`
	Refreshing bool       `json:"refreshing"`
	ScannedAt  int64      `json:"scannedAt"`
	Locations  []Location `json:"Location"`
}

// Location is a section folder as the server's OS names it.
type Location struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

func (c *Client) Sections(ctx context.Context) ([]Section, error) {
	var out struct {
		MediaContainer struct {
			Directory []Section `json:"Directory"`
		}
	}
	err := c.do(ctx, http.MethodGet, "/library/sections", nil, true, &out)
	return out.MediaContainer.Directory, err
}

// Section is one section by its key, false when the server holds none.
func (c *Client) Section(ctx context.Context, key ID) (Section, bool, error) {
	all, err := c.Sections(ctx)
	if err != nil {
		return Section{}, false, err
	}
	for _, s := range all {
		if s.Key == key {
			return s, true, nil
		}
	}
	return Section{}, false, nil
}

// RefreshPath is a partial scan of folder, as Plex stores it, inside one
// of section's Locations. Plex documents POST and every client sends GET,
// which Plex answers 200.
func (c *Client) RefreshPath(ctx context.Context, section ID, folder string) error {
	return c.do(ctx, http.MethodGet, "/library/sections/"+string(section)+"/refresh", url.Values{"path": {folder}}, true, nil)
}

// RefreshSection is a scan of the whole section.
func (c *Client) RefreshSection(ctx context.Context, section ID) error {
	return c.do(ctx, http.MethodGet, "/library/sections/"+string(section)+"/refresh", nil, true, nil)
}

// RefreshItem reads one item's media and Sidecars again.
func (c *Client) RefreshItem(ctx context.Context, ratingKey ID) error {
	return c.do(ctx, http.MethodPut, "/library/metadata/"+string(ratingKey)+"/refresh", nil, true, nil)
}

// Metadata is one item.
type Metadata struct {
	RatingKey            ID           `json:"ratingKey"`
	Key                  string       `json:"key"`
	GUID                 string       `json:"guid"`
	Type                 string       `json:"type"`
	Title                string       `json:"title"`
	Year                 int          `json:"year"`
	LibrarySectionID     ID           `json:"librarySectionID"`
	GrandparentRatingKey ID           `json:"grandparentRatingKey"`
	GrandparentTitle     string       `json:"grandparentTitle"`
	ParentIndex          int          `json:"parentIndex"`
	Index                int          `json:"index"`
	ViewCount            int          `json:"viewCount"`
	ViewOffset           int64        `json:"viewOffset"`
	LastViewedAt         int64        `json:"lastViewedAt"`
	UserRating           float64      `json:"userRating"`
	EditionTitle         string       `json:"editionTitle"`
	Collections          []Tag        `json:"Collection"`
	Media                []Media      `json:"Media"`
	User                 *User        `json:"User"`
	Player               *Player      `json:"Player"`
	Session              *SessionInfo `json:"Session"`
	SessionKey           ID           `json:"sessionKey"`
	TranscodeSession     *Transcode   `json:"TranscodeSession"`
}

type Tag struct {
	Tag string `json:"tag"`
}

// Media is one version of an item.
type Media struct {
	ID    ID     `json:"id"`
	Parts []Part `json:"Part"`
}

// Part is one file of a version.
type Part struct {
	ID       ID       `json:"id"`
	File     string   `json:"file"`
	Size     int64    `json:"size"`
	Decision string   `json:"decision"`
	Streams  []Stream `json:"Stream"`
}

// StreamSubtitle is a subtitle stream's streamType.
const StreamSubtitle = 3

// Stream is one stream of a part. An external Sidecar has a key.
type Stream struct {
	ID         ID     `json:"id"`
	StreamType int    `json:"streamType"`
	Codec      string `json:"codec"`
	Key        string `json:"key"`
}

// Item is one item by its ratingKey, false on 404: a moved file's old item
// answers 404 with an HTML body (recorded run 2026-11-14).
func (c *Client) Item(ctx context.Context, ratingKey ID) (Metadata, bool, error) {
	var out struct {
		MediaContainer struct {
			Metadata []Metadata `json:"Metadata"`
		}
	}
	err := c.do(ctx, http.MethodGet, "/library/metadata/"+string(ratingKey), nil, true, &out)
	if NotFound(err) {
		return Metadata{}, false, nil
	}
	if err != nil {
		return Metadata{}, false, err
	}
	if len(out.MediaContainer.Metadata) == 0 {
		return Metadata{}, false, nil
	}
	return out.MediaContainer.Metadata[0], true, nil
}

// ItemType is the type filter of a section listing.
type ItemType int

const (
	TypeMovie   ItemType = 1
	TypeEpisode ItemType = 4
)

// Page is one page of a section's items.
type Page struct {
	Items []Metadata
	Total int
}

// SectionPage lists size items of typ from start, with their files in
// Media[].Part[].file. Plex has no lookup by path.
func (c *Client) SectionPage(ctx context.Context, section ID, typ ItemType, start, size int) (Page, error) {
	var out struct {
		MediaContainer struct {
			TotalSize int        `json:"totalSize"`
			Metadata  []Metadata `json:"Metadata"`
		}
	}
	q := url.Values{
		"type":                   {strconv.Itoa(int(typ))},
		"X-Plex-Container-Start": {strconv.Itoa(start)},
		"X-Plex-Container-Size":  {strconv.Itoa(size)},
	}
	err := c.do(ctx, http.MethodGet, "/library/sections/"+string(section)+"/all", q, true, &out)
	return Page{Items: out.MediaContainer.Metadata, Total: out.MediaContainer.TotalSize}, err
}
