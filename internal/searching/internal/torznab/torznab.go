// Package torznab reads and writes Torznab: the api query a feed takes, its
// caps document, its results and its error element, with Prowlarr's
// defaults wherever an item leaves an attribute out.
package torznab

import (
	"cmp"
	"encoding/xml"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
)

// Query is one search as a feed takes it.
type Query struct {
	// Mode is the t= value: search, tvsearch or movie.
	Mode string
	// IDs are id parameters by name, such as tmdbid.
	IDs map[string]string
	// Text is q, left out when empty.
	Text       string
	Season     int
	Episode    string
	Categories []int
}

// Page is the api parameters of q's page from offset, limit items long,
// with every attribute the feed has.
func (q Query) Page(offset, limit int) url.Values {
	v := url.Values{"t": {q.Mode}, "extended": {"1"}, "offset": {strconv.Itoa(offset)}, "limit": {strconv.Itoa(limit)}}
	for name, id := range q.IDs {
		v.Set(name, id)
	}
	if q.Text != "" {
		v.Set("q", q.Text)
	}
	if q.Season > 0 {
		v.Set("season", strconv.Itoa(q.Season))
	}
	if q.Episode != "" {
		v.Set("ep", q.Episode)
	}
	if len(q.Categories) > 0 {
		cats := make([]string, len(q.Categories))
		for i, c := range q.Categories {
			cats[i] = strconv.Itoa(c)
		}
		v.Set("cat", strings.Join(cats, ","))
	}
	return v
}

// API is base's api address for params and key.
func API(base *url.URL, key string, params url.Values) *url.URL {
	u := base.JoinPath("api")
	q := url.Values{}
	for k, vs := range params {
		q[k] = vs
	}
	q.Set("apikey", key)
	u.RawQuery = q.Encode()
	return u
}

// Error is a Torznab error element: the feed understood the request and
// refused it.
type Error struct {
	Code        int
	Description string
}

type errorElement struct {
	XMLName     xml.Name
	Code        string `xml:"code,attr"`
	Description string `xml:"description,attr"`
}

// ErrorOf is the error element body is, false when body is no Torznab
// error.
func ErrorOf(body []byte) (Error, bool) {
	var e errorElement
	if xml.Unmarshal(body, &e) != nil || e.XMLName.Local != "error" {
		return Error{}, false
	}
	code, _ := strconv.Atoi(e.Code)
	return Error{Code: code, Description: e.Description}, true
}

// Caps are a feed's available search modes, keyed by their t= value, each
// with the parameters it takes in the order the feed lists them, and the
// categories it lists with their subcategories.
type Caps struct {
	Modes      map[string][]string
	Categories []int
}

type capsDocument struct {
	XMLName   xml.Name
	Searching struct {
		Modes []struct {
			XMLName         xml.Name
			Available       string `xml:"available,attr"`
			SupportedParams string `xml:"supportedParams,attr"`
		} `xml:",any"`
	} `xml:"searching"`
	Categories []struct {
		ID      string `xml:"id,attr"`
		Subcats []struct {
			ID string `xml:"id,attr"`
		} `xml:"subcat"`
	} `xml:"categories>category"`
}

// ParseCaps reads a t=caps document. A mode listing no parameters takes q,
// as Torznab defines.
func ParseCaps(body []byte) (Caps, error) {
	var doc capsDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return Caps{}, fmt.Errorf("read the caps: %w", err)
	}
	if doc.XMLName.Local != "caps" {
		return Caps{}, fmt.Errorf("read the caps: the document is <%s>, not <caps>", doc.XMLName.Local)
	}
	c := Caps{Modes: map[string][]string{}}
	for _, m := range doc.Searching.Modes {
		name, ok := Mode(m.XMLName.Local)
		if !ok || m.Available != "yes" {
			continue
		}
		params := []string{"q"}
		if m.SupportedParams != "" {
			params = strings.Split(m.SupportedParams, ",")
		}
		c.Modes[name] = params
	}
	for _, cat := range doc.Categories {
		c.Categories = appendID(c.Categories, cat.ID)
		for _, sub := range cat.Subcats {
			c.Categories = appendID(c.Categories, sub.ID)
		}
	}
	return c, nil
}

// Mode is the t= value of a caps search element: <x>-search is t=x, but
// for the generic search and tv-search, whose t= is tvsearch.
func Mode(element string) (string, bool) {
	switch element {
	case "search":
		return "search", true
	case "tv-search":
		return "tvsearch", true
	}
	name, ok := strings.CutSuffix(element, "-search")
	return name, ok && name != ""
}

// Element is the caps element of the search mode with t= value mode.
func Element(mode string) string {
	switch mode {
	case "search":
		return "search"
	case "tvsearch":
		return "tv-search"
	}
	return mode + "-search"
}

// Item is one result, normalized.
type Item struct {
	Title string
	// GUID is the item's guid, else its link.
	GUID      string
	Size      int64
	Published time.Time
	Seeders   int
	Leechers  int
	Grabs     int
	// Link is the first .torrent URL of the enclosure, the link and the
	// magneturl attr, empty when the item offers only magnet links.
	Link string
	// Magnet is the first magnet link of the same three.
	Magnet string
	// InfoHash is the infohash attr, else the first readable magnet link's.
	InfoHash       *bencode.InfoHash
	Categories     []int
	IDs            map[string]string
	DownloadFactor float64
	UploadFactor   float64
	MinRatio       float64
	MinSeed        time.Duration
	Details        string
	Source         *Source
}

// Source is the indexer an aggregating feed says an item came from.
type Source struct {
	ID      string
	Name    string
	Privacy string
	// Origin is "prowlarr" or "jackett".
	Origin string
}

type feed struct {
	Items []item `xml:"channel>item"`
}

type item struct {
	Title      string   `xml:"title"`
	GUID       string   `xml:"guid"`
	Link       string   `xml:"link"`
	Comments   string   `xml:"comments"`
	PubDate    string   `xml:"pubDate"`
	Size       string   `xml:"size"`
	Grabs      string   `xml:"grabs"`
	Type       string   `xml:"type"`
	Categories []string `xml:"category"`
	Enclosure  struct {
		URL    string `xml:"url,attr"`
		Length string `xml:"length,attr"`
	} `xml:"enclosure"`
	Attrs []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:"value,attr"`
	} `xml:"attr"`
	Prowlarr *struct {
		ID   string `xml:"id,attr"`
		Type string `xml:"type,attr"`
		Name string `xml:",chardata"`
	} `xml:"prowlarrindexer"`
	Jackett *struct {
		ID   string `xml:"id,attr"`
		Name string `xml:",chardata"`
	} `xml:"jackettindexer"`
}

// idAttrs are the id attributes an Item keeps.
var idAttrs = []string{"imdbid", "tmdbid", "tvdbid"}

// ParseFeed reads a results feed. An attribute that is missing or isn't a
// number takes Prowlarr's default: 1 for the volume factors, else none.
func ParseFeed(body []byte) ([]Item, error) {
	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("read the results: %w", err)
	}
	items := make([]Item, len(f.Items))
	for i, it := range f.Items {
		items[i] = it.parse()
	}
	return items, nil
}

func (it item) parse() Item {
	attrs := map[string]string{}
	var cats []int
	for _, c := range it.Categories {
		cats = appendID(cats, c)
	}
	for _, a := range it.Attrs {
		if a.Name == "category" {
			cats = appendID(cats, a.Value)
			continue
		}
		if _, seen := attrs[a.Name]; !seen {
			attrs[a.Name] = a.Value
		}
	}
	out := Item{
		Title: it.Title, GUID: cmp.Or(it.GUID, it.Link), Details: it.Comments, Categories: cats, IDs: map[string]string{},
		Size:           firstInt(attrs["size"], it.Size, it.Enclosure.Length),
		Published:      published(it.PubDate),
		Seeders:        int(firstInt(attrs["seeders"])),
		Grabs:          int(firstInt(attrs["grabs"], it.Grabs)),
		DownloadFactor: decimal(attrs["downloadvolumefactor"], 1),
		UploadFactor:   decimal(attrs["uploadvolumefactor"], 1),
		MinRatio:       decimal(attrs["minimumratio"], 0),
		MinSeed:        time.Duration(firstInt(attrs["minimumseedtime"])) * time.Second,
	}
	out.Leechers = int(firstInt(attrs["leechers"]))
	if peers, err := strconv.Atoi(attrs["peers"]); err == nil {
		out.Leechers = max(peers-out.Seeders, 0)
	}
	for _, name := range idAttrs {
		if v := attrs[name]; v != "" {
			out.IDs[name] = v
		}
	}
	out.Link, out.Magnet, out.InfoHash = offered([]string{it.Enclosure.URL, it.Link, attrs["magneturl"]}, attrs["infohash"])
	switch {
	case it.Prowlarr != nil:
		out.Source = &Source{ID: it.Prowlarr.ID, Name: strings.TrimSpace(it.Prowlarr.Name), Privacy: it.Prowlarr.Type, Origin: "prowlarr"}
	case it.Jackett != nil:
		out.Source = &Source{ID: it.Jackett.ID, Name: strings.TrimSpace(it.Jackett.Name), Privacy: it.Type, Origin: "jackett"}
	}
	return out
}

// offered is the .torrent link and the magnet link of those an item
// offers, in order of preference, and its info hash: hash, else the first
// readable magnet link's.
func offered(links []string, hash string) (link, magnetLink string, infoHash *bencode.InfoHash) {
	var magnets []string
	for _, l := range links {
		switch {
		case l == "":
		case magnet.Is(l):
			magnets = append(magnets, l)
		case link == "":
			link = l
		}
	}
	if h, err := bencode.ParseInfoHash(strings.ToLower(hash)); err == nil {
		infoHash = &h
	}
	for _, m := range magnets {
		if l, err := magnet.Parse(m); err == nil && infoHash == nil {
			infoHash = &l.InfoHash
		}
	}
	if len(magnets) > 0 {
		magnetLink = magnets[0]
	}
	return link, magnetLink, infoHash
}

func appendID(ids []int, s string) []int {
	id, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || slices.Contains(ids, id) {
		return ids
	}
	return append(ids, id)
}

func firstInt(values ...string) int64 {
	for _, v := range values {
		if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
			return n
		}
	}
	return 0
}

func decimal(v string, otherwise float64) float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
		return f
	}
	return otherwise
}

var pubDateLayouts = []string{time.RFC1123Z, time.RFC1123, "Mon, 2 Jan 2006 15:04:05 -0700", "2 Jan 2006 15:04:05 -0700", time.RFC3339}

func published(s string) time.Time {
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
